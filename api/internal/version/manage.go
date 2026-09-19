package version

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
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

func (s *Service) RestoreVersion(ctx context.Context, ownerID, workID uuid.UUID, number int, candidate *work.Candidate) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	workType, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return err
	}
	recorded, err := work.ReadVersion(ctx, tx, workID, number)
	if err != nil {
		return err
	}
	if recorded.Type != workType {
		return work.ErrNotFound
	}
	if err := private.PrepareRestoration(ctx, tx, workID, recorded.ID, recorded.PrivatePrompts, recorded.Blocks); err != nil {
		return err
	}

	metadata := recorded.Metadata
	_, err = tx.Exec(ctx, `
		update works set name = $2, blurb = $3, tags = $4, is_nsfw = $5,
		       credited_author = $6, nickname = $7, work_version = $8,
		       origin_format = $9, original_file_id = $10, cover_media_id = $11,
		       updated_at = now()
		 where id = $1
	`, workID, metadata.Name, metadata.Blurb, metadata.Tags, metadata.IsNSFW,
		metadata.CreditedAuthor, metadata.Nickname, metadata.WorkVersion,
		recorded.Origin, recorded.OriginalFileID, metadata.Cover)
	if err != nil {
		return fmt.Errorf("restore the recorded header: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		update work_media media set is_current = exists (
			select 1 from work_version_media kept
			 where kept.version_id = $2 and kept.media_id = media.id)
		 where media.work_id = $1
	`, workID, recorded.ID); err != nil {
		return fmt.Errorf("restore the recorded pictures: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from work_blocks where work_id = $1`, workID); err != nil {
		return fmt.Errorf("replace the drafted-changes blocks: %w", err)
	}
	if err := block.Insert(ctx, tx, workID, recorded.Blocks); err != nil {
		return fmt.Errorf("restore the recorded blocks: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from work_preserved_data where work_id = $1`, workID); err != nil {
		return fmt.Errorf("replace the preserved data: %w", err)
	}
	for _, kept := range recorded.Preserved {
		if _, err := tx.Exec(ctx, `
			insert into work_preserved_data (id, work_id, owner_type, owner_id, namespace, payload)
			values ($1, $2, $3, $4, $5, $6::jsonb)
		`, kept.ID, workID, kept.Owner, kept.OwnerID, kept.Namespace, kept.Payload); err != nil {
			return fmt.Errorf("restore preserved data: %w", err)
		}
	}
	if err := s.writeSummary(ctx, tx, workID); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, workID)
}

func (s *Service) CorrectVersionNotes(ctx context.Context, ownerID, workID uuid.UUID, number int, summary, notes string) error {
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
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		update work_versions set summary = $3, notes = $4, notes_edited_at = now()
		 where work_id = $1 and number = $2
	`, workID, number, summary, notes)
	if err != nil {
		return fmt.Errorf("correct the version notes: %w", err)
	}
	if result.RowsAffected() != 1 {
		return work.ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Service) WithdrawVersion(ctx context.Context, ownerID, workID uuid.UUID, number int, explanation string) error {
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
	if _, err := work.LockEditable(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	var current, chosen uuid.UUID
	var withdrawn bool
	err = tx.QueryRow(ctx, `
		select work.published_version_id, version.id, version.withdrawn_at is not null
		  from works work join work_versions version on version.work_id = work.id
		 where work.id = $1 and version.number = $2 for update of work, version
	`, workID, number).Scan(&current, &chosen, &withdrawn)
	if errors.Is(err, pgx.ErrNoRows) {
		return work.ErrNotFound
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
		update work_versions set withdrawn_at = now(), withdrawal_explanation = $2 where id = $1
	`, chosen, explanation); err != nil {
		return fmt.Errorf("withdraw the version: %w", err)
	}
	return tx.Commit(ctx)
}
