package work

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidDiscovery = errors.New("invalid discovery state")
	ErrAlreadyPublished = errors.New("the asset is already published")
)

func (s *Service) SetDiscovery(
	ctx context.Context,
	ownerID uuid.UUID,
	id uuid.UUID,
	discovery asset.Discovery,
) error {
	if !discovery.Valid() {
		return ErrInvalidDiscovery
	}

	queries := db.New(s.pool)
	changed, err := queries.SetAssetDiscovery(ctx, db.SetAssetDiscoveryParams{
		ID:        uuidToPgtype(id),
		OwnerID:   uuidToPgtype(ownerID),
		Discovery: string(discovery),
	})
	if err != nil {
		return fmt.Errorf("set asset discovery: %w", err)
	}
	if changed == 1 {
		return nil
	}

	state, err := queries.AssetStateForOwner(ctx, db.AssetStateForOwnerParams{
		ID:      uuidToPgtype(id),
		OwnerID: uuidToPgtype(ownerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return asset.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check asset discovery: %w", err)
	}
	if state.WithheldAt.Valid {
		return asset.ErrAssetFrozen
	}
	if asset.Lifecycle(state.Lifecycle) == asset.LifecycleDraft {
		return asset.ErrAssetIsDraft
	}
	return asset.ErrNotFound
}

// Publish makes a draft public once it clears the publish floor, and records its first version
func (s *Service) Publish(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
	candidate *asset.Candidate,
) ([]asset.ReadinessItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, assetID); err != nil {
		return nil, err
	}

	var kind, name, lifecycle string
	var isNSFW *bool
	err = tx.QueryRow(ctx, `select kind, name, is_nsfw, lifecycle from assets where id = $1`, assetID).Scan(&kind, &name, &isNSFW, &lifecycle)
	if err != nil {
		return nil, fmt.Errorf("read asset to publish: %w", err)
	}

	if asset.Lifecycle(lifecycle) != asset.LifecycleDraft {
		return nil, ErrAlreadyPublished
	}

	blocks, err := block.Read(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	items, err := s.assets.CandidateReadiness(ctx, tx, assetID, kind, name, isNSFW, blocks)
	if err != nil {
		return nil, err
	}
	if !asset.Ready(items) {
		return items, asset.ErrPublishFloor
	}

	if _, err := tx.Exec(ctx, `
		update assets set lifecycle = 'published', updated_at = now()
		 where id = $1
	`, assetID); err != nil {
		return nil, fmt.Errorf("publish asset: %w", err)
	}
	if _, err := tx.Exec(ctx, `select record_initial_asset_snapshot($1, false)`, assetID); err != nil {
		return nil, fmt.Errorf("record initial publication: %w", err)
	}
	if err := candidate.Commit(ctx, tx, assetID); err != nil {
		return nil, err
	}
	return items, nil
}
