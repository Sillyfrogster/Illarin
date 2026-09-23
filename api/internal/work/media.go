package work

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrMediaNotFound    = errors.New("media not found")
	ErrInvalidMediaRole = errors.New("invalid media role")
)

type MediaRole = mediaproc.Role

const (
	MediaAvatar           = mediaproc.Avatar
	MediaExpression       = mediaproc.Expression
	MediaGallery          = mediaproc.Gallery
	MediaAvatarAlt        = mediaproc.AvatarAlt
	MediaPerspectiveLayer = mediaproc.PerspectiveLayer
	MediaPackItem         = mediaproc.PackItem
)

type Media struct {
	ID               uuid.UUID
	WorkID           uuid.UUID
	Role             MediaRole
	Width            int
	Height           int
	ImageSizeVersion uint32
}

type AddMediaInput struct {
	OwnerID uuid.UUID
	WorkID  uuid.UUID
	Role    MediaRole
	File    io.Reader
}

type MediaDownload struct {
	InternalRedirect string
	MediaType        string
	Private          bool
}

type MediaRequest struct {
	MediaID   uuid.UUID
	Size      string
	Version   uint32
	ViewerID  *uuid.UUID
	Expires   string
	Signature string
}

type PreparedMedia struct {
	ID          uuid.UUID
	BlobID      uuid.UUID
	Role        MediaRole
	ElementRole block.Role
	Name        string
	Width       int
	Height      int
	Seeded      bool
}

type sourceErrorReader struct {
	reader io.Reader
	err    error
}

func (r *sourceErrorReader) Read(payload []byte) (int, error) {
	count, err := r.reader.Read(payload)
	if err != nil && !errors.Is(err, io.EOF) {
		r.err = err
	}
	return count, err
}

func (s *Service) AddMedia(ctx context.Context, in AddMediaInput, candidate *Candidate) (Media, error) {
	if !in.Role.Valid() {
		return Media{}, ErrInvalidMediaRole
	}
	var takenDownAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, `
		select taken_down_at
		  from works
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, in.WorkID, in.OwnerID).Scan(&takenDownAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, fmt.Errorf("check media owner: %w", err)
	}
	if takenDownAt.Valid {
		return Media{}, ErrWorkFrozen
	}

	stored, prepared, err := s.media.Accept(ctx, in.File)
	if err != nil {
		return Media{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Media{}, fmt.Errorf("begin media addition: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.EnsureAccountStorage(ctx, tx, in.OwnerID, []uuid.UUID{stored.ID}); err != nil {
		return Media{}, err
	}
	if _, err := candidate.Lock(ctx, tx, in.OwnerID, in.WorkID); err != nil {
		return Media{}, err
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into work_media (id, work_id, role, width, height, blob_id)
		values ($1, $2, $3, $4, $5, $6)
	`, id, in.WorkID, in.Role, prepared.Width, prepared.Height, stored.ID)
	if err != nil {
		return Media{}, fmt.Errorf("record media: %w", err)
	}
	switch in.Role {
	case MediaAvatar:
		if err := supersedeCoverMedia(ctx, tx, in.WorkID, id); err != nil {
			return Media{}, err
		}
		if err := setCoverMedia(ctx, tx, in.WorkID, &id); err != nil {
			return Media{}, err
		}
	case MediaAvatarAlt:
		if err := setAlternateCoverMedia(ctx, tx, in.WorkID, id); err != nil {
			return Media{}, err
		}
	}
	if err := candidate.Commit(ctx, tx, in.WorkID); err != nil {
		return Media{}, fmt.Errorf("commit media addition: %w", err)
	}
	return Media{
		ID: id, WorkID: in.WorkID, Role: in.Role,
		Width: prepared.Width, Height: prepared.Height,
		ImageSizeVersion: mediaproc.ImageSizeVersion,
	}, nil
}

func (s *Service) ListMedia(ctx context.Context, workID uuid.UUID, viewerID *uuid.UUID) ([]Media, error) {
	var foundWorkID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`select id
		   from works
		  where id = $1 and deleted_at is null
		    and (lifecycle = 'published' or owner_id = $2)
		    and (taken_down_at is null or owner_id = $2)`, workID, viewerID,
	).Scan(&foundWorkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find work media: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		select id, work_id, role, width, height
		  from work_public.work_media
		 where work_id = $1 and is_current
		 order by created_at, id
	`, foundWorkID)
	if err != nil {
		return nil, fmt.Errorf("list work media: %w", err)
	}
	defer rows.Close()
	media := make([]Media, 0)
	for rows.Next() {
		var found Media
		var width, height pgtype.Int4
		if err := rows.Scan(
			&found.ID, &found.WorkID, &found.Role, &width, &height,
		); err != nil {
			return nil, fmt.Errorf("read work media: %w", err)
		}
		if !width.Valid || !height.Valid {
			return nil, fmt.Errorf("media %s has no native dimensions", found.ID)
		}
		found.Width = int(width.Int32)
		found.Height = int(height.Int32)
		found.ImageSizeVersion = mediaproc.ImageSizeVersion
		media = append(media, found)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list work media: %w", err)
	}
	return media, nil
}

func (s *Service) PrepareExtractedMedia(
	ctx context.Context,
	file format.Inspection,
	extracted []format.Media,
) ([]PreparedMedia, error) {
	prepared := make([]PreparedMedia, 0, len(extracted))
	for _, item := range extracted {
		role := item.Role
		if !role.Valid() {
			return nil, fmt.Errorf("format returned media role %q: %w", item.Role, ErrInvalidMediaRole)
		}
		source, err := file.OpenImage(ctx, item.ImageID)
		if err != nil {
			if errors.Is(err, format.ErrImageUnavailable) {
				continue
			}
			return nil, fmt.Errorf("open extracted media: %w", err)
		}
		tracked := &sourceErrorReader{reader: source}
		stored, err := s.store.Put(ctx, tracked)
		closeErr := source.Close()
		if localImageReadFailure(tracked.err) || localImageReadFailure(closeErr) {
			continue
		}
		if tracked.err != nil {
			return nil, fmt.Errorf("read extracted media: %w", tracked.err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close extracted media: %w", closeErr)
		}
		if err != nil {
			return nil, fmt.Errorf("store extracted media: %w", err)
		}
		image, err := s.media.Prepare(ctx, stored)
		if err != nil {
			if errors.Is(err, mediaproc.ErrUnsupportedImage) {
				continue
			}
			return nil, err
		}
		prepared = append(prepared, PreparedMedia{
			ID: uuid.New(), BlobID: stored.ID, Role: role,
			ElementRole: item.ElementRole, Name: item.Name,
			Width: image.Width, Height: image.Height,
		})
	}
	return prepared, nil
}

func localImageReadFailure(err error) bool {
	return err != nil && !errors.Is(err, format.ErrRangeRead) && !errors.Is(err, context.Canceled)
}

func ElementsForExtractedMedia(media []PreparedMedia) []block.Element {
	grouped := make(map[block.Role][]block.ImageItem)
	order := make([]block.Role, 0, 2)
	for _, item := range media {
		if item.ElementRole == "" {
			continue
		}
		if _, seen := grouped[item.ElementRole]; !seen {
			order = append(order, item.ElementRole)
		}
		grouped[item.ElementRole] = append(grouped[item.ElementRole], block.ImageItem{
			ID: block.NewItemID(), MediaID: item.ID, Name: item.Name,
		})
	}
	elements := make([]block.Element, 0, len(order))
	for _, role := range order {
		elements = append(elements, block.Element{
			ID: uuid.New(), Type: block.TypeImageSet, Role: role,
			Content: block.ImageSet{Images: grouped[role]},
		})
	}
	return elements
}

func insertWorkMedia(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	media []PreparedMedia,
) error {
	for _, item := range media {
		_, err := tx.Exec(ctx, `
			insert into work_media
			  (id, work_id, role, width, height, blob_id, is_extracted)
			values ($1, $2, $3, $4, $5, $6, $7)
		`, item.ID, workID, item.Role, item.Width, item.Height, item.BlobID, !item.Seeded)
		if err != nil {
			return fmt.Errorf("record extracted media: %w", err)
		}
	}
	return nil
}

// supersedeCoverMedia retires the picture a new display picture replaces.
func supersedeCoverMedia(ctx context.Context, tx pgx.Tx, workID, keep uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update work_media
		   set is_current = false
		 where work_id = $1 and id <> $2 and is_current
		   and role in ('avatar', 'avatar_alt')
	`, workID, keep); err != nil {
		return fmt.Errorf("supersede cover media: %w", err)
	}
	return nil
}

func supersedeExtractedMedia(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update work_media
		   set is_current = false
		 where work_id = $1 and is_extracted and is_current
	`, workID); err != nil {
		return fmt.Errorf("supersede extracted media: %w", err)
	}
	return nil
}

func MediaUploadFailure(err error) format.FailureReason {
	switch {
	case errors.Is(err, mediaproc.ErrImageTooLarge):
		return format.FailureSafetyViolation
	case errors.Is(err, mediaproc.ErrUnsupportedImage):
		return format.FailureMalformedInput
	default:
		return format.FailureInternal
	}
}

func (s *Service) ImageSize(ctx context.Context, in MediaRequest) (MediaDownload, error) {
	_, ordinary := mediaproc.ImageSizeByName(in.Size)
	_, composed := mediaproc.LinkCardByName(in.Size)
	if (!ordinary && !composed) || in.Version != mediaproc.ImageSizeVersion {
		return MediaDownload{}, ErrMediaNotFound
	}
	size, version := in.Size, in.Version
	var blobID uuid.UUID
	var digestBytes []byte
	var private, owner, draft bool
	err := s.pool.QueryRow(ctx, `
		select media.blob_id, blob.sha256,
		       work.lifecycle = 'draft' or not exists (
		           select 1 from work_version_media recorded
		           join work_versions version on version.id = recorded.version_id
		           where recorded.work_id = work.id and recorded.media_id = media.id
		             and version.withdrawn_at is null
		       ), coalesce(work.owner_id = $2, false), work.lifecycle = 'draft'
		  from work_media media
		  join works work on work.id = media.work_id
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
		   and work.deleted_at is null
		   and (work.taken_down_at is null or work.owner_id = $2)
	`, in.MediaID, in.ViewerID).Scan(&blobID, &digestBytes, &private, &owner, &draft)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaDownload{}, ErrMediaNotFound
	}
	if err != nil {
		return MediaDownload{}, fmt.Errorf("find media: %w", err)
	}
	if private {
		path := fmt.Sprintf("/media/%s/%s/%d", in.MediaID, size, version)
		if (!draft && !owner) || !s.signer.Valid(path, in.Expires, in.Signature, s.now()) {
			return MediaDownload{}, ErrMediaNotFound
		}
	}
	if len(digestBytes) != sha256.Size {
		return MediaDownload{}, fmt.Errorf("media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, size, version)
	if err != nil {
		return MediaDownload{}, err
	}
	return MediaDownload{
		InternalRedirect: redirect,
		MediaType:        s.media.ImageSizeMediaType(),
		Private:          private,
	}, nil
}

// ImageAddress is where a picture size is served, signed when only its owner may see it
func (s *Service) ImageAddress(mediaID uuid.UUID, size string, blurred, private bool) string {
	if blurred {
		size += "_blurred"
	}
	path := fmt.Sprintf("/media/%s/%s/%d", mediaID, size, mediaproc.ImageSizeVersion)
	if !private {
		return path
	}
	return s.signer.Sign(path, s.now())
}

// ExportMediaURL is the full address a written file or a connected app fetches a picture from
func (s *Service) ExportMediaURL(mediaID uuid.UUID, private bool) string {
	return s.siteURL + s.ImageAddress(mediaID, "detail", false, private)
}
