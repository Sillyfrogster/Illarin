package blog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
)

type DiscordRepair struct {
	RequestID uuid.UUID `json:"requestId"`
	Action    string    `json:"action"`
	MessageID string    `json:"messageId"`
	Text      string    `json:"text"`
}

type DiscordRepairResult struct {
	Action    string `json:"action"`
	MessageID string `json:"messageId"`
	State     string `json:"state"`
	Detail    string `json:"detail"`
}

type discordRepairSender interface {
	Request(context.Context, string, string, []byte) (dispatch.Answer, error)
}

// RepairDiscord repairs an announcement and records the action
func (s *Service) RepairDiscord(ctx context.Context, actor, id uuid.UUID, in DiscordRepair) (DiscordRepairResult, error) {
	in.Text = strings.TrimSpace(in.Text)
	in.MessageID = strings.TrimSpace(in.MessageID)
	invalid := func(message string) (DiscordRepairResult, error) {
		return DiscordRepairResult{}, FieldError{Field: "repair", Message: message}
	}
	if in.RequestID == uuid.Nil || (in.Action != "edit" && in.Action != "delete" && in.Action != "correction") {
		return invalid("Choose edit, delete or correction and supply a new request ID.")
	}
	if in.Action != "delete" && (in.Text == "" || utf8.RuneCountInString(in.Text) > 1800) {
		return invalid("Write the replacement note or correction in 1 to 1800 characters.")
	}
	held, err := s.Delivery(ctx, id)
	if err != nil {
		return DiscordRepairResult{}, err
	}
	if held.Kind != KindDiscord || held.Removed {
		return invalid("This delivery has no Discord destination to repair.")
	}
	if held.SettledAt == nil {
		return DiscordRepairResult{}, ErrDeliveryUnsettled
	}
	if in.Action != "correction" {
		if err := discord.CheckRole(in.MessageID); err != nil {
			return invalid("Name the Discord message ID to edit or delete.")
		}
		if held.MessageID != "" && held.MessageID != in.MessageID {
			return invalid("The message ID does not match this announcement.")
		}
	}
	var destinationID uuid.UUID
	if err := s.pool.QueryRow(ctx, `select destination_id from publication_deliveries where id = $1`, id).Scan(&destinationID); err != nil {
		return DiscordRepairResult{}, err
	}
	_, state, err := s.destinationStanding(ctx, destinationID)
	if err != nil {
		return DiscordRepairResult{}, err
	}
	if state != DestinationActive {
		return DiscordRepairResult{}, ErrDeliveryUnsendable
	}
	capability, _, err := s.channelOf(ctx, destinationID)
	if err != nil {
		return DiscordRepairResult{}, err
	}
	method, address := http.MethodPost, capability.Confirming()
	var body []byte
	if in.Action == "delete" {
		method, address = http.MethodDelete, capability.URL+"/messages/"+in.MessageID
	} else {
		summary, err := s.Summary(ctx, s.pool, held.EventID)
		if err != nil {
			return DiscordRepairResult{}, err
		}
		announcement := announcementOf(summary, "")
		announcement.Note = in.Text
		if in.Action == "edit" {
			method, address = http.MethodPatch, capability.URL+"/messages/"+in.MessageID
		} else {
			announcement.Note = "Correction: " + in.Text
		}
		body, err = announcement.Body()
		if err != nil {
			return DiscordRepairResult{}, err
		}
	}
	sender, ok := s.sender.(discordRepairSender)
	if !ok {
		return invalid("Discord repair is unavailable with this sender.")
	}
	result := DiscordRepairResult{Action: in.Action, MessageID: in.MessageID, State: "unconfirmed",
		Detail: "The repair may have reached Discord. Check the channel before another action; another correction could create a duplicate."}
	if in.Action == "correction" {
		result.MessageID = ""
	}
	encoded, _ := json.Marshal(in)
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(encoded))
	stored, _ := json.Marshal(result)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DiscordRepairResult{}, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		insert into publication_audits (id, actor_id, credential, action, post_id, destination_id, delivery_id)
		values ($1, $2, $3, $4, $5, $6, $7) on conflict (id) do nothing
	`, in.RequestID, actor, CredentialSession, "discord."+in.Action, held.PostID, destinationID, id)
	if err != nil {
		return DiscordRepairResult{}, fmt.Errorf("record the Discord repair: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var previous string
		err := tx.QueryRow(ctx, `select result::text from publication_discord_repairs
			where id = $1 and actor_id = $2 and delivery_id = $3 and fingerprint = $4`,
			in.RequestID, actor, id, fingerprint).Scan(&previous)
		if err != nil {
			return invalid("Use a new request ID for a different repair action.")
		}
		if err := json.Unmarshal([]byte(previous), &result); err != nil {
			return DiscordRepairResult{}, err
		}
		return result, nil
	}
	_, err = tx.Exec(ctx, `insert into publication_discord_repairs
		(id, actor_id, delivery_id, target_message_id, fingerprint, result)
		values ($1, $2, $3, $4, $5, $6)`, in.RequestID, actor, id, in.MessageID, fingerprint, string(stored))
	if err != nil {
		return DiscordRepairResult{}, fmt.Errorf("record the Discord repair request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return DiscordRepairResult{}, err
	}
	answer, sendErr := sender.Request(ctx, method, address, body)
	if sendErr == nil {
		var fault struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(answer.Body, &fault)
		if answer.Status == http.StatusNotFound && (in.Action == "correction" || fault.Code == 10015) {
			if err := s.retireDestination(ctx, destinationID); err != nil {
				return DiscordRepairResult{}, err
			}
		}
		switch {
		case answer.Status >= 200 && answer.Status < 300:
			if in.Action == "delete" {
				result.State, result.Detail = "completed", "The named Discord message was deleted. Copies already received by readers remain outside Illarin's control."
			} else if message := dispatch.MessageID(answer.Body); message != "" {
				result.State, result.MessageID, result.Detail = "completed", message, "Discord confirmed the requested change. The blog post and delivery history are unchanged."
			}
		case answer.Status == http.StatusTooManyRequests:
			result.State, result.Detail = "refused", "Discord asked Illarin to wait. Try a new repair after the rate limit clears."
		case answer.Status >= 400 && answer.Status < 500:
			result.State, result.Detail = "refused", fmt.Sprintf("Discord refused the repair (%d). Check that the message and webhook still exist.", answer.Status)
		}
	}
	stored, _ = json.Marshal(result)
	_, err = s.pool.Exec(ctx, `update publication_discord_repairs set result = $2 where id = $1`, in.RequestID, string(stored))
	if err != nil {
		return DiscordRepairResult{}, fmt.Errorf("record the Discord repair outcome: %w", err)
	}
	return result, nil
}
