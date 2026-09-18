package version

import (
	"context"
	"errors"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool            *pgxpool.Pool
	reg             *format.Registry
	assets          *work.Service
	updateListeners []UpdateListener
}

func NewService(pool *pgxpool.Pool, assets *work.Service) *Service {
	return &Service{pool: pool, reg: assets.Registry(), assets: assets}
}

func (s *Service) writeSummary(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	err := summary.Write(ctx, tx, s.reg, workID)
	if errors.Is(err, summary.ErrNotFound) {
		return work.ErrNotFound
	}
	return err
}
