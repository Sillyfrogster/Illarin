package assetdestination

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Announcement is what the owner sees of one update on its way to one destination.
type Announcement struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	UpdateID      uuid.UUID
	UpdateNumber  int
	Destination   string
	Kind          string
	Removed       bool
	State         string
	SettledReason string
	MessageID     string
	Run           int
	Attempts      int
	OccurredAt    time.Time
	DueAt         time.Time
	SettledAt     *time.Time
	Last          *outbox.Attempt
}

// Announcements lists what an asset's updates have sent, newest update first.
func (s *Service) Announcements(ctx context.Context, owner, assetID uuid.UUID) ([]Announcement, error) {
	var owned bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from assets where id = $1 and owner_id = $2 and deleted_at is null)
	`, assetID, owner).Scan(&owned)
	if err != nil {
		return nil, fmt.Errorf("read who owns the announced asset: %w", err)
	}
	if !owned {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		select work.id, event.id, event.snapshot_id, snapshot.number,
		       work.destination_name, work.destination_kind, work.destination_id is null,
		       work.state, coalesce(work.settled_reason, ''), coalesce(work.message_id, ''),
		       work.run, work.attempts, event.occurred_at, work.due_at, work.settled_at,
		       last.run, last.number, last.outcome, last.status, last.detail,
		       last.took_ms, last.attempted_at
		  from asset_update_deliveries work
		  join asset_update_events event on event.id = work.event_id
		  join asset_snapshots snapshot on snapshot.id = event.snapshot_id
		  left join lateral (
			select run, number, outcome, status, detail, took_ms, attempted_at
			  from asset_update_delivery_attempts
			 where delivery_id = work.id
			 order by number desc
			 limit 1
		  ) last on true
		 where event.asset_id = $1
		 order by event.occurred_at desc, work.destination_name, work.id
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("read what an asset announced: %w", err)
	}
	return collectAnnouncements(rows)
}

func collectAnnouncements(rows pgx.Rows) ([]Announcement, error) {
	defer rows.Close()
	found := make([]Announcement, 0, 4)
	for rows.Next() {
		var one Announcement
		var run, number, status, took *int
		var outcome, detail *string
		var attempted *time.Time
		err := rows.Scan(
			&one.ID, &one.EventID, &one.UpdateID, &one.UpdateNumber,
			&one.Destination, &one.Kind, &one.Removed,
			&one.State, &one.SettledReason, &one.MessageID,
			&one.Run, &one.Attempts, &one.OccurredAt, &one.DueAt, &one.SettledAt,
			&run, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing an asset announced: %w", err)
		}
		if number != nil {
			one.Last = &outbox.Attempt{
				Run: *run, Number: *number, Outcome: *outcome, Status: status, Detail: *detail,
				Took: time.Duration(*took) * time.Millisecond, Attempted: *attempted,
			}
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what an asset announced: %w", err)
	}
	return found, nil
}
