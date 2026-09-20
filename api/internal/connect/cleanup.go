package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
)

func (s *Sends) RunCleanup(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(s.settings.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Cleanup(ctx); err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
		}
	}
}

func (s *Sends) Cleanup(ctx context.Context) (int64, error) {
	deleted, err := db.New(s.pool).DeleteExpiredSends(ctx, sendCleanupBatch)
	if err != nil {
		return 0, fmt.Errorf("clean up expired sends: %w", err)
	}
	return deleted, nil
}

const sendCleanupBatch = 200
