package asset

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
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
	ID                uuid.UUID
	AssetID           uuid.UUID
	Role              MediaRole
	Width             int
	Height            int
	DerivativeVersion uint32
}

type AddMediaInput struct {
	OwnerID uuid.UUID
	AssetID uuid.UUID
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
	Variant   string
	Version   uint32
	ViewerID  *uuid.UUID
	Expires   string
	Signature string
}

type preparedMedia struct {
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
	var withheldAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, `
		select withheld_at
		  from assets
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, in.AssetID, in.OwnerID).Scan(&withheldAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Media{}, ErrMediaNotFound
	}
	if err != nil {
		return Media{}, fmt.Errorf("check media owner: %w", err)
	}
	if withheldAt.Valid {
		return Media{}, ErrAssetFrozen
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

	if err := s.ensureAccountStorage(ctx, tx, in.OwnerID, []uuid.UUID{stored.ID}); err != nil {
		return Media{}, err
	}
	if _, err := candidate.Lock(ctx, tx, in.OwnerID, in.AssetID); err != nil {
		return Media{}, err
	}
	fingerprint, err := s.contentFingerprint(ctx, tx, in.AssetID)
	if err != nil {
		return Media{}, err
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into asset_media (id, asset_id, role, width, height, blob_id)
		values ($1, $2, $3, $4, $5, $6)
	`, id, in.AssetID, in.Role, prepared.Width, prepared.Height, stored.ID)
	if err != nil {
		return Media{}, fmt.Errorf("record media: %w", err)
	}
	switch in.Role {
	case MediaAvatar:
		if err := supersedeCoverMedia(ctx, tx, in.AssetID, id); err != nil {
			return Media{}, err
		}
		if err := setCoverMedia(ctx, tx, in.AssetID, &id); err != nil {
			return Media{}, err
		}
	case MediaAvatarAlt:
		if err := setAlternateCoverMedia(ctx, tx, in.AssetID, id); err != nil {
			return Media{}, err
		}
	}
	if err := s.moveContentGeneration(ctx, tx, in.AssetID, fingerprint); err != nil {
		return Media{}, err
	}
	if err := candidate.Commit(ctx, tx, in.AssetID); err != nil {
		return Media{}, fmt.Errorf("commit media addition: %w", err)
	}
	return Media{
		ID: id, AssetID: in.AssetID, Role: in.Role,
		Width: prepared.Width, Height: prepared.Height,
		DerivativeVersion: mediaproc.DerivativeVersion,
	}, nil
}

func (s *Service) ListMedia(ctx context.Context, assetID uuid.UUID, viewerID *uuid.UUID) ([]Media, error) {
	var foundAssetID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`select id
		   from assets
		  where id = $1 and deleted_at is null
		    and (lifecycle = 'published' or owner_id = $2)
		    and (withheld_at is null or owner_id = $2)`, assetID, viewerID,
	).Scan(&foundAssetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find asset media: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		select id, asset_id, role, width, height
		  from asset_public.asset_media
		 where asset_id = $1 and is_current
		 order by created_at, id
	`, foundAssetID)
	if err != nil {
		return nil, fmt.Errorf("list asset media: %w", err)
	}
	defer rows.Close()
	media := make([]Media, 0)
	for rows.Next() {
		var found Media
		var width, height pgtype.Int4
		if err := rows.Scan(
			&found.ID, &found.AssetID, &found.Role, &width, &height,
		); err != nil {
			return nil, fmt.Errorf("read asset media: %w", err)
		}
		if !width.Valid || !height.Valid {
			return nil, fmt.Errorf("media %s has no native dimensions", found.ID)
		}
		found.Width = int(width.Int32)
		found.Height = int(height.Int32)
		found.DerivativeVersion = mediaproc.DerivativeVersion
		media = append(media, found)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list asset media: %w", err)
	}
	return media, nil
}

func (s *Service) prepareExtractedMedia(
	ctx context.Context,
	file probe.Inspection,
	extracted []format.Media,
) ([]preparedMedia, error) {
	prepared := make([]preparedMedia, 0, len(extracted))
	for _, item := range extracted {
		role := item.Role
		if !role.Valid() {
			return nil, fmt.Errorf("format returned media role %q: %w", item.Role, ErrInvalidMediaRole)
		}
		source, err := file.OpenImage(ctx, item.ImageID)
		if err != nil {
			if errors.Is(err, probe.ErrImageUnavailable) {
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
		prepared = append(prepared, preparedMedia{
			ID: uuid.New(), BlobID: stored.ID, Role: role,
			ElementRole: item.ElementRole, Name: item.Name,
			Width: image.Width, Height: image.Height,
		})
	}
	return prepared, nil
}

func localImageReadFailure(err error) bool {
	return err != nil && !errors.Is(err, probe.ErrRangeRead) && !errors.Is(err, context.Canceled)
}

func elementsForExtractedMedia(media []preparedMedia) []block.Element {
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

func insertAssetMedia(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	media []preparedMedia,
) error {
	for _, item := range media {
		_, err := tx.Exec(ctx, `
			insert into asset_media
			  (id, asset_id, role, width, height, blob_id, is_extracted)
			values ($1, $2, $3, $4, $5, $6, $7)
		`, item.ID, assetID, item.Role, item.Width, item.Height, item.BlobID, !item.Seeded)
		if err != nil {
			return fmt.Errorf("record extracted media: %w", err)
		}
	}
	return nil
}

// supersedeCoverMedia retires the picture a new display picture replaces.
func supersedeCoverMedia(ctx context.Context, tx pgx.Tx, assetID, keep uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update asset_media
		   set is_current = false
		 where asset_id = $1 and id <> $2 and is_current
		   and role in ('avatar', 'avatar_alt')
	`, assetID, keep); err != nil {
		return fmt.Errorf("supersede cover media: %w", err)
	}
	return nil
}

func supersedeExtractedMedia(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update asset_media
		   set is_current = false
		 where asset_id = $1 and is_extracted and is_current
	`, assetID); err != nil {
		return fmt.Errorf("supersede extracted media: %w", err)
	}
	return nil
}

func mediaIngestFailure(err error) format.FailureReason {
	switch {
	case errors.Is(err, mediaproc.ErrImageTooLarge):
		return format.FailureSafetyViolation
	case errors.Is(err, mediaproc.ErrUnsupportedImage):
		return format.FailureMalformedInput
	default:
		return format.FailureInternal
	}
}

func (s *Service) MediaVariant(ctx context.Context, in MediaRequest) (MediaDownload, error) {
	_, ordinary := mediaproc.VariantByName(in.Variant)
	_, composed := mediaproc.SocialPreviewByName(in.Variant)
	if (!ordinary && !composed) || in.Version != mediaproc.DerivativeVersion {
		return MediaDownload{}, ErrMediaNotFound
	}
	variant, version := in.Variant, in.Version
	var blobID uuid.UUID
	var digestBytes []byte
	var private, owner, draft bool
	err := s.pool.QueryRow(ctx, `
		select media.blob_id, blob.sha256,
		       asset.lifecycle = 'draft' or not exists (
		           select 1 from asset_snapshot_media recorded
		           join asset_snapshots snapshot on snapshot.id = recorded.snapshot_id
		           where recorded.asset_id = asset.id and recorded.media_id = media.id
		             and snapshot.withdrawn_at is null
		       ), coalesce(asset.owner_id = $2, false), asset.lifecycle = 'draft'
		  from asset_media media
		  join assets asset on asset.id = media.asset_id
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
		   and asset.deleted_at is null
		   and (asset.withheld_at is null or asset.owner_id = $2)
	`, in.MediaID, in.ViewerID).Scan(&blobID, &digestBytes, &private, &owner, &draft)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaDownload{}, ErrMediaNotFound
	}
	if err != nil {
		return MediaDownload{}, fmt.Errorf("find media: %w", err)
	}
	if private {
		path := fmt.Sprintf("/media/%s/%s/%d", in.MediaID, variant, version)
		if (!draft && !owner) || !s.signer.Valid(path, in.Expires, in.Signature, s.now()) {
			return MediaDownload{}, ErrMediaNotFound
		}
	}
	if len(digestBytes) != sha256.Size {
		return MediaDownload{}, fmt.Errorf("media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, variant, version)
	if err != nil {
		return MediaDownload{}, err
	}
	return MediaDownload{
		InternalRedirect: redirect,
		MediaType:        s.media.DerivativeType(),
		Private:          private,
	}, nil
}

// ImageAddress is where a picture variant is served, signed when only its owner may see it
func (s *Service) ImageAddress(mediaID uuid.UUID, variant string, blurred, private bool) string {
	if blurred {
		variant += "_blurred"
	}
	path := fmt.Sprintf("/media/%s/%s/%d", mediaID, variant, mediaproc.DerivativeVersion)
	if !private {
		return path
	}
	return s.signer.Sign(path, s.now())
}
