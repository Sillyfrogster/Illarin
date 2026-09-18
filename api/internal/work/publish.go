package work

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrPublishFloor = errors.New("the draft is not ready to publish")

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

func PublishedShortfall(kind, name string, isNSFW *bool, blocks []block.Block) []ReadinessItem {
	items := Readiness(kind, name, isNSFW, blocks)
	if Ready(items) {
		return nil
	}
	return items
}

func Readiness(kind, name string, isNSFW *bool, blocks []block.Block) []ReadinessItem {
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

func (s *Service) CandidateReadiness(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	kind, name string,
	isNSFW *bool,
	blocks []block.Block,
) ([]ReadinessItem, error) {
	items := Readiness(kind, name, isNSFW, blocks)
	targets, err := summary.Formats(ctx, tx, s.reg, assetID)
	if err != nil {
		return nil, missingAsset(err)
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
