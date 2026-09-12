package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrPublishFloor = errors.New("the draft is not ready to publish")

var ErrAlreadyPublished = errors.New("the asset is already published")

const (
	nameRequirement         = "name"
	adultContentRequirement = "adult_content"
	exportRequirement       = "export"
	mediaRequirement        = "media"
	uploadRequirement       = "upload"
)

type ReadinessItem struct {
	ID      string
	Label   string
	Detail  string
	Met     bool
	BlockID *uuid.UUID
}

func Ready(items []ReadinessItem) bool {
	for _, item := range items {
		if !item.Met {
			return false
		}
	}
	return true
}

func readiness(kind, name string, isNSFW *bool, blocks []block.Block) []ReadinessItem {
	items := []ReadinessItem{
		{
			ID:     nameRequirement,
			Label:  "Name",
			Detail: fmt.Sprintf("Give this %s a name.", kind),
			Met:    name != "",
		},
		{
			ID:     adultContentRequirement,
			Label:  "Adult content answer",
			Detail: "Say whether this contains adult content.",
			Met:    isNSFW != nil,
		},
	}
	for _, check := range block.ContentFloor(kind, blocks) {
		items = append(items, ReadinessItem{
			ID: check.ID, Label: check.Label, Detail: check.Detail,
			Met: check.Met, BlockID: check.BlockID,
		})
	}
	return items
}

func (s *Service) candidateReadiness(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	kind, name string,
	isNSFW *bool,
	blocks []block.Block,
) ([]ReadinessItem, error) {
	items := readiness(kind, name, isNSFW, blocks)
	targets, err := s.exportCapability(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	items = append(items, ReadinessItem{
		ID:     exportRequirement,
		Label:  "A file to download",
		Detail: fmt.Sprintf("No format Illarin writes can hold this %s as it stands.", kind),
		Met:    len(targets) > 0 || !s.reg.WritesKind(kind),
	})
	pictures, err := picturesReady(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	items = append(items, ReadinessItem{
		ID:     mediaRequirement,
		Label:  "Pictures",
		Detail: "A picture on this page has no file behind it.",
		Met:    pictures,
	})
	reviewed, err := uploadReviewed(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	items = append(items, ReadinessItem{
		ID:     uploadRequirement,
		Label:  "Uploaded file",
		Detail: "An uploaded file is waiting to be accepted or cancelled.",
		Met:    reviewed,
	})
	return items, nil
}

func picturesReady(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (bool, error) {
	var ready bool
	err := tx.QueryRow(ctx, `
		with referenced as (
			select cover_media_id as media_id from assets
			 where id = $1 and cover_media_id is not null
			union
			select (value #>> '{}')::uuid from asset_blocks,
			    lateral jsonb_path_query(elements,
			        '$[*] ? (@.type == "image_set").content.images[*].mediaId') value
			 where asset_id = $1
		)
		select not exists (
			select 1 from referenced
			 where not exists (
				select 1 from asset_media
				 where id = referenced.media_id and asset_id = $1 and blob_id is not null))
	`, assetID).Scan(&ready)
	if err != nil {
		return false, fmt.Errorf("read the pictures to publish: %w", err)
	}
	return ready, nil
}

func uploadReviewed(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (bool, error) {
	var reviewed bool
	err := tx.QueryRow(ctx, `
		select not exists (
			select 1 from ingest_operations
			 where target_asset_id = $1 and status = 'preview')
	`, assetID).Scan(&reviewed)
	if err != nil {
		return false, fmt.Errorf("read the uploads waiting on this asset: %w", err)
	}
	return reviewed, nil
}

func (s *Service) Publish(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
	candidate *Candidate,
) ([]ReadinessItem, error) {
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

	if Lifecycle(lifecycle) != LifecycleDraft {
		return nil, ErrAlreadyPublished
	}

	blocks, err := readBlocks(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	items, err := s.candidateReadiness(ctx, tx, assetID, kind, name, isNSFW, blocks)
	if err != nil {
		return nil, err
	}
	if !Ready(items) {
		return items, ErrPublishFloor
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
	if err := candidate.commit(ctx, tx, assetID); err != nil {
		return nil, err
	}
	return items, nil
}
