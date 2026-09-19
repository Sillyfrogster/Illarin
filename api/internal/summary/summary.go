package summary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("work not found")

// Write rewrites both halves of a work's summary row inside the transaction that changed the work
func Write(ctx context.Context, tx pgx.Tx, reg *format.Registry, workID uuid.UUID) error {
	if err := writeFormats(ctx, tx, reg, workID); err != nil {
		return err
	}
	if err := WriteFilters(ctx, tx, workID); err != nil {
		return err
	}
	return writePublished(ctx, tx, reg, workID)
}

func writeFormats(ctx context.Context, tx pgx.Tx, reg *format.Registry, workID uuid.UUID) error {
	targets, err := Formats(ctx, tx, reg, workID)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("write the export summary: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into work_summaries (work_id, export, export_stamp, export_computed_at)
		values ($1, $2, $3, now())
		on conflict (work_id) do update
		   set export = excluded.export,
		       export_stamp = excluded.export_stamp,
		       export_computed_at = excluded.export_computed_at
	`, workID, stored, reg.CapabilityStamp()); err != nil {
		return fmt.Errorf("store the export summary: %w", err)
	}
	return nil
}

// Formats works out the formats a work can be downloaded in from its drafted blocks
func Formats(ctx context.Context, q db.DBTX, reg *format.Registry, workID uuid.UUID) ([]format.Offered, error) {
	var workType string
	var origin pgtype.Text
	err := q.QueryRow(ctx, `
		select type, origin_format from works where id = $1 and deleted_at is null
	`, workID).Scan(&workType, &origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read the work to project: %w", err)
	}
	blocks, err := block.Read(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		elements = append(elements, holder.Elements...)
	}
	return reg.OfferedFormats(format.CapabilitySubject{
		Type: workType, Origin: origin.String, Elements: elements,
	}), nil
}

func Offered(ctx context.Context, q db.DBTX, workID uuid.UUID) ([]format.Offered, error) {
	var stored []byte
	err := q.QueryRow(ctx,
		`select export from work_summaries where work_id = $1`, workID,
	).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return []format.Offered{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the export summary: %w", err)
	}
	targets := make([]format.Offered, 0)
	if err := json.Unmarshal(stored, &targets); err != nil {
		return nil, fmt.Errorf("read the stored export summary: %w", err)
	}
	return targets, nil
}

func RecomputeStaleFormats(ctx context.Context, pool *pgxpool.Pool, reg *format.Registry) (int, error) {
	stale, err := staleWorks(ctx, pool, `
		select work.id
		  from works work
		  left join work_summaries summary on summary.work_id = work.id
		 where work.deleted_at is null
		   and (summary.work_id is null or summary.export_stamp <> $1)
		 order by work.id
	`, reg.CapabilityStamp())
	if err != nil {
		return 0, err
	}
	for _, workID := range stale {
		if err := inTransaction(ctx, pool, func(tx pgx.Tx) error {
			if err := writeFormats(ctx, tx, reg, workID); err != nil {
				return err
			}
			return writePublished(ctx, tx, reg, workID)
		}); err != nil {
			return 0, fmt.Errorf("recompute the export summary for %s: %w", workID, err)
		}
	}
	return len(stale), nil
}

func staleWorks(ctx context.Context, pool *pgxpool.Pool, query string, stamp string) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, query, stamp)
	if err != nil {
		return nil, fmt.Errorf("find stale summaries: %w", err)
	}
	defer rows.Close()
	stale := make([]uuid.UUID, 0)
	for rows.Next() {
		var workID uuid.UUID
		if err := rows.Scan(&workID); err != nil {
			return nil, fmt.Errorf("read a stale summary: %w", err)
		}
		stale = append(stale, workID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find stale summaries: %w", err)
	}
	return stale, nil
}

func inTransaction(ctx context.Context, pool *pgxpool.Pool, run func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := run(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
