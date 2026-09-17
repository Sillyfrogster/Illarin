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
	"github.com/jackc/pgx/v5/pgxpool"
)

func WriteFilters(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	counts, err := filterCounts(ctx, tx, workID)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(counts)
	if err != nil {
		return fmt.Errorf("write the facet projection: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into asset_projections (asset_id, facets, facet_stamp, facet_computed_at)
		values ($1, $2, $3, now())
		on conflict (asset_id) do update
		   set facets = excluded.facets,
		       facet_stamp = excluded.facet_stamp,
		       facet_computed_at = excluded.facet_computed_at
	`, workID, stored, block.FacetStamp()); err != nil {
		return fmt.Errorf("store the facet projection: %w", err)
	}
	return nil
}

func filterCounts(ctx context.Context, q db.DBTX, workID uuid.UUID) (map[block.FacetKey]int, error) {
	var kind string
	err := q.QueryRow(ctx, `
		select kind from assets where id = $1 and deleted_at is null
	`, workID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read the asset to measure: %w", err)
	}
	blocks, err := block.Read(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	return block.MeasureFacets(kind, shownElements(blocks)), nil
}

func shownElements(blocks []block.Block) []block.Element {
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		if holder.Hidden {
			continue
		}
		elements = append(elements, holder.Elements...)
	}
	return elements
}

func RecomputeStaleFilters(ctx context.Context, pool *pgxpool.Pool, reg *format.Registry) (int, error) {
	stale, err := staleWorks(ctx, pool, `
		select asset.id
		  from assets asset
		  left join asset_projections projection on projection.asset_id = asset.id
		 where asset.deleted_at is null
		   and (projection.asset_id is null or projection.facet_stamp <> $1)
		 order by asset.id
	`, block.FacetStamp())
	if err != nil {
		return 0, err
	}
	for _, workID := range stale {
		if err := inTransaction(ctx, pool, func(tx pgx.Tx) error {
			if err := WriteFilters(ctx, tx, workID); err != nil {
				return err
			}
			return writePublished(ctx, tx, reg, workID)
		}); err != nil {
			return 0, fmt.Errorf("recompute the facet projection for %s: %w", workID, err)
		}
	}
	return len(stale), nil
}
