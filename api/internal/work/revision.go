package work

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type revisionRow struct {
	Revision   int
	BlobID     uuid.UUID
	MediaType  string
	Format     string
	Identifier string
}

func insertRevision(ctx context.Context, tx pgx.Tx, id, assetID uuid.UUID, row revisionRow) error {
	queries := db.New(tx)
	params := db.InsertRevisionParams{
		ID:         uuidToPgtype(id),
		AssetID:    uuidToPgtype(assetID),
		Revision:   int32(row.Revision),
		BlobID:     uuidToPgtype(row.BlobID),
		MediaType:  row.MediaType,
		Format:     row.Format,
		Identifier: row.Identifier,
	}
	if err := queries.InsertRevision(ctx, params); err != nil {
		return fmt.Errorf("insert revision: %w", err)
	}
	return nil
}

func setCurrentRevision(ctx context.Context, tx pgx.Tx, assetID, revisionID uuid.UUID) error {
	queries := db.New(tx)
	params := db.SetCurrentRevisionParams{
		ID:                uuidToPgtype(assetID),
		CurrentRevisionID: uuidToPgtype(revisionID),
	}
	if err := queries.SetCurrentRevision(ctx, params); err != nil {
		return fmt.Errorf("set current revision: %w", err)
	}
	return nil
}

type RevisionLocation struct {
	AssetID    uuid.UUID
	RevisionID uuid.UUID
	BlobID     uuid.UUID
	MediaType  string
	OwnerID    *uuid.UUID
}

func CurrentRevisionLocation(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) (RevisionLocation, error) {
	queries := db.New(q)
	row, err := queries.CurrentRevisionLocation(ctx, db.CurrentRevisionLocationParams{
		ID: uuidToPgtype(assetID), ViewerID: uuidToNullable(viewerID),
	})
	if err != nil {
		return RevisionLocation{}, err
	}
	var ownerID *uuid.UUID
	if row.OwnerID.Valid {
		owner := uuidFromPgtype(row.OwnerID)
		ownerID = &owner
	}
	return RevisionLocation{
		AssetID: uuidFromPgtype(row.AssetID), RevisionID: uuidFromPgtype(row.RevisionID),
		BlobID: uuidFromPgtype(row.BlobID), MediaType: row.MediaType,
		OwnerID: ownerID,
	}, nil
}

func setCoverMedia(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, mediaID *uuid.UUID) error {
	if _, err := tx.Exec(ctx,
		`update assets set cover_media_id = $2 where id = $1`, assetID, mediaID,
	); err != nil {
		return fmt.Errorf("set cover media: %w", err)
	}
	return nil
}

func setAlternateCoverMedia(ctx context.Context, tx pgx.Tx, assetID, mediaID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update assets asset
		   set cover_media_id = $2
		 where asset.id = $1
		   and (
		       asset.cover_media_id is null
		       or exists (
		           select 1
		             from asset_media cover
		            where cover.id = asset.cover_media_id
		              and cover.asset_id = asset.id
		              and cover.role = 'avatar_alt'
		       )
		   )
	`, assetID, mediaID); err != nil {
		return fmt.Errorf("set alternate cover media: %w", err)
	}
	return nil
}

func clearSupersededCover(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		update assets asset
		   set cover_media_id = null
		 where asset.id = $1
		   and exists (
		       select 1
		         from asset_media media
		        where media.id = asset.cover_media_id
		          and media.asset_id = asset.id
		          and not media.is_current
		   )
	`, assetID); err != nil {
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

// Revision is an uploaded original file and the pictures read out of it
type Revision struct {
	AssetID    uuid.UUID
	Number     int
	BlobID     uuid.UUID
	MediaType  string
	Format     string
	Identifier string
	Media      []PreparedMedia
}

// RecordRevision makes an uploaded file the work's current original and replaces the pictures read from the one before
func RecordRevision(ctx context.Context, tx pgx.Tx, revision Revision) (uuid.UUID, error) {
	revisionID := uuid.New()
	if err := insertRevision(ctx, tx, revisionID, revision.AssetID, revisionRow{
		Revision: revision.Number, BlobID: revision.BlobID, MediaType: revision.MediaType,
		Format: revision.Format, Identifier: revision.Identifier,
	}); err != nil {
		return uuid.Nil, err
	}
	if err := supersedeExtractedMedia(ctx, tx, revision.AssetID); err != nil {
		return uuid.Nil, err
	}
	if err := insertAssetMedia(ctx, tx, revision.AssetID, revision.Media); err != nil {
		return uuid.Nil, err
	}
	if err := setCurrentRevision(ctx, tx, revision.AssetID, revisionID); err != nil {
		return uuid.Nil, err
	}
	if coverID := avatarMedia(revision.Media); coverID != nil {
		return revisionID, setCoverMedia(ctx, tx, revision.AssetID, coverID)
	}
	return revisionID, clearSupersededCover(ctx, tx, revision.AssetID)
}
