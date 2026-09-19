package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Announcement reports a version sent to an integration
type Announcement struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	UpdateID      uuid.UUID
	UpdateNumber  int
	Destination   string
	Type          string
	Removed       bool
	State         string
	SettledReason string
	MessageID     string
	Run           int
	Attempts      int
	OccurredAt    time.Time
	DueAt         time.Time
	SettledAt     *time.Time
	Last          *dispatch.Attempt
}

// Announcements lists a work's sent versions, newest first
func (s *Service) Announcements(ctx context.Context, owner, workID uuid.UUID) ([]Announcement, error) {
	var owned bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from works where id = $1 and owner_id = $2 and deleted_at is null)
	`, workID, owner).Scan(&owned)
	if err != nil {
		return nil, fmt.Errorf("read who owns the announced work: %w", err)
	}
	if !owned {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		select work.id, event.id, event.version_id, version.number,
		       work.destination_name, work.destination_type, work.destination_id is null,
		       work.state, coalesce(work.settled_reason, ''), coalesce(work.message_id, ''),
		       work.run, work.attempts, event.occurred_at, work.due_at, work.settled_at,
		       last.run, last.number, last.outcome, last.status, last.detail,
		       last.took_ms, last.attempted_at
		  from work_update_deliveries work
		  join work_update_events event on event.id = work.event_id
		  join work_versions version on version.id = event.version_id
		  left join lateral (
			select run, number, outcome, status, detail, took_ms, attempted_at
			  from work_update_delivery_attempts
			 where delivery_id = work.id
			 order by number desc
			 limit 1
		  ) last on true
		 where event.work_id = $1
		 order by event.occurred_at desc, work.destination_name, work.id
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("read what a work announced: %w", err)
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
			&one.Destination, &one.Type, &one.Removed,
			&one.State, &one.SettledReason, &one.MessageID,
			&one.Run, &one.Attempts, &one.OccurredAt, &one.DueAt, &one.SettledAt,
			&run, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing a work announced: %w", err)
		}
		if number != nil {
			one.Last = &dispatch.Attempt{
				Run: *run, Number: *number, Outcome: *outcome, Status: status, Detail: *detail,
				Took: time.Duration(*took) * time.Millisecond, Attempted: *attempted,
			}
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what a work announced: %w", err)
	}
	return found, nil
}
