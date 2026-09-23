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
	pool  *pgxpool.Pool
	reg   *format.Registry
	works *work.Service
}

func NewService(pool *pgxpool.Pool, works *work.Service) *Service {
	return &Service{pool: pool, reg: works.Registry(), works: works}
}

type SavedBlock struct {
	Type  string
	Block block.Block
}

type SavedBlocks struct {
	Type   string
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

func validatePublishedWidths(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	rows, err := tx.Query(ctx, `select layout, width from work_public.work_blocks where work_id = $1`, workID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var layout block.Layout
		var width block.Width
		if err := rows.Scan(&layout, &width); err != nil {
			return err
		}
		if width.Columns() < layout.MinimumWidth().Columns() {
			return fmt.Errorf("%w: publish the new layout before narrowing this block", block.ErrInvalid)
		}
	}
	return rows.Err()
}

func deleteBlockAndClosePositions(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	blockID uuid.UUID,
	remaining []block.Block,
) error {
	if _, err := tx.Exec(ctx, `delete from work_blocks where id = $1 and work_id = $2`, blockID, workID); err != nil {
		return fmt.Errorf("remove block: %w", err)
	}
	for position := range remaining {
		if _, err := tx.Exec(ctx, `
			update work_blocks set position = $3 where id = $1 and work_id = $2
		`, remaining[position].ID, workID, position); err != nil {
			return fmt.Errorf("close block positions: %w", err)
		}
	}
	return nil
}
