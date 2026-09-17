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

var ErrNotFound = errors.New("asset not found")

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
		return fmt.Errorf("write the export projection: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into asset_projections (asset_id, export, export_stamp, export_computed_at)
		values ($1, $2, $3, now())
		on conflict (asset_id) do update
		   set export = excluded.export,
		       export_stamp = excluded.export_stamp,
		       export_computed_at = excluded.export_computed_at
	`, workID, stored, reg.CapabilityStamp()); err != nil {
		return fmt.Errorf("store the export projection: %w", err)
	}
	return nil
}

// Formats works out the formats a work can be downloaded in from its drafted blocks
func Formats(ctx context.Context, q db.DBTX, reg *format.Registry, workID uuid.UUID) ([]format.Target, error) {
	var kind string
	var origin pgtype.Text
	err := q.QueryRow(ctx, `
		select kind, origin_format from assets where id = $1 and deleted_at is null
	`, workID).Scan(&kind, &origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read the asset to project: %w", err)
	}
	blocks, err := block.Read(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		elements = append(elements, holder.Elements...)
	}
	return reg.OfferedTargets(format.CapabilitySubject{
		Kind: kind, Origin: origin.String, Elements: elements,
	}), nil
}

func Offered(ctx context.Context, q db.DBTX, workID uuid.UUID) ([]format.Target, error) {
	var stored []byte
	err := q.QueryRow(ctx,
		`select export from asset_projections where asset_id = $1`, workID,
	).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return []format.Target{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the export projection: %w", err)
	}
	targets := make([]format.Target, 0)
	if err := json.Unmarshal(stored, &targets); err != nil {
		return nil, fmt.Errorf("read the stored export projection: %w", err)
	}
	return targets, nil
}

func RecomputeStaleFormats(ctx context.Context, pool *pgxpool.Pool, reg *format.Registry) (int, error) {
	stale, err := staleWorks(ctx, pool, `
		select asset.id
		  from assets asset
		  left join asset_projections projection on projection.asset_id = asset.id
		 where asset.deleted_at is null
		   and (projection.asset_id is null or projection.export_stamp <> $1)
		 order by asset.id
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
			return 0, fmt.Errorf("recompute the export projection for %s: %w", workID, err)
		}
	}
	return len(stale), nil
}

func staleWorks(ctx context.Context, pool *pgxpool.Pool, query string, stamp string) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, query, stamp)
	if err != nil {
		return nil, fmt.Errorf("find stale projections: %w", err)
	}
	defer rows.Close()
	stale := make([]uuid.UUID, 0)
	for rows.Next() {
		var workID uuid.UUID
		if err := rows.Scan(&workID); err != nil {
			return nil, fmt.Errorf("read a stale projection: %w", err)
		}
		stale = append(stale, workID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find stale projections: %w", err)
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
