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
		return fmt.Errorf("write the facet summary: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		insert into work_summaries (work_id, facets, facet_stamp, facet_computed_at)
		values ($1, $2, $3, now())
		on conflict (work_id) do update
		   set facets = excluded.facets,
		       facet_stamp = excluded.facet_stamp,
		       facet_computed_at = excluded.facet_computed_at
	`, workID, stored, block.FacetStamp()); err != nil {
		return fmt.Errorf("store the facet summary: %w", err)
	}
	if _, err := tx.Exec(ctx, `set local search_path = work_public, public`); err != nil {
		return err
	}
	counts, err = filterCounts(ctx, tx, workID)
	if err != nil {
		return err
	}
	stored, err = json.Marshal(counts)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `set local search_path = public`); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `update work_version_summaries set summary = summary ||
		jsonb_build_object('facets', $2::jsonb, 'facet_stamp', $3::text, 'facet_computed_at', now())
		where version_id = (select published_version_id from works where id = $1)`, workID, stored, block.FacetStamp())
	if err != nil {
		return err
	}
	return nil
}

func filterCounts(ctx context.Context, q db.DBTX, workID uuid.UUID) (map[block.FacetKey]int, error) {
	var workType string
	err := q.QueryRow(ctx, `
		select type from works where id = $1 and deleted_at is null
	`, workID).Scan(&workType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read the work to measure: %w", err)
	}
	blocks, err := block.Read(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	return block.MeasureFacets(workType, shownElements(blocks)), nil
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
		select work.id
		  from works work
		  left join work_summaries summary on summary.work_id = work.id
		 where work.deleted_at is null
		   and (summary.work_id is null or summary.facet_stamp <> $1)
		 order by work.id
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
			return 0, fmt.Errorf("recompute the facet summary for %s: %w", workID, err)
		}
	}
	return len(stale), nil
}
