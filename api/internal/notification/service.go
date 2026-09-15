package notification

import (
	"context"
	"fmt"
	"time"

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
	var handled int
	if err := tx.QueryRow(ctx, `
		with due as (
			select id, type, account_id, asset_id, words, recorded_at
			  from notification_events
			 where recorded_at <= $1
			 order by recorded_at, id
			 for update skip locked
			 limit $2
		), handled as (
			delete from notification_events event
			 using due
			 where event.id = due.id
			returning event.id
		), written as (
			insert into notifications (id, account_id, type, asset_id, words, created_at)
			select gen_random_uuid(), account_id, type, asset_id, words, recorded_at
			  from due
		)
		select count(*) from handled
	`, now, fanOutBatch).Scan(&handled); err != nil {
		return 0, fmt.Errorf("fan out notification events: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit a notification fan-out: %w", err)
	}
	return handled, nil
}
