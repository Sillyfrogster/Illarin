package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Attempt is one announcement queued for one integration
type Attempt struct {
	ID             uuid.UUID
	AnnouncementID uuid.UUID
	VersionID      uuid.UUID
	VersionNumber  int
	Integration    string
	Type           string
	Removed        bool
	State          string
	SettledReason  string
	MessageID      string
	Run            int
	Tries          int
	OccurredAt     time.Time
	DueAt          time.Time
	SettledAt      *time.Time
	Last           *dispatch.Try
}

// Attempts lists what a work announced, newest first
func (s *Service) Attempts(ctx context.Context, owner, workID uuid.UUID) ([]Attempt, error) {
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
		       work.integration_name, work.integration_type, work.integration_id is null,
		       work.state, coalesce(work.settled_reason, ''), coalesce(work.message_id, ''),
		       work.run, work.tries, event.occurred_at, work.due_at, work.settled_at,
		       last.run, last.number, last.outcome, last.status, last.detail,
		       last.took_ms, last.attempted_at
		  from work_announcement_attempts work
		  join work_announcements event on event.id = work.announcement_id
		  join work_versions version on version.id = event.version_id
		  left join lateral (
			select run, number, outcome, status, detail, took_ms, attempted_at
			  from work_announcement_tries
			 where attempt_id = work.id
			 order by number desc
			 limit 1
		  ) last on true
		 where event.work_id = $1
		 order by event.occurred_at desc, work.integration_name, work.id
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("read what a work announced: %w", err)
	}
	return collectAttempts(rows)
}

func collectAttempts(rows pgx.Rows) ([]Attempt, error) {
	defer rows.Close()
	found := make([]Attempt, 0, 4)
	for rows.Next() {
		var one Attempt
		var run, number, status, took *int
		var outcome, detail *string
		var attempted *time.Time
		err := rows.Scan(
			&one.ID, &one.AnnouncementID, &one.VersionID, &one.VersionNumber,
			&one.Integration, &one.Type, &one.Removed,
			&one.State, &one.SettledReason, &one.MessageID,
			&one.Run, &one.Tries, &one.OccurredAt, &one.DueAt, &one.SettledAt,
			&run, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing a work announced: %w", err)
		}
		if number != nil {
			one.Last = &dispatch.Try{
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
