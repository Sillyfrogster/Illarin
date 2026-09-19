package work

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) DeliverableWork(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
) (connect.Deliverable, error) {
	var found connect.Deliverable
	var number int32
	var originalFileID, coverID pgtype.UUID
	err := q.QueryRow(ctx, `
		select type, name, version_number, original_file_id, cover_media_id
		  from work_public.works
		 where id = $1
		   and deleted_at is null
		   and withheld_at is null
		   and lifecycle = 'published'
	`, workID).Scan(&found.Type, &found.Name, &number, &originalFileID, &coverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.Deliverable{}, connect.ErrNotDeliverable
	}
	if err != nil {
		return connect.Deliverable{}, fmt.Errorf("read the work to deliver: %w", err)
	}
	found.VersionNumber = int(number)
	found.HasOriginal = originalFileID.Valid

	offered, err := deliveryTargets(ctx, q, workID)
	if err != nil {
		return connect.Deliverable{}, err
	}
	found.Targets = offered
	apps, err := private.Apps(ctx, q, workID)
	if err != nil {
		return connect.Deliverable{}, err
	}
	if len(apps) == 0 {
		blocks, err := ReadPublishedBlocks(ctx, q, workID)
		if err != nil {
			return connect.Deliverable{}, err
		}
		if err := private.ApplyPublishedPolicy(ctx, q, workID, blocks); err != nil {
			return connect.Deliverable{}, err
		}
		if private.HasPromptFragments(blocks) {
			return connect.Deliverable{}, connect.ErrNotDeliverable
		}
	}
	if len(apps) > 0 {
		found.HasOriginal = false
		filtered := make([]connect.DeliveryTarget, 0, len(found.Targets))
		for _, target := range found.Targets {
			if private.AllowsTarget(apps, found.Type, target.Format) {
				filtered = append(filtered, target)
			}
		}
		found.Targets = filtered
	}
	found.Pictures, err = s.deliveryPictures(ctx, q, workID, uuidOrNil(coverID))
	if err != nil {
		return connect.Deliverable{}, err
	}
	found.InstallCapabilities = format.InstallCapabilities(found.Type, targetFormats(found.Targets))
	return found, nil
}

func targetFormats(targets []connect.DeliveryTarget) []string {
	formats := make([]string, 0, len(targets))
	for _, target := range targets {
		formats = append(formats, target.Format)
	}
	return formats
}

func deliveryTargets(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
) ([]connect.DeliveryTarget, error) {
	var stored []byte
	err := q.QueryRow(ctx,
		`select export from work_public.work_summaries where work_id = $1`, workID,
	).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return []connect.DeliveryTarget{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the export summary to deliver: %w", err)
	}
	offered := make([]format.Target, 0)
	if err := json.Unmarshal(stored, &offered); err != nil {
		return nil, fmt.Errorf("read the stored export summary: %w", err)
	}
	targets := make([]connect.DeliveryTarget, 0, len(offered))
	for _, target := range offered {
		targets = append(targets, connect.DeliveryTarget{Format: target.Format, Label: target.Label})
	}
	return targets, nil
}

func (s *Service) deliveryPictures(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
	coverID *uuid.UUID,
) ([]connect.DeliveryPicture, error) {
	blocks, err := ReadPublishedBlocks(ctx, q, workID)
	if err != nil {
		return nil, fmt.Errorf("read the blocks to deliver: %w", err)
	}
	withheld := galleryImagesLeftBehind(blocks)
	rows, err := q.Query(ctx, `
		select id, role
		  from work_public.work_media
		 where work_id = $1
		   and is_current
		   and blob_id is not null
		 order by created_at desc, id desc
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("list the pictures to deliver: %w", err)
	}
	defer rows.Close()
	pictures := make([]connect.DeliveryPicture, 0)
	for rows.Next() {
		var picture connect.DeliveryPicture
		if err := rows.Scan(&picture.MediaID, &picture.Role); err != nil {
			return nil, fmt.Errorf("read a picture to deliver: %w", err)
		}
		if withheld[picture.MediaID] {
			continue
		}
		picture.IsCover = coverID != nil && *coverID == picture.MediaID
		picture.URL = s.ExportMediaURL(picture.MediaID, true)
		pictures = append(pictures, picture)
	}
	return pictures, rows.Err()
}

// galleryImagesLeftBehind names the gallery images the creator keeps out of downloads.
func galleryImagesLeftBehind(blocks []block.Block) map[uuid.UUID]bool {
	withheld := make(map[uuid.UUID]bool)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			set, isSet := element.Content.(block.ImageSet)
			if !isSet || element.Role != block.RoleGallery {
				continue
			}
			for _, image := range set.Images {
				if image.OmitFromDownloads {
					withheld[image.MediaID] = true
				}
			}
		}
	}
	return withheld
}

func (s *Service) SignedURL(path string) string {
	return s.siteURL + s.signer.Sign(path, s.now())
}

func (s *Service) ValidSignature(path, expires, signature string) bool {
	return s.signer.Valid(path, expires, signature, s.now())
}

func ReadPublishedBlocks(ctx context.Context, q db.DBTX, workID uuid.UUID) ([]block.Block, error) {
	var stored []byte
	err := q.QueryRow(ctx, `select coalesce(jsonb_agg(to_jsonb(b) order by position), '[]'::jsonb)
		from work_public.work_blocks b where work_id = $1`, workID).Scan(&stored)
	if err != nil {
		return nil, err
	}
	var blocks []block.Block
	if err := json.Unmarshal(stored, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
