package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	FanOutEvery = 5 * time.Second
	fanOutBatch = 200
)

// Service fans recorded events out to inboxes and answers the inbox queries.
type Service struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

// RunFanOut turns recorded events into inbox entries until the context ends.
func (s *Service) RunFanOut(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(FanOutEvery)
	defer ticker.Stop()
	for {
		if _, err := s.FanOut(ctx, s.now()); err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// FanOut turns every event recorded by the given time into inbox entries and reports how many it handled.
func (s *Service) FanOut(ctx context.Context, now time.Time) (int, error) {
	handled := 0
	for {
		batch, err := s.fanOutBatch(ctx, now)
		handled += batch
		if err != nil || batch < fanOutBatch {
			return handled, err
		}
	}
}

func (s *Service) fanOutBatch(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin a notification fan-out: %w", err)
	}
	defer tx.Rollback(ctx)
	due, err := takeDueEvents(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	for _, event := range due {
		if err := writeEntries(ctx, tx, event); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit a notification fan-out: %w", err)
	}
	return len(due), nil
}

// recorded is one event the fan-out has taken off the queue.
type recorded struct {
	Type       Type
	Account    *uuid.UUID
	Work       *uuid.UUID
	Words      []byte
	RecordedAt time.Time
}

// takeDueEvents claims a batch of recorded events for this pass alone.
func takeDueEvents(ctx context.Context, tx pgx.Tx, now time.Time) ([]recorded, error) {
	rows, err := tx.Query(ctx, `
		delete from notification_events
		 where id in (
			select id from notification_events
			 where recorded_at <= $1
			 order by recorded_at, id
			 for update skip locked
			 limit $2
		 )
		returning type, account_id, work_id, words, recorded_at
	`, now, fanOutBatch)
	if err != nil {
		return nil, fmt.Errorf("take notification events off the queue: %w", err)
	}
	defer rows.Close()
	due := make([]recorded, 0, fanOutBatch)
	for rows.Next() {
		var event recorded
		if err := rows.Scan(
			&event.Type, &event.Account, &event.Work, &event.Words, &event.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("read a notification event: %w", err)
		}
		due = append(due, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("take notification events off the queue: %w", err)
	}
	return due, nil
}

// writeEntries gives one event to everyone it is for, folding repeated updates together.
func writeEntries(ctx context.Context, tx pgx.Tx, event recorded) error {
	if _, err := tx.Exec(ctx, `
		insert into notifications (id, account_id, type, work_id, words, created_at)
		select gen_random_uuid(), hearer.account_id, $1, $2, $3, $4
		  from (
			select $5::uuid where $5::uuid is not null
			union all
			select account_id from (
				select account_id from work_follows
				 where work_id = $2 and state = 'following'
				union
				select app.user_id
				  from app_library_entries entry
				  join connected_apps app on app.id = entry.connected_app_id
				 where entry.work_id = $2 and app.revoked_at is null
				except
				select account_id from work_follows
				 where work_id = $2 and state = 'stopped'
				except
				select owner_id from works where id = $2
			) following
			 where $5::uuid is null
		  ) hearer (account_id)
		on conflict (account_id, work_id) where type = 'work_updated' and read_at is null
		do update set words = excluded.words,
		              created_at = excluded.created_at,
		              update_count = notifications.update_count + 1
	`, event.Type, event.Work, event.Words, event.RecordedAt, event.Account); err != nil {
		return fmt.Errorf("write the inbox entries for a notification event: %w", err)
	}
	return nil
}
