package download

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OpenRecordedExport writes one recorded version through the current writer for the format.
func (s *Service) OpenRecordedExport(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
	formatID string,
	gallery *GallerySelection,
) (Export, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Export{}, fmt.Errorf("begin recorded export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, sealed, err := s.recordedExportSubject(ctx, tx, workID, viewerID, number)
	if err != nil {
		return Export{}, err
	}
	if sealed {
		return Export{}, ErrLinkedInstallOnly
	}
	subject.gallery = gallery
	module, known := s.reg.ByID(formatID)
	writer, writes := module.(format.Writer)
	if !known || !writes || !offersFormat(s.reg.OfferedFormats(subject.capability()), formatID) {
		return Export{}, ErrFormatNotOffered
	}
	written, err := s.writeExport(ctx, tx, subject, writer)
	if err != nil {
		return Export{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Export{}, fmt.Errorf("finish recorded export snapshot: %w", err)
	}
	return subject.export(written, formatID, module.Declaration().Label, viewerID), nil
}

// RecordedDownloads lists the formats and media available for a historical download.
type RecordedDownloads struct {
	Version           work.Version
	Type              string
	LinkedInstallOnly bool
	Downloads         []format.Offered
	AppFormats        []format.AppFormat
	Blocks            []block.Block
	Media             []work.DetailImage
}

// RecordedDownloads reads a historical version's download choices under current protection.
func (s *Service) RecordedDownloads(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
	preference work.NSFWPreference,
) (RecordedDownloads, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return RecordedDownloads{}, fmt.Errorf("begin recorded downloads snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, sealed, err := s.recordedExportSubject(ctx, tx, workID, viewerID, number)
	if err != nil {
		return RecordedDownloads{}, err
	}
	offered := RecordedDownloads{
		Version: subject.recorded.Version, Type: subject.workType, LinkedInstallOnly: sealed,
		Downloads: []format.Offered{}, AppFormats: []format.AppFormat{}, Blocks: []block.Block{},
	}
	if !sealed {
		offered.Downloads = s.reg.OfferedFormats(subject.capability())
		offered.AppFormats = format.AppFormats(offered.Downloads, s.reg)
		offered.Blocks = subject.blocks
	}
	offered.Media, err = s.recordedPictures(ctx, tx, subject, preference)
	if err != nil {
		return RecordedDownloads{}, err
	}
	return offered, nil
}

func (s *Service) recordedPictures(
	ctx context.Context,
	tx pgx.Tx,
	subject exportSubject,
	preference work.NSFWPreference,
) ([]work.DetailImage, error) {
	var flagged *bool
	if err := tx.QueryRow(ctx,
		`select is_nsfw from public.works where id = $1`, subject.workID).Scan(&flagged); err != nil {
		return nil, fmt.Errorf("read the work to address its pictures: %w", err)
	}
	blurred := flagged != nil && *flagged && preference != work.NSFWShown
	rows, err := tx.Query(ctx, `
		select media.id, media.role, media.width, media.height, blob.byte_size
		  from public.work_version_media kept
		  join public.work_media media on media.id = kept.media_id
		  join public.blobs blob on blob.id = media.blob_id
		 where kept.version_id = $1
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
	pictures := make([]work.DetailImage, 0)
	for rows.Next() {
		var picture work.DetailImage
		var width, height *int32
		if err := rows.Scan(&picture.ID, &picture.Role, &width, &height, &picture.Bytes); err != nil {
			return nil, fmt.Errorf("read a picture a version recorded: %w", err)
		}
		picture.Width, picture.Height = int(*width), int(*height)
		picture.IsCover = subject.cover != nil && *subject.cover == picture.ID
		picture.DetailURL = s.works.ImageAddress(picture.ID, "detail", blurred, false)
		picture.ThumbURL = s.works.ImageAddress(picture.ID, "thumb", blurred, false)
		pictures = append(pictures, picture)
	}
	return pictures, rows.Err()
}

// recordedExportSubject loads a historical version under current access and protection.
func (s *Service) recordedExportSubject(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	viewerID *uuid.UUID,
	number int,
) (exportSubject, bool, error) {
	var subject exportSubject
	var ownerID *uuid.UUID
	err := tx.QueryRow(ctx, `
		select work.owner_id, work.lifecycle
		  from public.works work
		 where work.id = $1 and work.deleted_at is null
		   and (work.lifecycle = 'published' or work.owner_id = $2)
		   and (work.withheld_at is null or work.owner_id = $2)
	`, workID, viewerID).Scan(&ownerID, &subject.lifecycle)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportSubject{}, false, work.ErrNotFound
	}
	if err != nil {
		return exportSubject{}, false, fmt.Errorf("read the work to export: %w", err)
	}
	recorded, err := work.ReadVersion(ctx, tx, workID, number)
	if err != nil {
		return exportSubject{}, false, err
	}
	if recorded.WithdrawnAt != nil && (viewerID == nil || ownerID == nil || *viewerID != *ownerID) {
		return exportSubject{}, false, work.ErrNotFound
	}
	if err := private.RestoreRecordedPrompts(recorded.ProtectedPayloads, recorded.Blocks); err != nil {
		return exportSubject{}, false, err
	}
	if _, err := private.ApplyRecordedPolicy(ctx, tx, workID, &recorded.ID, recorded.Blocks); err != nil {
		return exportSubject{}, false, err
	}
	apps, err := private.Apps(ctx, tx, workID)
	if err != nil {
		return exportSubject{}, false, err
	}
	sealed := len(apps) > 0 || private.HasPromptFragments(recorded.Blocks)
	subject.workID = workID
	subject.workType = recorded.Type
	subject.name = recorded.Metadata.Name
	subject.origin = recorded.Origin
	subject.header = format.Header{
		Name:           recorded.Metadata.Name,
		Blurb:          recorded.Metadata.Blurb,
		WorkVersion:    recorded.Metadata.WorkVersion,
		CreditedAuthor: recorded.Metadata.CreditedAuthor,
		Nickname:       recorded.Metadata.Nickname,
	}
	subject.blocks = recorded.Blocks
	subject.cover = recorded.Metadata.Cover
	subject.ownerID = ownerID
	subject.originalFileID = recorded.OriginalFileID
	subject.recorded = &recorded
	return subject, sealed, nil
}

func recordedRemainder(v work.FullVersion) []format.Remainder {
	preserved := make([]format.Remainder, 0, len(v.Preserved))
	for _, row := range v.Preserved {
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
