package upload

import (
	"context"
	"errors"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool     *pgxpool.Pool
	reg      *format.Registry
	store    storage.Store
	settings work.IngestSettings
	assets   *work.Service
	now      func() time.Time
}

func NewService(pool *pgxpool.Pool, assets *work.Service) *Service {
	return &Service{
		pool: pool, reg: assets.Registry(), store: assets.Store(),
		settings: assets.IngestSettings(), assets: assets, now: assets.Now,
	}
}

// writeSummary rewrites the work's summary row inside the transaction that changed it
func (s *Service) writeSummary(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	if err := summary.Write(ctx, tx, s.reg, assetID); err != nil {
		if errors.Is(err, summary.ErrNotFound) {
			return work.ErrNotFound
		}
		return err
	}
	return nil
}
