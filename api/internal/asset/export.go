package asset

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

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
	Event     *DownloadEvent
}

type exportSubject struct {
	assetID    uuid.UUID
	kind       string
	name       string
	origin     string
	header     format.Header
	blocks     []block.Block
	ownerID    *uuid.UUID
	lifecycle  Lifecycle
	revisionID *uuid.UUID
	gallery    *GallerySelection
}

func (s *Service) OpenExport(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	tx, err := s.beginReadSnapshot(ctx)
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
	export := Export{
		Body: written.Body, MediaType: written.MediaType, Target: target,
		Filename: downloadFilename(subject.name, declaration.Label, written.Extension),
	}
	if subject.lifecycle == LifecyclePublished {
		event := downloadEvent(assetID, subject.revisionID, target, subject.ownerID, viewerID)
		export.Event = &event
	}
	return export, nil
}

func (s *Service) OpenExportForLinkedInstance(
	ctx context.Context,
	assetID uuid.UUID,
	target string,
) (Export, error) {
	tx, err := s.beginReadSnapshot(ctx)
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
	if len(apps) > 0 && !targetAllowed(apps, subject.kind, target) {
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
	export := Export{
		Body: written.Body, MediaType: written.MediaType, Target: target,
		Filename: downloadFilename(subject.name, module.Declaration().Label, written.Extension),
	}
	if subject.lifecycle == LifecyclePublished {
		event := downloadEvent(assetID, subject.revisionID, target, subject.ownerID, nil)
		event.AuthorizationClass = AuthorizationLinkedInstance
		export.Event = &event
	}
	return export, nil
}

func targetAllowed(apps []string, kind, target string) bool {
	for _, app := range apps {
		for _, accepted := range protected.AppTargets(kind, app) {
			if target == accepted {
				return true
			}
		}
	}
	return false
}

func (s *Service) writeExport(
	ctx context.Context,
	q db.DBTX,
	subject exportSubject,
	writer format.Writer,
) (format.Artifact, error) {
	travelling := subject.travellingElements()
	asset := format.ExportAsset{
		Kind: subject.kind, Header: subject.header, Elements: travelling,
	}
	cover, images, err := s.exportImages(ctx, q, subject, travelling)
	if err != nil {
		return format.Artifact{}, err
	}
	asset.Cover, asset.Images = cover, images
	asset.Preserved, err = s.travellingPreservedData(ctx, q, subject, writer.ID())
	if err != nil {
		return format.Artifact{}, err
	}
	written, err := writer.Write(ctx, asset)
	if err != nil {
		return format.Artifact{}, fmt.Errorf("write %s: %w", writer.ID(), err)
	}
	if len(written.Body) > MaxExportBytes {
		return format.Artifact{}, ErrExportTooLarge
	}
	return written, nil
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
	var ownerID, revisionID pgtype.UUID
	err := q.QueryRow(ctx, `
		select asset.kind, asset.name, asset.blurb, asset.origin_format, asset.lifecycle,
		       asset.asset_version, asset.credited_author, asset.nickname,
		       asset.owner_id, asset.current_revision_id
		  from assets asset
		 where asset.id = $1 and asset.deleted_at is null
		   and (asset.lifecycle = 'published' or asset.owner_id = $2)
		   and (asset.withheld_at is null or asset.owner_id = $2)
	`, assetID, viewerID).Scan(
		&subject.kind, &subject.name, &subject.header.Blurb, &origin, &subject.lifecycle,
		&subject.header.AssetVersion, &subject.header.CreditedAuthor,
		&subject.header.Nickname, &ownerID, &revisionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportSubject{}, ErrNotFound
	}
	if err != nil {
		return exportSubject{}, fmt.Errorf("read the asset to export: %w", err)
	}
	subject.assetID = assetID
	subject.header.Name = subject.name
	subject.origin = origin.String
	subject.ownerID = uuidOrNil(ownerID)
	subject.revisionID = uuidOrNil(revisionID)
	subject.blocks, err = readBlocks(ctx, q, assetID)
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
	origin, known := s.reg.Declaration(subject.origin)
	written, writes := s.reg.Declaration(target)
	if !known || !writes || !format.TravelsWithOrigin(origin, written) {
		return nil, nil
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
	var held pgtype.UUID
	if err := q.QueryRow(ctx,
		`select cover_media_id from assets where id = $1`, subject.assetID,
	).Scan(&held); err != nil {
		return nil, nil, fmt.Errorf("read the cover: %w", err)
	}
	coverID := uuidOrNil(held)
	if coverID != nil {
		wanted = append(wanted, *coverID)
	}
	if len(wanted) == 0 {
		return nil, map[uuid.UUID]format.ExportMedia{}, nil
	}

	rows, err := q.Query(ctx, `
		select media.id, media.blob_id, blob.byte_size
		  from asset_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.asset_id = $1 and media.is_current and media.id = any($2)
	`, subject.assetID, wanted)
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
	private := subject.lifecycle == LifecycleDraft
	for mediaID, blobID := range blobs {
		picture, err := s.readBlob(ctx, blobID)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: picture %s: %w", ErrExportImageUnreadable, mediaID, err)
		}
		picture.URL = s.exportMediaURL(mediaID, private)
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

func (s *Service) exportMediaURL(mediaID uuid.UUID, private bool) string {
	return s.siteURL + s.variantURL(mediaID, "detail", false, private)
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

func downloadFilename(name, label, extension string) string {
	parts := make([]string, 0, 2)
	for _, part := range []string{name, label} {
		if slug := filenameSlug(part); slug != "" {
			parts = append(parts, slug)
		}
	}
	if len(parts) == 0 {
		return "download" + extension
	}
	return strings.Join(parts, "-") + extension
}

func filenameSlug(text string) string {
	slug := make([]rune, 0, len(text))
	for _, letter := range strings.ToLower(text) {
		switch {
		case letter >= 'a' && letter <= 'z', letter >= '0' && letter <= '9':
			slug = append(slug, letter)
		case len(slug) > 0 && slug[len(slug)-1] != '-':
			slug = append(slug, '-')
		}
	}
	return strings.Trim(string(slug), "-")
}

type OriginalUpload struct {
	Label     string
	MediaType string
	ArrivedAt time.Time
}
