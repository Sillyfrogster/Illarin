package version

import (
	"context"
	"errors"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool   *pgxpool.Pool
	reg    *format.Registry
	assets *asset.Service
}

func NewService(pool *pgxpool.Pool, assets *asset.Service) *Service {
	return &Service{pool: pool, reg: assets.Registry(), assets: assets}
}

func (s *Service) writeSummary(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	err := summary.Write(ctx, tx, s.reg, workID)
	if errors.Is(err, summary.ErrNotFound) {
		return asset.ErrNotFound
	}
	return err
}
