package asset

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OpenRecordedExport writes one recorded version through the current writer for the target.
func (s *Service) OpenRecordedExport(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Export{}, fmt.Errorf("begin recorded export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, sealed, err := s.recordedExportSubject(ctx, tx, assetID, viewerID, number)
	if err != nil {
		return Export{}, err
	}
	if sealed {
		return Export{}, ErrLinkedInstallOnly
	}
	subject.gallery = gallery
	module, known := s.reg.ByID(target)
	writer, writes := module.(format.Writer)
	if !known || !writes || !offersTarget(s.reg.OfferedTargets(subject.capability()), target) {
		return Export{}, ErrTargetNotOffered
	}
	written, err := s.writeExport(ctx, tx, subject, writer)
	if err != nil {
		return Export{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Export{}, fmt.Errorf("finish recorded export snapshot: %w", err)
	}
	return subject.export(written, target, module.Declaration().Label, viewerID), nil
}

// RecordedDownloads is what a file of one recorded version can carry today.
type RecordedDownloads struct {
	Version           Version
	Kind              string
	LinkedInstallOnly bool
	Downloads         []format.Target
	AppTargets        []format.AppTarget
	Blocks            []block.Block
	Media             []DetailImage
}

// RecordedDownloads reads the formats, pictures and protection standing of one recorded version.
func (s *Service) RecordedDownloads(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
	visibility ContentVisibility,
) (RecordedDownloads, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return RecordedDownloads{}, fmt.Errorf("begin recorded downloads snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, sealed, err := s.recordedExportSubject(ctx, tx, assetID, viewerID, number)
	if err != nil {
		return RecordedDownloads{}, err
	}
	offered := RecordedDownloads{
		Version: subject.recorded.Version, Kind: subject.kind, LinkedInstallOnly: sealed,
		Downloads: []format.Target{}, AppTargets: []format.AppTarget{}, Blocks: []block.Block{},
	}
	if !sealed {
		offered.Downloads = s.reg.OfferedTargets(subject.capability())
		offered.AppTargets = format.AppTargets(offered.Downloads, s.reg)
		offered.Blocks = subject.blocks
	}
	offered.Media, err = s.recordedPictures(ctx, tx, subject, visibility)
	if err != nil {
		return RecordedDownloads{}, err
	}
	return offered, nil
}

func (s *Service) recordedPictures(
	ctx context.Context,
	tx pgx.Tx,
	subject exportSubject,
	visibility ContentVisibility,
) ([]DetailImage, error) {
	var flagged *bool
	if err := tx.QueryRow(ctx,
		`select is_nsfw from public.assets where id = $1`, subject.assetID).Scan(&flagged); err != nil {
		return nil, fmt.Errorf("read the asset to address its pictures: %w", err)
	}
	blurred := flagged != nil && *flagged && visibility != ContentShown
	rows, err := tx.Query(ctx, `
		select media.id, media.role, media.width, media.height, blob.byte_size
		  from public.asset_snapshot_media kept
		  join public.asset_media media on media.id = kept.media_id
		  join public.blobs blob on blob.id = media.blob_id
		 where kept.snapshot_id = $1
		   and media.width is not null and media.height is not null
		 order by (media.id = $2) desc,
		          case media.role
		            when 'avatar' then 1
		            when 'avatar_alt' then 2
		            when 'gallery' then 3
		            when 'expression' then 4
		            when 'pack_item' then 5
		            else 6
		          end,
		          media.created_at desc, media.id desc
	`, subject.recorded.ID, subject.cover)
	if err != nil {
		return nil, fmt.Errorf("list the pictures a version recorded: %w", err)
	}
	defer rows.Close()
	pictures := make([]DetailImage, 0)
	for rows.Next() {
		var picture DetailImage
		var width, height *int32
		if err := rows.Scan(&picture.ID, &picture.Role, &width, &height, &picture.Bytes); err != nil {
			return nil, fmt.Errorf("read a picture a version recorded: %w", err)
		}
		picture.Width, picture.Height = int(*width), int(*height)
		picture.IsCover = subject.cover != nil && *subject.cover == picture.ID
		picture.DetailURL = s.variantURL(picture.ID, "detail", blurred, false)
		picture.ThumbURL = s.variantURL(picture.ID, "thumb", blurred, false)
		pictures = append(pictures, picture)
	}
	return pictures, rows.Err()
}

// recordedExportSubject reads a recorded version as an export subject held to the asset's current access and protection.
func (s *Service) recordedExportSubject(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
) (exportSubject, bool, error) {
	var subject exportSubject
	var ownerID *uuid.UUID
	err := tx.QueryRow(ctx, `
		select asset.owner_id, asset.lifecycle
		  from public.assets asset
		 where asset.id = $1 and asset.deleted_at is null
		   and (asset.lifecycle = 'published' or asset.owner_id = $2)
		   and (asset.withheld_at is null or asset.owner_id = $2)
	`, assetID, viewerID).Scan(&ownerID, &subject.lifecycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportSubject{}, false, ErrNotFound
	}
	if err != nil {
		return exportSubject{}, false, fmt.Errorf("read the asset to export: %w", err)
	}
	recorded, err := readVersion(ctx, tx, assetID, number)
	if err != nil {
		return exportSubject{}, false, err
	}
	if recorded.WithdrawnAt != nil && (viewerID == nil || ownerID == nil || *viewerID != *ownerID) {
		return exportSubject{}, false, ErrNotFound
	}
	if err := protected.RestoreRecordedPrompts(recorded.protectedPayloads, recorded.blocks); err != nil {
		return exportSubject{}, false, err
	}
	if _, err := protected.ApplyRecordedPolicy(ctx, tx, assetID, &recorded.ID, recorded.blocks); err != nil {
		return exportSubject{}, false, err
	}
	apps, err := protected.Apps(ctx, tx, assetID)
	if err != nil {
		return exportSubject{}, false, err
	}
	sealed := len(apps) > 0 || protected.HasPromptFragments(recorded.blocks)
	subject.assetID = assetID
	subject.kind = recorded.kind
	subject.name = recorded.metadata.Name
	subject.origin = recorded.origin
	subject.header = format.Header{
		Name:           recorded.metadata.Name,
		Blurb:          recorded.metadata.Blurb,
		AssetVersion:   recorded.metadata.AssetVersion,
		CreditedAuthor: recorded.metadata.CreditedAuthor,
		Nickname:       recorded.metadata.Nickname,
	}
	subject.blocks = recorded.blocks
	subject.cover = recorded.metadata.Cover
	subject.ownerID = ownerID
	subject.revisionID = recorded.sourceRevisionID
	subject.recorded = &recorded
	return subject, sealed, nil
}

func (v recordedVersion) remainder() []format.Remainder {
	preserved := make([]format.Remainder, 0, len(v.preserved))
	for _, row := range v.preserved {
		preserved = append(preserved, format.Remainder{
			Owner: format.Owner(row.Owner), OwnerID: row.OwnerID,
			Namespace: row.Namespace, Payload: []byte(row.Payload),
		})
	}
	slices.SortFunc(preserved, func(a, b format.Remainder) int {
		return cmp.Or(strings.Compare(a.Namespace, b.Namespace), strings.Compare(a.OwnerID.String(), b.OwnerID.String()))
	})
	return preserved
}
