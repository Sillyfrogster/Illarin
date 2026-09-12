package asset

import (
	"context"
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) writePublishedProjections(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	var published bool
	if err := tx.QueryRow(ctx, `select published_snapshot_id is not null from public.assets where id = $1`, assetID).Scan(&published); err != nil {
		return err
	}
	if !published {
		return nil
	}
	if _, err := tx.Exec(ctx, `set local search_path = asset_public, public`); err != nil {
		return err
	}
	targets, err := s.exportCapability(ctx, tx, assetID)
	if err != nil {
		return err
	}
	facets, err := facetCounts(ctx, tx, assetID)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(map[string]any{
		"asset_id": assetID, "export": targets, "export_stamp": s.reg.CapabilityStamp(),
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
		on conflict (snapshot_id) do update set projection = excluded.projection`, assetID, stored)
	return err
}
