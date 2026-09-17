package asset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotDeliverable = errors.New("that asset cannot be sent to an instance")

type DeliveryTarget struct {
	Format string
	Label  string
}

type DeliveryPicture struct {
	MediaID uuid.UUID
	Role    string
	IsCover bool
	URL     string
}

type Deliverable struct {
	Kind              string
	Name              string
	ContentGeneration int
	Targets           []DeliveryTarget
	HasOriginal       bool
	Pictures          []DeliveryPicture
	// InstallCapabilities lists what an instance must declare, any one of them, before the asset is sent to it.
	InstallCapabilities []string
}

func (s *Service) DeliverableAsset(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
) (Deliverable, error) {
	var found Deliverable
	var generation int32
	var revisionID, coverID pgtype.UUID
	err := q.QueryRow(ctx, `
		select kind, name, content_generation, current_revision_id, cover_media_id
		  from asset_public.assets
		 where id = $1
		   and deleted_at is null
		   and withheld_at is null
		   and lifecycle = 'published'
	`, assetID).Scan(&found.Kind, &found.Name, &generation, &revisionID, &coverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deliverable{}, ErrNotDeliverable
	}
	if err != nil {
		return Deliverable{}, fmt.Errorf("read the asset to deliver: %w", err)
	}
	found.ContentGeneration = int(generation)
	found.HasOriginal = revisionID.Valid

	offered, err := deliveryTargets(ctx, q, assetID)
	if err != nil {
		return Deliverable{}, err
	}
	found.Targets = offered
	apps, err := protected.Apps(ctx, q, assetID)
	if err != nil {
		return Deliverable{}, err
	}
	if len(apps) == 0 {
		blocks, err := ReadPublishedBlocks(ctx, q, assetID)
		if err != nil {
			return Deliverable{}, err
		}
		if err := protected.ApplyPublishedPolicy(ctx, q, assetID, blocks); err != nil {
			return Deliverable{}, err
		}
		if protected.HasPromptFragments(blocks) {
			return Deliverable{}, ErrNotDeliverable
		}
	}
	if len(apps) > 0 {
		found.HasOriginal = false
		filtered := make([]DeliveryTarget, 0, len(found.Targets))
		for _, target := range found.Targets {
			if targetAllowed(apps, found.Kind, target.Format) {
				filtered = append(filtered, target)
			}
		}
		found.Targets = filtered
	}
	found.Pictures, err = s.deliveryPictures(ctx, q, assetID, uuidOrNil(coverID))
	if err != nil {
		return Deliverable{}, err
	}
	found.InstallCapabilities = format.InstallCapabilities(found.Kind, targetFormats(found.Targets))
	return found, nil
}

func targetFormats(targets []DeliveryTarget) []string {
	formats := make([]string, 0, len(targets))
	for _, target := range targets {
		formats = append(formats, target.Format)
	}
	return formats
}

func deliveryTargets(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
) ([]DeliveryTarget, error) {
	var stored []byte
	err := q.QueryRow(ctx,
		`select export from asset_public.asset_projections where asset_id = $1`, assetID,
	).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return []DeliveryTarget{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the export projection to deliver: %w", err)
	}
	offered := make([]format.Target, 0)
	if err := json.Unmarshal(stored, &offered); err != nil {
		return nil, fmt.Errorf("read the stored export projection: %w", err)
	}
	targets := make([]DeliveryTarget, 0, len(offered))
	for _, target := range offered {
		targets = append(targets, DeliveryTarget{Format: target.Format, Label: target.Label})
	}
	return targets, nil
}

func (s *Service) deliveryPictures(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
	coverID *uuid.UUID,
) ([]DeliveryPicture, error) {
	blocks, err := ReadPublishedBlocks(ctx, q, assetID)
	if err != nil {
		return nil, fmt.Errorf("read the blocks to deliver: %w", err)
	}
	withheld := galleryImagesLeftBehind(blocks)
	rows, err := q.Query(ctx, `
		select id, role
		  from asset_public.asset_media
		 where asset_id = $1
		   and is_current
		   and blob_id is not null
		 order by created_at desc, id desc
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list the pictures to deliver: %w", err)
	}
	defer rows.Close()
	pictures := make([]DeliveryPicture, 0)
	for rows.Next() {
		var picture DeliveryPicture
		if err := rows.Scan(&picture.MediaID, &picture.Role); err != nil {
			return nil, fmt.Errorf("read a picture to deliver: %w", err)
		}
		if withheld[picture.MediaID] {
			continue
		}
		picture.IsCover = coverID != nil && *coverID == picture.MediaID
		picture.URL = s.exportMediaURL(picture.MediaID, true)
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

func (s *Service) DownloadSourceForLinkedInstance(
	ctx context.Context,
	assetID uuid.UUID,
) (SourceDownload, error) {
	download, err := s.DownloadSource(ctx, assetID, nil)
	if err != nil {
		return SourceDownload{}, err
	}
	download.Event.AuthorizationClass = AuthorizationLinkedInstance
	return download, nil
}

func ReadPublishedBlocks(ctx context.Context, q db.DBTX, assetID uuid.UUID) ([]block.Block, error) {
	var stored []byte
	err := q.QueryRow(ctx, `select coalesce(jsonb_agg(to_jsonb(b) order by position), '[]'::jsonb)
		from asset_public.asset_blocks b where asset_id = $1`, assetID).Scan(&stored)
	if err != nil {
		return nil, err
	}
	var blocks []block.Block
	if err := json.Unmarshal(stored, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
