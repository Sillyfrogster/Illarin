package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
)

func (s *Sends) RunSweeper(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(s.settings.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Sweep(ctx); err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
		}
	}
}

func (s *Sends) Sweep(ctx context.Context) (int64, error) {
	swept, err := db.New(s.pool).DeleteExpiredDeliveries(ctx, sweepBatch)
	if err != nil {
		return 0, fmt.Errorf("sweep expired deliveries: %w", err)
	}
	return swept, nil
}

const sweepBatch = 200
