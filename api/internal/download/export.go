package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrTargetNotOffered = errors.New("that download is not offered for this asset")

var ErrLinkedInstallOnly = errors.New("this asset is linked-install-only")

var ErrExportTooLarge = errors.New("that choice of images makes a file too large to produce")

var ErrExportImageUnreadable = errors.New("an image this download needs could not be read")

// MaxExportBytes is the largest file a download may produce.
const MaxExportBytes = 64 << 20

// GallerySelection is one reader's choice of gallery images for one download.
type GallerySelection struct {
	Images []uuid.UUID
}

func (g *GallerySelection) holds(mediaID uuid.UUID) bool {
	return slices.Contains(g.Images, mediaID)
}

type Export struct {
	Body      []byte
	MediaType string
	Filename  string
	Target    string
	Event     *Event
}

type exportSubject struct {
	assetID    uuid.UUID
	kind       string
	name       string
	origin     string
	header     format.Header
	blocks     []block.Block
	cover      *uuid.UUID
	ownerID    *uuid.UUID
	lifecycle  asset.Lifecycle
	revisionID *uuid.UUID
	gallery    *GallerySelection
	recorded   *asset.RecordedVersion
}

func (s *Service) OpenExport(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	tx, err := s.assets.BeginReadSnapshot(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("begin export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, err := s.exportSubject(ctx, tx, assetID, viewerID)
	if err != nil {
		return Export{}, err
	}
	subject.gallery = gallery
	if err := protected.ApplyPublishedPolicy(ctx, tx, assetID, subject.blocks); err != nil {
		return Export{}, err
	}
	apps, err := protected.Apps(ctx, tx, assetID)
	if err != nil {
		return Export{}, err
	}
	if len(apps) > 0 || protected.HasPromptFragments(subject.blocks) {
		return Export{}, ErrLinkedInstallOnly
	}
	offered := s.reg.OfferedTargets(subject.capability())
	if !offersTarget(offered, target) {
		return Export{}, ErrTargetNotOffered
	}
	module, known := s.reg.ByID(target)
	if !known {
		return Export{}, ErrTargetNotOffered
	}
	declaration := module.Declaration()
	writer, writes := module.(format.Writer)
	if !writes {
		return Export{}, ErrTargetNotOffered
	}
	written, err := s.writeExport(ctx, tx, subject, writer)
	if err != nil {
		return Export{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Export{}, fmt.Errorf("finish export snapshot: %w", err)
	}
	return subject.export(written, target, declaration.Label, viewerID), nil
}

func (subject exportSubject) export(
	written format.Artifact,
	target, label string,
	viewerID *uuid.UUID,
) Export {
	export := Export{
		Body: written.Body, MediaType: written.MediaType, Target: target,
		Filename: format.Filename(subject.name, subject.updateName(), label, written.Extension),
	}
	if subject.lifecycle == asset.LifecyclePublished {
		event := newEvent(subject.assetID, subject.revisionID, target, subject.ownerID, viewerID)
		export.Event = &event
	}
	return export
}

func (subject exportSubject) updateName() string {
	if subject.recorded == nil {
		return ""
	}
	return fmt.Sprintf("update %d", subject.recorded.Number)
}

func (s *Service) OpenExportForLinkedInstance(
	ctx context.Context,
	assetID uuid.UUID,
	target string,
) (Export, error) {
	tx, err := s.assets.BeginReadSnapshot(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("begin linked export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, err := s.exportSubject(ctx, tx, assetID, nil)
	if err != nil {
		return Export{}, err
	}
	apps, err := protected.Apps(ctx, tx, assetID)
	if err != nil {
		return Export{}, err
	}
	if len(apps) > 0 && !protected.AllowsTarget(apps, subject.kind, target) {
		return Export{}, ErrTargetNotOffered
	}
	if len(apps) == 0 {
		if err := protected.ApplyPublishedPolicy(ctx, tx, assetID, subject.blocks); err != nil {
			return Export{}, err
		}
		if protected.HasPromptFragments(subject.blocks) {
			return Export{}, ErrLinkedInstallOnly
		}
	}
	if err := protected.RestorePromptFragments(ctx, tx, assetID, subject.blocks); err != nil {
		return Export{}, err
	}
	if !offersTarget(s.reg.OfferedTargets(subject.capability()), target) {
		return Export{}, ErrTargetNotOffered
	}
	module, known := s.reg.ByID(target)
	if !known {
		return Export{}, ErrTargetNotOffered
	}
	writer, writes := module.(format.Writer)
	if !writes {
		return Export{}, ErrTargetNotOffered
	}
	written, err := s.writeExport(ctx, tx, subject, writer)
	if err != nil {
		return Export{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Export{}, fmt.Errorf("finish linked export snapshot: %w", err)
	}
	export := subject.export(written, target, module.Declaration().Label, nil)
	if export.Event != nil {
		export.Event.AuthorizationClass = AuthorizationLinkedInstance
	}
	return export, nil
}

func (s *Service) writeExport(
	ctx context.Context,
	q db.DBTX,
	subject exportSubject,
	writer format.Writer,
) (format.Artifact, error) {
	travelling := subject.travellingElements()
	work := format.ExportAsset{
		Kind: subject.kind, Header: subject.header, Elements: travelling,
	}
	cover, images, err := s.exportImages(ctx, q, subject, travelling)
	if err != nil {
		return format.Artifact{}, err
	}
	work.Cover, work.Images = cover, images
	work.Preserved, err = s.travellingPreservedData(ctx, q, subject, writer.ID())
	if err != nil {
		return format.Artifact{}, err
	}
	if declaration, known := s.reg.Declaration(writer.ID()); known && declaration.KeepsUpload {
		work.Upload, err = s.readUpload(ctx, q, subject)
		if err != nil {
			return format.Artifact{}, err
		}
	}
	written, err := writer.Write(ctx, work)
	if err != nil {
		return format.Artifact{}, fmt.Errorf("write %s: %w", writer.ID(), err)
	}
	if len(written.Body) > MaxExportBytes {
		return format.Artifact{}, ErrExportTooLarge
	}
	return written, nil
}

// readUpload reads the file behind the version being exported.
func (s *Service) readUpload(ctx context.Context, q db.DBTX, subject exportSubject) ([]byte, error) {
	if subject.revisionID == nil {
		return nil, fmt.Errorf("%w: no upload is recorded for this version", ErrTargetNotOffered)
	}
	var blobID uuid.UUID
	if err := q.QueryRow(ctx,
		`select blob_id from asset_revisions where id = $1 and asset_id = $2`,
		*subject.revisionID, subject.assetID,
	).Scan(&blobID); err != nil {
		return nil, fmt.Errorf("find the upload to export: %w", err)
	}
	upload, err := s.readBlob(ctx, blobID)
	if err != nil {
		return nil, fmt.Errorf("read the upload to export: %w", err)
	}
	return upload.Data, nil
}

func (subject exportSubject) elements() []block.Element {
	elements := make([]block.Element, 0)
	for _, holder := range subject.blocks {
		elements = append(elements, holder.Elements...)
	}
	return elements
}

// travellingElements drops the gallery images this download leaves behind.
func (subject exportSubject) travellingElements() []block.Element {
	elements := subject.elements()
	for i, element := range elements {
		if element.Role != block.RoleGallery {
			continue
		}
		set, isSet := element.Content.(block.ImageSet)
		if !isSet {
			continue
		}
		travelling := make([]block.ImageItem, 0, len(set.Images))
		for _, image := range set.Images {
			if subject.carries(image) {
				travelling = append(travelling, image)
			}
		}
		elements[i].Content = block.ImageSet{Images: travelling}
	}
	return elements
}

func (subject exportSubject) carries(image block.ImageItem) bool {
	if subject.gallery == nil {
		return !image.OmitFromDownloads
	}
	return subject.gallery.holds(image.MediaID)
}

func (subject exportSubject) capability() format.CapabilitySubject {
	return format.CapabilitySubject{
		Kind: subject.kind, Origin: subject.origin, Elements: subject.elements(),
	}
}

func offersTarget(offered []format.Target, target string) bool {
	for _, candidate := range offered {
		if candidate.Format == target {
			return true
		}
	}
	return false
}

func (s *Service) exportSubject(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) (exportSubject, error) {
	var subject exportSubject
	var origin pgtype.Text
	var ownerID, revisionID, cover pgtype.UUID
	err := q.QueryRow(ctx, `
		select asset.kind, asset.name, asset.blurb, asset.origin_format, asset.lifecycle,
		       asset.asset_version, asset.credited_author, asset.nickname,
		       asset.owner_id, asset.current_revision_id, asset.cover_media_id
		  from assets asset
		 where asset.id = $1 and asset.deleted_at is null
		   and (asset.lifecycle = 'published' or asset.owner_id = $2)
		   and (asset.withheld_at is null or asset.owner_id = $2)
	`, assetID, viewerID).Scan(
		&subject.kind, &subject.name, &subject.header.Blurb, &origin, &subject.lifecycle,
		&subject.header.AssetVersion, &subject.header.CreditedAuthor,
		&subject.header.Nickname, &ownerID, &revisionID, &cover,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportSubject{}, asset.ErrNotFound
	}
	if err != nil {
		return exportSubject{}, fmt.Errorf("read the asset to export: %w", err)
	}
	subject.assetID = assetID
	subject.header.Name = subject.name
	subject.origin = origin.String
	subject.cover = uuidOrNil(cover)
	subject.ownerID = uuidOrNil(ownerID)
	subject.revisionID = uuidOrNil(revisionID)
	subject.blocks, err = block.Read(ctx, q, assetID)
	if err != nil {
		return exportSubject{}, err
	}
	return subject, nil
}

func (s *Service) travellingPreservedData(
	ctx context.Context,
	q db.DBTX,
	subject exportSubject,
	target string,
) ([]format.Remainder, error) {
	if subject.origin == "" {
		return nil, nil
	}
	written, writes := s.reg.Declaration(target)
	if !writes || !s.reg.TravelsWithOrigin(subject.origin, written) {
		return nil, nil
	}
	if subject.recorded != nil {
		return recordedRemainder(*subject.recorded), nil
	}
	rows, err := q.Query(ctx, `
		select owner_kind, owner_id, namespace, payload
		  from asset_preserved_data
		 where asset_id = $1
		 order by namespace, owner_id
	`, subject.assetID)
	if err != nil {
		return nil, fmt.Errorf("read preserved data: %w", err)
	}
	defer rows.Close()
	preserved := make([]format.Remainder, 0)
	for rows.Next() {
		var row format.Remainder
		if err := rows.Scan(&row.Owner, &row.OwnerID, &row.Namespace, &row.Payload); err != nil {
			return nil, fmt.Errorf("read a preserved row: %w", err)
		}
		preserved = append(preserved, row)
	}
	return preserved, rows.Err()
}

func (s *Service) exportImages(
	ctx context.Context,
	q db.DBTX,
	subject exportSubject,
	travelling []block.Element,
) (*format.ExportMedia, map[uuid.UUID]format.ExportMedia, error) {
	wanted := make([]uuid.UUID, 0)
	for _, element := range travelling {
		switch content := element.Content.(type) {
		case block.ImageSet:
			for _, image := range content.Images {
				wanted = append(wanted, image.MediaID)
			}
		case block.RecordList:
			for _, record := range content.Records {
				if record.AvatarURL != nil {
					wanted = append(wanted, *record.AvatarURL)
				}
			}
		}
	}
	coverID := subject.cover
	if coverID != nil {
		wanted = append(wanted, *coverID)
	}
	if len(wanted) == 0 {
		return nil, map[uuid.UUID]format.ExportMedia{}, nil
	}

	rows, err := q.Query(ctx, subject.pictureQuery(), subject.assetID, wanted, subject.snapshotID())
	if err != nil {
		return nil, nil, fmt.Errorf("list the pictures to export: %w", err)
	}
	defer rows.Close()
	blobs := make(map[uuid.UUID]uuid.UUID)
	var total int64
	for rows.Next() {
		var mediaID, blobID uuid.UUID
		var size int64
		if err := rows.Scan(&mediaID, &blobID, &size); err != nil {
			return nil, nil, fmt.Errorf("read a picture to export: %w", err)
		}
		blobs[mediaID] = blobID
		total += size
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("list the pictures to export: %w", err)
	}
	if total > MaxExportBytes {
		return nil, nil, ErrExportTooLarge
	}

	images := make(map[uuid.UUID]format.ExportMedia, len(blobs))
	private := subject.lifecycle == asset.LifecycleDraft
	for mediaID, blobID := range blobs {
		picture, err := s.readBlob(ctx, blobID)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: picture %s: %w", ErrExportImageUnreadable, mediaID, err)
		}
		picture.URL = s.assets.ExportMediaURL(mediaID, private)
		images[mediaID] = picture
	}
	var cover *format.ExportMedia
	if coverID != nil {
		if picture, held := images[*coverID]; held {
			cover = &picture
			delete(images, *coverID)
		}
	}
	return cover, images, nil
}

// pictureQuery lists the pictures a download may carry, published now or kept by a recorded version.
func (subject exportSubject) pictureQuery() string {
	if subject.recorded == nil {
		return `
			select media.id, media.blob_id, blob.byte_size
			  from asset_media media
			  join blobs blob on blob.id = media.blob_id
			 where media.asset_id = $1 and media.is_current and media.id = any($2)
			   and $3::uuid is null`
	}
	return `
		select media.id, media.blob_id, blob.byte_size
		  from public.asset_media media
		  join public.asset_snapshot_media kept on kept.media_id = media.id
		  join public.blobs blob on blob.id = media.blob_id
		 where media.asset_id = $1 and media.id = any($2) and kept.snapshot_id = $3`
}

func (subject exportSubject) snapshotID() *uuid.UUID {
	if subject.recorded == nil {
		return nil
	}
	return &subject.recorded.ID
}

func (s *Service) readBlob(ctx context.Context, blobID uuid.UUID) (format.ExportMedia, error) {
	opened, err := s.store.Open(ctx, blobID)
	if err != nil {
		return format.ExportMedia{}, err
	}
	data, readErr := io.ReadAll(opened)
	closeErr := opened.Close()
	if readErr != nil {
		return format.ExportMedia{}, readErr
	}
	if closeErr != nil {
		return format.ExportMedia{}, closeErr
	}
	return format.ExportMedia{MediaType: http.DetectContentType(data), Data: data}, nil
}

func uuidOrNil(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	found := uuid.UUID(p.Bytes)
	return &found
}
