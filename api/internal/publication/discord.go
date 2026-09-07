package publication

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const roleNameLimit = 48

// ErrNotDiscord says an operation meant for one kind of destination was aimed
// at the other.
var ErrNotDiscord = errors.New("the destination is not a Discord channel")

// SettledUnconfirmed is the reason on an announcement Discord accepted without
// confirming, which is neither a success nor a failure.
const SettledUnconfirmed = "unconfirmed"

// ChannelEdit is what the authority supplies to add or change a Discord
// destination. An empty role id takes the approved role away.
type ChannelEdit struct {
	Name     string
	Address  string
	RoleID   string
	RoleName string
}

// AddChannel records one Discord channel, having asked Discord itself what the
// capability address is for. It is active from the moment Discord answers,
// because that answer is the proof a challenge would otherwise have to be.
func (s *Service) AddChannel(
	ctx context.Context,
	actor uuid.UUID,
	in ChannelEdit,
) (Destination, error) {
	name := checkDestinationName(in.Name)
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Destination{}, err
	}
	capability, found, err := s.askDiscord(ctx, in.Address)
	if err != nil {
		return Destination{}, err
	}
	sealed, err := s.sealing.Seal([]byte(capability.URL))
	if err != nil {
		return Destination{}, fmt.Errorf("seal the Discord capability: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord destination: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into publication_destinations
		       (id, kind, name, host, address, events, state, verified_at,
		        guild_id, channel_id, webhook_name, role_id, role_name, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, now(), $8, $9, $10, $11, $12, $13)
	`, id, KindDiscord, name, discordHost, sealed, []string{EventPublished}, DestinationActive,
		found.GuildID, found.ChannelID, found.Name, role.id, role.name, actor)
	if err != nil {
		return Destination{}, fmt.Errorf("record the Discord destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.added", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord destination: %w", err)
	}
	return s.Destination(ctx, id)
}

// UpdateChannel renames a Discord destination, changes its approved role or
// points it at another capability, which Discord is asked about again.
func (s *Service) UpdateChannel(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in ChannelEdit,
) (Destination, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if current.Kind != KindDiscord {
		return Destination{}, ErrNotDiscord
	}
	name := checkDestinationName(in.Name)
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Destination{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord destination update: %w", err)
	}
	defer tx.Rollback(ctx)
	if strings.TrimSpace(in.Address) == "" {
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, role_id = $3, role_name = $4, updated_at = now()
			 where id = $1
		`, id, name, role.id, role.name)
	} else {
		var capability discord.Capability
		var found discord.Webhook
		if capability, found, err = s.askDiscord(ctx, in.Address); err != nil {
			return Destination{}, err
		}
		var sealed []byte
		if sealed, err = s.sealing.Seal([]byte(capability.URL)); err != nil {
			return Destination{}, fmt.Errorf("seal the Discord capability: %w", err)
		}
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, address = $3, guild_id = $4, channel_id = $5, webhook_name = $6,
			       role_id = $7, role_name = $8, state = $9, verified_at = now(),
			       disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, sealed, found.GuildID, found.ChannelID, found.Name,
			role.id, role.name, DestinationActive)
	}
	if err != nil {
		return Destination{}, fmt.Errorf("update the Discord destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.updated", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord destination update: %w", err)
	}
	return s.Destination(ctx, id)
}

// provenByDiscord asks Discord about a channel's stored capability again and
// puts the destination back in service when Discord still recognizes it.
func (s *Service) provenByDiscord(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Destination, error) {
	capability, _, err := s.channelOf(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	_, found, err := s.askDiscord(ctx, capability.URL)
	if err != nil {
		return Destination{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, guild_id = $3, channel_id = $4, webhook_name = $5,
		       verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, DestinationActive, found.GuildID, found.ChannelID, found.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("put the Discord destination back in service: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.verified", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord verification: %w", err)
	}
	return s.Destination(ctx, id)
}

// discordHost is the host every Discord capability answers on, held apart from
// the sealed address so a masked view can name it.
const discordHost = "discord.com"

// askDiscord reads the capability and has Discord confirm what it points at,
// which is the whole of what proves a Discord destination.
func (s *Service) askDiscord(
	ctx context.Context,
	address string,
) (discord.Capability, discord.Webhook, error) {
	capability, err := discord.ReadCapability(address)
	if err != nil {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field: "address", Message: capitalize(err.Error()) + ".", cause: err,
		}
	}
	answer, err := s.sender.Get(ctx, capability.URL)
	if err != nil {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field: "address", Message: "Illarin could not reach Discord.", cause: err,
		}
	}
	if answer.Status != http.StatusOK {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field:   "address",
			Message: fmt.Sprintf("Discord answered %d for that webhook.", answer.Status),
			cause:   discord.ErrNotAWebhook,
		}
	}
	found, err := discord.ReadWebhook(answer.Body)
	if err != nil {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field:   "address",
			Message: "That address is not a webhook on a Discord channel.",
			cause:   err,
		}
	}
	return capability, found, nil
}

// approved is the one role a Discord destination may name, held as the pair of
// nulls the row keeps when there is none.
type approved struct {
	id   *string
	name *string
}

func checkRole(roleID, roleName string) (approved, error) {
	id := strings.TrimSpace(roleID)
	name := strings.TrimSpace(roleName)
	if id == "" && name == "" {
		return approved{}, nil
	}
	if err := discord.CheckRole(id); err != nil {
		return approved{}, FieldError{
			Field: "roleId", Message: capitalize(err.Error()) + ".", cause: err,
		}
	}
	if name == "" || len(name) > roleNameLimit {
		return approved{}, FieldError{
			Field:   "roleName",
			Message: fmt.Sprintf("Name the role in %d characters or fewer.", roleNameLimit),
		}
	}
	return approved{id: &id, name: &name}, nil
}

// announceOnDiscord composes the one announcement Illarin decided on and sends
// it in confirming mode, so an accepted request either carries a message id or
// is recorded as one Discord never confirmed.
func (s *Service) announceOnDiscord(
	ctx context.Context,
	held waiting,
	destinationID uuid.UUID,
	now time.Time,
) error {
	summary, err := s.summary(ctx, held.EventID)
	if err != nil {
		return err
	}
	capability, role, err := s.channelOf(ctx, destinationID)
	if err != nil {
		return err
	}
	if !held.Ping {
		role = ""
	}
	body, err := announcementOf(summary, role).Body()
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, capability.Confirming(), nil, body)
	if err != nil {
		return s.record(ctx, held, unreachable, now)
	}
	said := readAnnouncement(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if said.Message != "" {
		if err := s.keepMessage(ctx, held.ID, said.Message); err != nil {
			return err
		}
	}
	return s.record(ctx, held, said.verdict, now)
}

// announced is what Discord said, alongside the id of the message it made.
type announced struct {
	verdict
	Message string
}

// readAnnouncement turns Discord's answer into what Illarin does next. An
// accepted request with no message id is never called a success, because a
// second attempt at it could post the announcement twice.
func readAnnouncement(answer outbound.Answer) announced {
	if answer.Status < http.StatusOK || answer.Status >= http.StatusMultipleChoices {
		return announced{verdict: readAnswer(answer)}
	}
	message := discord.MessageID(answer.Body)
	if message == "" {
		return announced{verdict: verdict{
			Outcome: AttemptUnconfirmed,
			Detail:  "Discord took it without saying which message it made.",
			Reason:  SettledUnconfirmed,
		}}
	}
	return announced{verdict: arrived, Message: message}
}

// announcementOf turns the summary an event carries into the fixed shape
// Illarin announces in.
func announcementOf(said sent, role string) discord.Announcement {
	one := discord.Announcement{
		Title:    said.Post.Title,
		Summary:  said.Post.Summary,
		URL:      said.Post.URL,
		Image:    said.Post.SocialImage,
		Category: said.Post.Category.Label,
		Note:     said.Note,
		Role:     role,
		Author:   discord.Author{Name: said.Post.Byline.Name, URL: said.Post.Byline.URL},
		At:       said.Post.PublishedAt,
	}
	if said.Post.Release != nil {
		one.Version = said.Post.Release.Version
	}
	return one
}

// channelOf opens the capability one Discord destination was configured with
// and the role it may name. They leave this package only as a request already
// made.
func (s *Service) channelOf(
	ctx context.Context,
	id uuid.UUID,
) (discord.Capability, string, error) {
	var sealed []byte
	var role *string
	err := s.pool.QueryRow(ctx, `
		select address, role_id from publication_destinations where id = $1
	`, id).Scan(&sealed, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return discord.Capability{}, "", ErrDestinationNotFound
	}
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("read the Discord configuration: %w", err)
	}
	address, err := s.sealing.Open(sealed)
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("open the Discord capability: %w", err)
	}
	capability, err := discord.ReadCapability(string(address))
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("read the Discord capability: %w", err)
	}
	if role == nil {
		return capability, "", nil
	}
	return capability, *role, nil
}

// keepMessage holds the id of the announcement Discord made, which is the only
// handle a deliberate repair ever has on it.
func (s *Service) keepMessage(ctx context.Context, deliveryID uuid.UUID, message string) error {
	_, err := s.pool.Exec(ctx, `
		update publication_deliveries set message_id = $2 where id = $1
	`, deliveryID, message)
	if err != nil {
		return fmt.Errorf("keep the announcement's message id: %w", err)
	}
	return nil
}
