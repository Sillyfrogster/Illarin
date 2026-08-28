package publication

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// markVariant is the one size a mark is published at.
const markVariant = "grid"

// markOwner is a table whose mark_media_id column points into publication_media.
type markOwner string

const (
	distinctionMarks markOwner = "profile_distinctions"
	appMarks         markOwner = "publication_apps"
)

// Mark is the Illarin-hosted image a badge or an app is shown with.
type Mark struct {
	MediaID           uuid.UUID
	Width             int
	Height            int
	DerivativeVersion uint32
}

// MarkURL addresses one mark on the byte path every image shares.
func MarkURL(mediaID uuid.UUID, version uint32) string {
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, markVariant, version)
}

// replaceMark points one row at a newly stored image and drops the one it had.
func (s *Service) replaceMark(
	ctx context.Context,
	tx pgx.Tx,
	owner markOwner,
	id, blobID uuid.UUID,
	width, height int,
) error {
	var superseded *uuid.UUID
	err := tx.QueryRow(ctx, fmt.Sprintf(`
		select mark_media_id from %s where id = $1 for update
	`, owner), id).Scan(&superseded)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read the mark being replaced: %w", err)
	}
	mediaID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into publication_media (id, blob_id, width, height) values ($1, $2, $3, $4)
	`, mediaID, blobID, width, height)
	if err != nil {
		return fmt.Errorf("record mark: %w", err)
	}
	_, err = tx.Exec(ctx, fmt.Sprintf(`
		update %s set mark_media_id = $2, updated_at = now() where id = $1
	`, owner), id, mediaID)
	if err != nil {
		return fmt.Errorf("point the row at its mark: %w", err)
	}
	if superseded != nil {
		if _, err := tx.Exec(ctx, `delete from publication_media where id = $1`, *superseded); err != nil {
			return fmt.Errorf("drop the superseded mark: %w", err)
		}
	}
	return nil
}

// dropMark takes the mark off one row and deletes the image it pointed at.
func (s *Service) dropMark(
	ctx context.Context,
	tx pgx.Tx,
	owner markOwner,
	id uuid.UUID,
) error {
	var held *uuid.UUID
	err := tx.QueryRow(ctx, fmt.Sprintf(`
		select mark_media_id from %s where id = $1 for update
	`, owner), id).Scan(&held)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read the mark being removed: %w", err)
	}
	if held == nil {
		return nil
	}
	_, err = tx.Exec(ctx, fmt.Sprintf(`
		update %s set mark_media_id = null, updated_at = now() where id = $1
	`, owner), id)
	if err != nil {
		return fmt.Errorf("take the mark off the row: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from publication_media where id = $1`, *held); err != nil {
		return fmt.Errorf("drop the removed mark: %w", err)
	}
	return nil
}

// MarkVariant serves one size of a mark. Every mark is public.
func (s *Service) MarkVariant(
	ctx context.Context,
	mediaID uuid.UUID,
	variant string,
	version uint32,
) (string, string, error) {
	if _, known := mediaproc.VariantByName(variant); !known || version != mediaproc.DerivativeVersion {
		return "", "", ErrNotFound
	}
	var blobID uuid.UUID
	var digestBytes []byte
	err := s.pool.QueryRow(ctx, `
		select media.blob_id, blob.sha256
		  from publication_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, mediaID).Scan(&blobID, &digestBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("find publication media: %w", err)
	}
	if len(digestBytes) != sha256.Size {
		return "", "", fmt.Errorf("publication media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, variant, version)
	if err != nil {
		return "", "", err
	}
	return redirect, s.media.DerivativeType(), nil
}

func scanMark(markID *uuid.UUID, width, height *int) *Mark {
	if markID == nil || width == nil || height == nil {
		return nil
	}
	return &Mark{
		MediaID:           *markID,
		Width:             *width,
		Height:            *height,
		DerivativeVersion: mediaproc.DerivativeVersion,
	}
}
