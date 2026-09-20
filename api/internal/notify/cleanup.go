package notify

import (
	"context"
	"fmt"
	"time"
)

const (
	Retention    = 90 * 24 * time.Hour
	CleanupEvery = time.Hour
	cleanupBatch = 1000
)

// RunCleanup removes expired entries every hour until the context ends.
func (s *Service) RunCleanup(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(CleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Cleanup(ctx, s.now()); err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
		}
	}
}

// Cleanup removes every entry that arrived more than the retention period before the given time.
func (s *Service) Cleanup(ctx context.Context, now time.Time) (int64, error) {
	var deleted int64
	for {
		tag, err := s.pool.Exec(ctx, `
			delete from notifications
			 where id in (select id from notifications where created_at < $1 limit $2)
		`, now.Add(-Retention), cleanupBatch)
		if err != nil {
			return deleted, fmt.Errorf("clean up expired notifications: %w", err)
		}
		deleted += tag.RowsAffected()
		if tag.RowsAffected() < cleanupBatch {
			return deleted, nil
		}
	}
}
