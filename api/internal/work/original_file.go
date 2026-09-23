package work

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type originalFileRow struct {
	Number     int
	BlobID     uuid.UUID
	MediaType  string
	Format     string
	Identifier string
	Filename   string
}

func insertOriginalFile(ctx context.Context, tx pgx.Tx, id, workID uuid.UUID, row originalFileRow) error {
	queries := db.New(tx)
	params := db.InsertOriginalFileParams{
		ID:         uuidToPgtype(id),
		WorkID:     uuidToPgtype(workID),
		Number:     int32(row.Number),
		BlobID:     uuidToPgtype(row.BlobID),
		MediaType:  row.MediaType,
		Format:     row.Format,
		Identifier: row.Identifier,
		Filename:   pgtype.Text{String: row.Filename, Valid: row.Filename != ""},
	}
	if err := queries.InsertOriginalFile(ctx, params); err != nil {
		return fmt.Errorf("insert the original file: %w", err)
	}
	return nil
}

func setOriginalFile(ctx context.Context, tx pgx.Tx, workID, originalFileID uuid.UUID) error {
	queries := db.New(tx)
	params := db.SetOriginalFileParams{
		ID:             uuidToPgtype(workID),
		OriginalFileID: uuidToPgtype(originalFileID),
	}
	if err := queries.SetOriginalFile(ctx, params); err != nil {
		return fmt.Errorf("set the original file: %w", err)
	}
	return nil
}

type OriginalFileLocation struct {
	WorkID         uuid.UUID
	OriginalFileID uuid.UUID
	BlobID         uuid.UUID
	MediaType      string
	Filename       string
	Format         string
	OwnerID        *uuid.UUID
}

// LocateOriginalFile finds the original file a download of the work hands over
func LocateOriginalFile(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
	viewerID *uuid.UUID,
) (OriginalFileLocation, error) {
	queries := db.New(q)
	row, err := queries.OriginalFileLocation(ctx, db.OriginalFileLocationParams{
		ID: uuidToPgtype(workID), ViewerID: uuidToNullable(viewerID),
	})
	if err != nil {
		return OriginalFileLocation{}, err
	}
	var ownerID *uuid.UUID
	if row.OwnerID.Valid {
		owner := uuidFromPgtype(row.OwnerID)
		ownerID = &owner
	}
	return OriginalFileLocation{
		WorkID: uuidFromPgtype(row.WorkID), OriginalFileID: uuidFromPgtype(row.OriginalFileID),
		BlobID: uuidFromPgtype(row.BlobID), MediaType: row.MediaType,
		Filename: row.Filename.String, Format: row.Format,
		OwnerID: ownerID,
	}, nil
}

func setCoverMedia(ctx context.Context, tx pgx.Tx, workID uuid.UUID, mediaID *uuid.UUID) error {
	if _, err := tx.Exec(ctx,
		`update works set cover_media_id = $2 where id = $1`, workID, mediaID,
	); err != nil {
		return fmt.Errorf("set cover media: %w", err)
	}
	return nil
}

func setAlternateCoverMedia(ctx context.Context, tx pgx.Tx, workID, mediaID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update works work
		   set cover_media_id = $2
		 where work.id = $1
		   and (
		       work.cover_media_id is null
		       or exists (
		           select 1
		             from work_media cover
		            where cover.id = work.cover_media_id
		              and cover.work_id = work.id
		              and cover.role = 'avatar_alt'
		       )
		   )
	`, workID, mediaID); err != nil {
		return fmt.Errorf("set alternate cover media: %w", err)
	}
	return nil
}

func clearSupersededCover(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update works work
		   set cover_media_id = null
		 where work.id = $1
		   and exists (
		       select 1
		         from work_media media
		        where media.id = work.cover_media_id
		          and media.work_id = work.id
		          and not media.is_current
		   )
	`, workID); err != nil {
		return fmt.Errorf("clear superseded cover: %w", err)
	}
	return nil
}

func avatarMedia(media []PreparedMedia) *uuid.UUID {
	var alternate *uuid.UUID
	for _, item := range media {
		if item.Role == MediaAvatar {
			id := item.ID
			return &id
		}
		if item.Role == MediaAvatarAlt && alternate == nil {
			id := item.ID
			alternate = &id
		}
	}
	return alternate
}

// OriginalFile is an uploaded file and the pictures read out of it
type OriginalFile struct {
	WorkID     uuid.UUID
	Number     int
	BlobID     uuid.UUID
	MediaType  string
	Format     string
	Identifier string
	Filename   string
	Media      []PreparedMedia
}

// RecordOriginalFile makes an uploaded file the work's current original file and replaces the pictures read from the one before
func RecordOriginalFile(ctx context.Context, tx pgx.Tx, original OriginalFile) (uuid.UUID, error) {
	originalFileID := uuid.New()
	if err := insertOriginalFile(ctx, tx, originalFileID, original.WorkID, originalFileRow{
		Number: original.Number, BlobID: original.BlobID, MediaType: original.MediaType,
		Format: original.Format, Identifier: original.Identifier, Filename: original.Filename,
	}); err != nil {
		return uuid.Nil, err
	}
	if err := supersedeExtractedMedia(ctx, tx, original.WorkID); err != nil {
		return uuid.Nil, err
	}
	if err := insertWorkMedia(ctx, tx, original.WorkID, original.Media); err != nil {
		return uuid.Nil, err
	}
	if err := setOriginalFile(ctx, tx, original.WorkID, originalFileID); err != nil {
		return uuid.Nil, err
	}
	if coverID := avatarMedia(original.Media); coverID != nil {
		return originalFileID, setCoverMedia(ctx, tx, original.WorkID, coverID)
	}
	return originalFileID, clearSupersededCover(ctx, tx, original.WorkID)
}
