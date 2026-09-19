package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const EventUpdatePublished = "work.update.published.v1"

type sent struct {
	ID         uuid.UUID  `json:"id"`
	Type       string     `json:"type"`
	OccurredAt time.Time  `json:"occurredAt"`
	Work       sentWork   `json:"work"`
	Update     sentUpdate `json:"update"`
}

type sentWork struct {
	ID   uuid.UUID `json:"id"`
	Type string    `json:"type"`
	Name string    `json:"name"`
	URL  string    `json:"url"`
}

type sentUpdate struct {
	ID             uuid.UUID `json:"id"`
	Number         int       `json:"number"`
	VersionLabel   string    `json:"versionLabel,omitempty"`
	Summary        string    `json:"summary"`
	RecordedAt     time.Time `json:"recordedAt"`
	ContentChanged bool      `json:"contentChanged"`
	HistoryURL     string    `json:"historyUrl"`
}

type announced struct {
	owner           uuid.UUID
	name            string
	destinationType string
	visibility      string
}

// Announce queues a published version for its chosen integrations
func (s *Service) Announce(
	ctx context.Context,
	tx pgx.Tx,
	published version.Version,
	choice version.Announcement,
) error {
	var held announced
	err := tx.QueryRow(ctx, `
		select owner_id, name, type, visibility from works where id = $1
	`, published.WorkID).Scan(&held.owner, &held.name, &held.destinationType, &held.visibility)
	if err != nil {
		return fmt.Errorf("read the work being announced: %w", err)
	}
	unlisted := held.visibility == string(work.VisibilityUnlisted)
	picked, err := s.pick(ctx, tx, held.owner, published.WorkID, unlisted, choice)
	if err != nil || len(picked) == 0 {
		return err
	}
	eventID := uuid.New()
	occurred := s.now().UTC()
	body, err := json.Marshal(sent{
		ID: eventID, Type: EventUpdatePublished, OccurredAt: occurred,
		Work: sentWork{
			ID: published.WorkID, Type: held.destinationType, Name: held.name,
			URL: s.workAddress(published.WorkID),
		},
		Update: sentUpdate{
			ID: published.ID, Number: published.Number, VersionLabel: published.VersionLabel,
			Summary: published.Summary, RecordedAt: published.RecordedAt,
			ContentChanged: published.ContentChanged,
			HistoryURL:     s.historyAddress(published.WorkID, published.Number),
		},
	})
	if err != nil {
		return fmt.Errorf("write the update announcement: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into work_update_events
		       (id, work_id, version_id, type, occurred_at, unlisted_consent, payload)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, eventID, published.WorkID, published.ID, EventUpdatePublished, occurred,
		choice.AnnounceUnlisted, body)
	if err != nil {
		return fmt.Errorf("record the update announcement: %w", err)
	}
	for _, one := range picked {
		_, err := tx.Exec(ctx, `
			insert into work_update_deliveries
			       (id, event_id, destination_id, destination_name, destination_type, due_at)
			values ($1, $2, $3, $4, $5, $6)
		`, uuid.New(), eventID, one.ID, one.Name, one.Type, occurred)
		if err != nil {
			return fmt.Errorf("keep the announcement work: %w", err)
		}
	}
	return nil
}

type picked struct {
	ID   uuid.UUID
	Name string
	Type string
}

func (s *Service) pick(
	ctx context.Context,
	tx pgx.Tx,
	owner, workID uuid.UUID,
	unlisted bool,
	choice version.Announcement,
) ([]picked, error) {
	if choice.DestinationIDs == nil {
		if unlisted {
			return nil, nil
		}
		return collectPicked(tx.Query(ctx, `
			select destination.id, destination.name, destination.type
			  from work_update_destination_defaults chosen
			  join work_update_destinations destination on destination.id = chosen.destination_id
			 where chosen.work_id = $1 and destination.owner_id = $2 and destination.state = $3
			 order by destination.name, destination.id
			 for share of destination
		`, workID, owner, Active))
	}
	wanted := distinct(*choice.DestinationIDs)
	if len(wanted) > 0 && unlisted && !choice.AnnounceUnlisted {
		return nil, version.ErrUnlistedConsentRequired
	}
	found, err := collectPicked(tx.Query(ctx, `
		select id, name, type from work_update_destinations
		 where owner_id = $1 and id = any($2::uuid[]) and state = $3
		 order by name, id
		 for share
	`, owner, wanted, Active))
	if err != nil {
		return nil, err
	}
	if len(found) != len(wanted) {
		return nil, version.ErrDestinationIneligible
	}
	if err := remember(ctx, tx, workID, wanted); err != nil {
		return nil, err
	}
	return found, nil
}

func remember(ctx context.Context, tx pgx.Tx, workID uuid.UUID, ids []uuid.UUID) error {
	_, err := tx.Exec(ctx, `delete from work_update_destination_defaults where work_id = $1`, workID)
	if err != nil {
		return fmt.Errorf("forget the remembered destinations: %w", err)
	}
	for _, id := range ids {
		_, err := tx.Exec(ctx, `
			insert into work_update_destination_defaults (work_id, destination_id) values ($1, $2)
		`, workID, id)
		if err != nil {
			return fmt.Errorf("remember a destination: %w", err)
		}
	}
	return nil
}

func distinct(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(ids))
	kept := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		kept = append(kept, id)
	}
	return kept
}

func collectPicked(rows pgx.Rows, err error) ([]picked, error) {
	if err != nil {
		return nil, fmt.Errorf("read the destinations an update goes to: %w", err)
	}
	defer rows.Close()
	found := make([]picked, 0, 4)
	for rows.Next() {
		var one picked
		if err := rows.Scan(&one.ID, &one.Name, &one.Type); err != nil {
			return nil, fmt.Errorf("read one destination an update goes to: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the destinations an update goes to: %w", err)
	}
	return found, nil
}

func (s *Service) workAddress(workID uuid.UUID) string {
	return s.site + "/a/" + workID.String()
}

func (s *Service) historyAddress(workID uuid.UUID, number int) string {
	return s.workAddress(workID) + "/history#version-" + strconv.Itoa(number)
}
