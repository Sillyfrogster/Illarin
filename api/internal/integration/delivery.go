package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	SettledWithheld  = "withheld"
	SettledWithdrawn = "withdrawn"
	SettledUnlisted  = "unlisted"
	SettledDeleted   = "deleted"
)

var whyCancelled = map[string]string{
	SettledWithheld:  "The work is withheld.",
	SettledWithdrawn: "The update was withdrawn.",
	SettledUnlisted:  "The work is unlisted and this update was not cleared to send its link.",
	SettledDeleted:   "The work is no longer published.",
}

// RunAnnouncements sends due announcements until the context ends.
func (s *Service) RunAnnouncements(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(dispatch.Poll)
	defer ticker.Stop()
	for {
		_, err := s.SendDueAnnouncements(ctx, s.now())
		if err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SendDueAnnouncements makes one attempt on every announcement due at the given time.
func (s *Service) SendDueAnnouncements(ctx context.Context, now time.Time) (int, error) {
	made := 0
	for {
		held, taken, err := s.ledger.Lease(ctx, now)
		if err != nil || !taken {
			return made, err
		}
		if err := s.sendLeased(ctx, held, now); err != nil {
			return made, err
		}
		made++
	}
}

type standing struct {
	unpublished     bool
	withheld        bool
	unlisted        bool
	withdrawn       bool
	consent         bool
	payload         []byte
	destinationType string
	state           string
	sameOwner       bool
	hasDestination  bool
}

func (s *Service) sendLeased(ctx context.Context, held dispatch.Work, now time.Time) error {
	found, err := s.recheck(ctx, held)
	if err != nil {
		return err
	}
	if said, cancelled := found.cancels(); cancelled {
		return s.ledger.Record(ctx, held, said, now)
	}
	if found.destinationType == Discord {
		return s.announceOnDiscord(ctx, held, *held.DestinationID, found.payload, now)
	}
	endpoint, err := s.configurationByID(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	headers, err := dispatch.Headers(endpoint.secrets, held.ID.String(), now.UTC(), found.payload)
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, endpoint.address, headers, found.payload)
	if err != nil {
		return s.ledger.Record(ctx, held, dispatch.Unreachable, now)
	}
	said := dispatch.ReadAnswer(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if err := s.ledger.Record(ctx, held, said, now); err != nil {
		return err
	}
	if !said.Gone {
		return nil
	}
	return s.retire(ctx, *held.DestinationID)
}

func (s *Service) recheck(ctx context.Context, held dispatch.Work) (standing, error) {
	var found standing
	var destinationType, state *string
	var sameOwner *bool
	err := s.pool.QueryRow(ctx, `
		select owned.deleted_at is not null or owned.lifecycle <> 'published',
		       owned.withheld_at is not null, owned.visibility = 'unlisted',
		       version.withdrawn_at is not null, event.unlisted_consent, event.payload::text,
		       destination.type, destination.state, destination.owner_id = owned.owner_id
		  from work_update_events event
		  join works owned on owned.id = event.work_id
		  join work_versions version on version.id = event.version_id
		  left join work_update_destinations destination on destination.id = $2
		 where event.id = $1
	`, held.EventID, held.DestinationID).Scan(
		&found.unpublished, &found.withheld, &found.unlisted, &found.withdrawn, &found.consent,
		&found.payload, &destinationType, &state, &sameOwner,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		found.unpublished = true
		return found, nil
	}
	if err != nil {
		return found, fmt.Errorf("recheck an announcement before sending it: %w", err)
	}
	if destinationType != nil && state != nil && sameOwner != nil {
		found.hasDestination, found.destinationType, found.state, found.sameOwner = true, *destinationType, *state, *sameOwner
	}
	return found, nil
}

func (f standing) cancels() (dispatch.Verdict, bool) {
	switch {
	case f.unpublished:
		return cancelled(SettledDeleted), true
	case f.withheld:
		return cancelled(SettledWithheld), true
	case f.withdrawn:
		return cancelled(SettledWithdrawn), true
	case f.unlisted && !f.consent:
		return cancelled(SettledUnlisted), true
	case !f.hasDestination || !f.sameOwner:
		return dispatch.Stopped(dispatch.Removed), true
	case f.state != Active:
		return dispatch.Stopped(dispatch.Disabled), true
	}
	return dispatch.Verdict{}, false
}

func cancelled(reason string) dispatch.Verdict {
	return dispatch.Cancelled(reason, whyCancelled[reason])
}

func (s *Service) announceOnDiscord(
	ctx context.Context,
	held dispatch.Work,
	destinationID uuid.UUID,
	payload []byte,
	now time.Time,
) error {
	if held.Reclaimed {
		return s.ledger.Record(ctx, held, dispatch.DiscordUnconfirmed, now)
	}
	var summary sent
	if err := json.Unmarshal(payload, &summary); err != nil {
		return fmt.Errorf("read the announcement to render: %w", err)
	}
	endpoint, err := s.configurationByID(ctx, destinationID)
	if err != nil {
		return err
	}
	capability, err := discord.ReadCapability(endpoint.address)
	if err != nil {
		return fmt.Errorf("read the Discord capability: %w", err)
	}
	body, err := noticeOf(summary).Body()
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, capability.Confirming(), nil, body)
	if err != nil {
		return s.ledger.Record(ctx, held, dispatch.DiscordUnconfirmed, now)
	}
	said, message := dispatch.ReadAnnouncement(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if message != "" {
		if err := s.ledger.KeepMessage(ctx, held.ID, message); err != nil {
			return err
		}
	}
	if err := s.ledger.Record(ctx, held, said, now); err != nil {
		return err
	}
	if said.Gone {
		return s.retire(ctx, destinationID)
	}
	return nil
}

func noticeOf(said sent) discord.Announcement {
	update := fmt.Sprintf("#%d", said.Update.Number)
	if said.Update.VersionLabel != "" {
		update = said.Update.VersionLabel
	}
	return discord.Announcement{
		Title:   said.Work.Name,
		Summary: said.Update.Summary,
		URL:     said.Update.HistoryURL,
		Update:  update,
		At:      said.OccurredAt,
		Footer:  discord.Site,
	}
}

func (s *Service) configurationByID(ctx context.Context, id uuid.UUID) (configuration, error) {
	var owner uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select owner_id from work_update_destinations where id = $1
	`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return configuration{}, ErrNotFound
	}
	if err != nil {
		return configuration{}, err
	}
	return s.configuration(ctx, owner, id)
}

func (s *Service) retire(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin retiring a destination: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update work_update_destinations
		   set state = $2, disabled_at = now(), version = version + 1, updated_at = now()
		 where id = $1 and state <> $2
	`, id, Disabled)
	if err != nil {
		return fmt.Errorf("retire the destination: %w", err)
	}
	if err := s.ledger.StopTo(ctx, tx, id, dispatch.Stopped(dispatch.Gone)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit retiring a destination: %w", err)
	}
	return nil
}
