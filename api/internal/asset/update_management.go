package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const MaxWithdrawalExplanationRunes = 1000

var (
	ErrWithdrawalExplanationRequired = errors.New("a withdrawal needs a public explanation")
	ErrWithdrawalExplanationTooLong  = errors.New("the withdrawal explanation is too long")
	ErrCurrentVersionWithdrawal      = errors.New("publish a replacement before withdrawing the current version")
	ErrVersionAlreadyWithdrawn       = errors.New("the version is already withdrawn")
)

func (s *Service) RestoreVersion(ctx context.Context, ownerID, assetID uuid.UUID, number int, candidate *Candidate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	kind, err := candidate.Lock(ctx, tx, ownerID, assetID)
	if err != nil {
		return err
	}
	recorded, err := readVersion(ctx, tx, assetID, number)
	if err != nil {
		return err
	}
	if recorded.kind != kind {
		return ErrNotFound
	}
	if err := protected.PrepareRestoration(ctx, tx, assetID, recorded.ID, recorded.protectedPayloads, recorded.blocks); err != nil {
		return err
	}

	metadata := recorded.metadata
	_, err = tx.Exec(ctx, `
		update assets set name = $2, blurb = $3, tags = $4, is_nsfw = $5,
		       credited_author = $6, nickname = $7, asset_version = $8,
		       origin_format = $9, current_revision_id = $10, cover_media_id = $11,
		       updated_at = now()
		 where id = $1
	`, assetID, metadata.Name, metadata.Blurb, metadata.Tags, metadata.IsNSFW,
		metadata.CreditedAuthor, metadata.Nickname, metadata.AssetVersion,
		recorded.origin, recorded.sourceRevisionID, metadata.Cover)
	if err != nil {
		return fmt.Errorf("restore the recorded header: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		update asset_media media set is_current = exists (
			select 1 from asset_snapshot_media kept
			 where kept.snapshot_id = $2 and kept.media_id = media.id)
		 where media.asset_id = $1
	`, assetID, recorded.ID); err != nil {
		return fmt.Errorf("restore the recorded pictures: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from asset_blocks where asset_id = $1`, assetID); err != nil {
		return fmt.Errorf("replace the working-copy blocks: %w", err)
	}
	if err := block.Insert(ctx, tx, assetID, recorded.blocks); err != nil {
		return fmt.Errorf("restore the recorded blocks: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from asset_preserved_data where asset_id = $1`, assetID); err != nil {
		return fmt.Errorf("replace the preserved data: %w", err)
	}
	for _, kept := range recorded.preserved {
		if _, err := tx.Exec(ctx, `
			insert into asset_preserved_data (id, asset_id, owner_kind, owner_id, namespace, payload)
			values ($1, $2, $3, $4, $5, $6::jsonb)
		`, kept.ID, assetID, kept.Owner, kept.OwnerID, kept.Namespace, kept.Payload); err != nil {
			return fmt.Errorf("restore preserved data: %w", err)
		}
	}
	if err := s.writeProjections(ctx, tx, assetID); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, assetID)
}

func (s *Service) CorrectVersionNotes(ctx context.Context, ownerID, assetID uuid.UUID, number int, summary, notes string) error {
	summary = strings.TrimSpace(summary)
	notes = strings.TrimSpace(notes)
	if summary == "" {
		return ErrSummaryRequired
	}
	if utf8.RuneCountInString(summary) > MaxSummaryRunes || utf8.RuneCountInString(notes) > MaxNotesRunes {
		return ErrSummaryTooLong
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := lockEditableAsset(ctx, tx, ownerID, assetID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		update asset_snapshots set summary = $3, notes = $4, notes_edited_at = now()
		 where asset_id = $1 and number = $2
	`, assetID, number, summary, notes)
	if err != nil {
		return fmt.Errorf("correct the update notes: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Service) WithdrawVersion(ctx context.Context, ownerID, assetID uuid.UUID, number int, explanation string) error {
	explanation = strings.TrimSpace(explanation)
	if explanation == "" {
		return ErrWithdrawalExplanationRequired
	}
	if utf8.RuneCountInString(explanation) > MaxWithdrawalExplanationRunes {
		return ErrWithdrawalExplanationTooLong
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := lockEditableAsset(ctx, tx, ownerID, assetID); err != nil {
		return err
	}
	var current, chosen uuid.UUID
	var withdrawn bool
	err = tx.QueryRow(ctx, `
		select asset.published_snapshot_id, snapshot.id, snapshot.withdrawn_at is not null
		  from assets asset join asset_snapshots snapshot on snapshot.asset_id = asset.id
		 where asset.id = $1 and snapshot.number = $2 for update of asset, snapshot
	`, assetID, number).Scan(&current, &chosen, &withdrawn)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read the version to withdraw: %w", err)
	}
	if chosen == current {
		return ErrCurrentVersionWithdrawal
	}
	if withdrawn {
		return ErrVersionAlreadyWithdrawn
	}
	if _, err := tx.Exec(ctx, `
		update asset_snapshots set withdrawn_at = now(), withdrawal_explanation = $2 where id = $1
	`, chosen, explanation); err != nil {
		return fmt.Errorf("withdraw the version: %w", err)
	}
	return tx.Commit(ctx)
}
