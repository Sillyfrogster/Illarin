package summary

import (
	"context"
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func writePublished(ctx context.Context, tx pgx.Tx, reg *format.Registry, workID uuid.UUID) error {
	var published bool
	if err := tx.QueryRow(ctx, `select published_snapshot_id is not null from public.assets where id = $1`, workID).Scan(&published); err != nil {
		return err
	}
	if !published {
		return nil
	}
	if _, err := tx.Exec(ctx, `set local search_path = asset_public, public`); err != nil {
		return err
	}
	targets, err := Formats(ctx, tx, reg, workID)
	if err != nil {
		return err
	}
	facets, err := filterCounts(ctx, tx, workID)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(map[string]any{
		"asset_id": workID, "export": targets, "export_stamp": reg.CapabilityStamp(),
		"facets": facets, "facet_stamp": block.FacetStamp(),
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `set local search_path = public`); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into asset_snapshot_projections (snapshot_id, projection)
		select published_snapshot_id, $2::jsonb || jsonb_build_object(
		    'export_computed_at', now(), 'facet_computed_at', now())
		from assets where id = $1
		on conflict (snapshot_id) do update set projection = excluded.projection`, workID, stored)
	return err
}
