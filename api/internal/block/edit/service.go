package edit

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool   *pgxpool.Pool
	reg    *format.Registry
	assets *work.Service
}

func NewService(pool *pgxpool.Pool, assets *work.Service) *Service {
	return &Service{pool: pool, reg: assets.Registry(), assets: assets}
}

type SavedBlock struct {
	Kind  string
	Block block.Block
}

type SavedBlocks struct {
	Kind   string
	Blocks []block.Block
}

func invalid(err error) error {
	return fmt.Errorf("%w: %v", block.ErrInvalid, err)
}

func (s *Service) writeSummary(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	return missingWork(summary.Write(ctx, tx, s.reg, workID))
}

func missingWork(err error) error {
	if errors.Is(err, summary.ErrNotFound) {
		return work.ErrNotFound
	}
	return err
}

func deleteBlockAndClosePositions(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	blockID uuid.UUID,
	remaining []block.Block,
) error {
	if _, err := tx.Exec(ctx, `delete from asset_blocks where id = $1 and asset_id = $2`, blockID, workID); err != nil {
		return fmt.Errorf("remove block: %w", err)
	}
	for position := range remaining {
		if _, err := tx.Exec(ctx, `
			update asset_blocks set position = $3 where id = $1 and asset_id = $2
		`, remaining[position].ID, workID, position); err != nil {
			return fmt.Errorf("close block positions: %w", err)
		}
	}
	return nil
}
