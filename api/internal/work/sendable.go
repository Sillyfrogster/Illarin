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

func (s *Service) SendableWork(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
) (connect.Sendable, error) {
	var found connect.Sendable
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
		return connect.Sendable{}, connect.ErrNotSendable
	}
	if err != nil {
		return connect.Sendable{}, fmt.Errorf("read the work to send: %w", err)
	}
	found.VersionNumber = int(number)
	found.HasOriginal = originalFileID.Valid

	offered, err := sendFormats(ctx, q, workID)
	if err != nil {
		return connect.Sendable{}, err
	}
	found.Formats = offered
	apps, err := private.Apps(ctx, q, workID)
	if err != nil {
		return connect.Sendable{}, err
	}
	if len(apps) == 0 {
		blocks, err := ReadPublishedBlocks(ctx, q, workID)
		if err != nil {
			return connect.Sendable{}, err
		}
		if err := private.ApplyPublishedPolicy(ctx, q, workID, blocks); err != nil {
			return connect.Sendable{}, err
		}
		if private.HasPromptFragments(blocks) {
			return connect.Sendable{}, connect.ErrNotSendable
		}
	}
	if len(apps) > 0 {
		found.HasOriginal = false
		filtered := make([]connect.SendFormat, 0, len(found.Formats))
		for _, one := range found.Formats {
			if private.AllowsFormat(s.reg, apps, one.Format) {
				filtered = append(filtered, one)
			}
		}
		found.Formats = filtered
	}
	found.Pictures, err = s.sendPictures(ctx, q, workID, uuidOrNil(coverID))
	if err != nil {
		return connect.Sendable{}, err
	}
	found.InstallCapabilities = format.InstallCapabilities(found.Type, formatIDs(found.Formats))
	return found, nil
}

func formatIDs(offered []connect.SendFormat) []string {
	ids := make([]string, 0, len(offered))
	for _, one := range offered {
		ids = append(ids, one.Format)
	}
	return ids
}

func sendFormats(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
) ([]connect.SendFormat, error) {
	var stored []byte
	err := q.QueryRow(ctx,
		`select export from work_public.work_summaries where work_id = $1`, workID,
	).Scan(&stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return []connect.SendFormat{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the formats summary to send: %w", err)
	}
	offered := make([]format.Offered, 0)
	if err := json.Unmarshal(stored, &offered); err != nil {
		return nil, fmt.Errorf("read the stored export summary: %w", err)
	}
	formats := make([]connect.SendFormat, 0, len(offered))
	for _, one := range offered {
		formats = append(formats, connect.SendFormat{Format: one.Format, Label: one.Label})
	}
	return formats, nil
}

func (s *Service) sendPictures(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
	coverID *uuid.UUID,
) ([]connect.SendPicture, error) {
	blocks, err := ReadPublishedBlocks(ctx, q, workID)
	if err != nil {
		return nil, fmt.Errorf("read the blocks to send: %w", err)
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
		return nil, fmt.Errorf("list the pictures to send: %w", err)
	}
	defer rows.Close()
	pictures := make([]connect.SendPicture, 0)
	for rows.Next() {
		var picture connect.SendPicture
		if err := rows.Scan(&picture.MediaID, &picture.Role); err != nil {
			return nil, fmt.Errorf("read a picture to send: %w", err)
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
