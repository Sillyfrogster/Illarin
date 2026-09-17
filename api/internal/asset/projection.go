package asset

import (
	"context"
	"errors"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) WriteProjections(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	return missingAsset(summary.Write(ctx, tx, s.reg, assetID))
}

func (s *Service) writeFacetProjection(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	return missingAsset(summary.WriteFilters(ctx, tx, assetID))
}

func (s *Service) exportCapability(ctx context.Context, q db.DBTX, assetID uuid.UUID) ([]format.Target, error) {
	targets, err := summary.Formats(ctx, q, s.reg, assetID)
	return targets, missingAsset(err)
}

func missingAsset(err error) error {
	if errors.Is(err, summary.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
