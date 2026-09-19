package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrTargetNotOffered = errors.New("that download is not offered for this work")

var ErrLinkedInstallOnly = errors.New("this work is linked-install-only")

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
	workID         uuid.UUID
	workType       string
	name           string
	origin         string
	header         format.Header
	blocks         []block.Block
	cover          *uuid.UUID
	ownerID        *uuid.UUID
	lifecycle      work.Lifecycle
	originalFileID *uuid.UUID
	gallery        *GallerySelection
	recorded       *work.FullVersion
}

func (s *Service) OpenExport(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
	target string,
	gallery *GallerySelection,
) (Export, error) {
	tx, err := s.works.BeginReadSnapshot(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("begin export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, err := s.exportSubject(ctx, tx, workID, viewerID)
	if err != nil {
		return Export{}, err
	}
	subject.gallery = gallery
	if err := private.ApplyPublishedPolicy(ctx, tx, workID, subject.blocks); err != nil {
		return Export{}, err
	}
	apps, err := private.Apps(ctx, tx, workID)
	if err != nil {
		return Export{}, err
	}
	if len(apps) > 0 || private.HasPromptFragments(subject.blocks) {
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
	written format.MainFile,
	target, label string,
	viewerID *uuid.UUID,
) Export {
	export := Export{
		Body: written.Body, MediaType: written.MediaType, Target: target,
		Filename: format.Filename(subject.name, subject.versionName(), label, written.Extension),
	}
	if subject.lifecycle == work.LifecyclePublished {
		event := newEvent(subject.workID, subject.originalFileID, target, subject.ownerID, viewerID)
		export.Event = &event
	}
	return export
}

func (subject exportSubject) versionName() string {
	if subject.recorded == nil {
		return ""
	}
	return fmt.Sprintf("version %d", subject.recorded.Number)
}

func (s *Service) OpenExportForLinkedInstance(
	ctx context.Context,
	workID uuid.UUID,
	target string,
) (Export, error) {
	tx, err := s.works.BeginReadSnapshot(ctx)
	if err != nil {
		return Export{}, fmt.Errorf("begin linked export snapshot: %w", err)
	}
	defer tx.Rollback(ctx)

	subject, err := s.exportSubject(ctx, tx, workID, nil)
	if err != nil {
		return Export{}, err
	}
	apps, err := private.Apps(ctx, tx, workID)
	if err != nil {
		return Export{}, err
	}
	if len(apps) > 0 && !private.AllowsTarget(apps, subject.workType, target) {
		return Export{}, ErrTargetNotOffered
	}
	if len(apps) == 0 {
		if err := private.ApplyPublishedPolicy(ctx, tx, workID, subject.blocks); err != nil {
			return Export{}, err
		}
		if private.HasPromptFragments(subject.blocks) {
			return Export{}, ErrLinkedInstallOnly
		}
	}
	if err := private.RestorePromptFragments(ctx, tx, workID, subject.blocks); err != nil {
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
) (format.MainFile, error) {
	travelling := subject.travellingElements()
	work := format.ExportWork{
		Type: subject.workType, Header: subject.header, Elements: travelling,
	}
	cover, images, err := s.exportImages(ctx, q, subject, travelling)
	if err != nil {
		return format.MainFile{}, err
	}
	work.Cover, work.Images = cover, images
	work.Preserved, err = s.travellingPreservedData(ctx, q, subject, writer.ID())
	if err != nil {
		return format.MainFile{}, err
	}
	if declaration, known := s.reg.Declaration(writer.ID()); known && declaration.KeepsUpload {
		work.Upload, err = s.readUpload(ctx, q, subject)
		if err != nil {
			return format.MainFile{}, err
		}
	}
	written, err := writer.Write(ctx, work)
	if err != nil {
		return format.MainFile{}, fmt.Errorf("write %s: %w", writer.ID(), err)
	}
	if len(written.Body) > MaxExportBytes {
		return format.MainFile{}, ErrExportTooLarge
	}
	return written, nil
}

// readUpload reads the file behind the version being exported.
func (s *Service) readUpload(ctx context.Context, q db.DBTX, subject exportSubject) ([]byte, error) {
	if subject.originalFileID == nil {
		return nil, fmt.Errorf("%w: no upload is recorded for this version", ErrTargetNotOffered)
	}
	var blobID uuid.UUID
	if err := q.QueryRow(ctx,
		`select blob_id from work_original_files where id = $1 and work_id = $2`,
		*subject.originalFileID, subject.workID,
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
		Type: subject.workType, Origin: subject.origin, Elements: subject.elements(),
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
	workID uuid.UUID,
	viewerID *uuid.UUID,
) (exportSubject, error) {
	var subject exportSubject
	var origin pgtype.Text
	var ownerID, originalFileID, cover pgtype.UUID
	err := q.QueryRow(ctx, `
		select work.type, work.name, work.blurb, work.origin_format, work.lifecycle,
		       work.work_version, work.credited_author, work.nickname,
		       work.owner_id, work.original_file_id, work.cover_media_id
		  from works work
		 where work.id = $1 and work.deleted_at is null
		   and (work.lifecycle = 'published' or work.owner_id = $2)
		   and (work.withheld_at is null or work.owner_id = $2)
	`, workID, viewerID).Scan(
		&subject.workType, &subject.name, &subject.header.Blurb, &origin, &subject.lifecycle,
		&subject.header.WorkVersion, &subject.header.CreditedAuthor,
		&subject.header.Nickname, &ownerID, &originalFileID, &cover,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return exportSubject{}, work.ErrNotFound
	}
	if err != nil {
		return exportSubject{}, fmt.Errorf("read the work to export: %w", err)
	}
	subject.workID = workID
	subject.header.Name = subject.name
	subject.origin = origin.String
	subject.cover = uuidOrNil(cover)
	subject.ownerID = uuidOrNil(ownerID)
	subject.originalFileID = uuidOrNil(originalFileID)
	subject.blocks, err = block.Read(ctx, q, workID)
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
		select owner_type, owner_id, namespace, payload
		  from work_preserved_data
		 where work_id = $1
		 order by namespace, owner_id
	`, subject.workID)
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

	rows, err := q.Query(ctx, subject.pictureQuery(), subject.workID, wanted, subject.versionID())
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
	private := subject.lifecycle == work.LifecycleDraft
	for mediaID, blobID := range blobs {
		picture, err := s.readBlob(ctx, blobID)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: picture %s: %w", ErrExportImageUnreadable, mediaID, err)
		}
		picture.URL = s.works.ExportMediaURL(mediaID, private)
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
			  from work_media media
			  join blobs blob on blob.id = media.blob_id
			 where media.work_id = $1 and media.is_current and media.id = any($2)
			   and $3::uuid is null`
	}
	return `
		select media.id, media.blob_id, blob.byte_size
		  from public.work_media media
		  join public.work_version_media kept on kept.media_id = media.id
		  join public.blobs blob on blob.id = media.blob_id
		 where media.work_id = $1 and media.id = any($2) and kept.version_id = $3`
}

func (subject exportSubject) versionID() *uuid.UUID {
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
