package notification

import (
	"context"
	"fmt"
	"time"
)

const (
	Retention  = 90 * 24 * time.Hour
	SweepEvery = time.Hour
	sweepBatch = 1000
)

// RunSweeper removes expired entries every hour until the context ends.
func (s *Service) RunSweeper(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(SweepEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Sweep(ctx, s.now()); err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
		}
	}
}

// Sweep removes every entry that arrived more than the retention period before the given time.
func (s *Service) Sweep(ctx context.Context, now time.Time) (int64, error) {
	var swept int64
	for {
		tag, err := s.pool.Exec(ctx, `
			delete from notifications
			 where id in (select id from notifications where created_at < $1 limit $2)
		`, now.Add(-Retention), sweepBatch)
		if err != nil {
			return swept, fmt.Errorf("sweep expired notifications: %w", err)
		}
		swept += tag.RowsAffected()
		if tag.RowsAffected() < sweepBatch {
			return swept, nil
		}
	}
}
