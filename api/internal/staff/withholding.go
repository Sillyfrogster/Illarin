package staff

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) Withhold(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrInvalidWithholdReason
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin withholding a work: %w", err)
	}
	defer tx.Rollback(ctx)
	withheld, err := db.New(tx).WithholdWork(ctx, db.WithholdWorkParams{
		ID:             uuidToPgtype(id),
		WithheldBy:     uuidToPgtype(actorID),
		WithheldReason: textToPgtype(reason),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkNotFound
	}
	if err != nil {
		return fmt.Errorf("withhold work: %w", err)
	}
	if err := tellOwner(ctx, tx, id, withheld.OwnerID, notify.WorkWithheld, notify.Words{
		WorkName: withheld.PublicName, Reason: reason,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit the withhold: %w", err)
	}
	return nil
}

func (s *Service) ClearWithhold(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin clearing a withhold: %w", err)
	}
	defer tx.Rollback(ctx)
	cleared, err := db.New(tx).ClearWorkWithhold(ctx, uuidToPgtype(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWorkNotFound
	}
	if err != nil {
		return fmt.Errorf("clear work withhold: %w", err)
	}
	if err := tellOwner(ctx, tx, id, cleared.OwnerID, notify.WorkRestored, notify.Words{
		WorkName: cleared.PublicName,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit clearing the withhold: %w", err)
	}
	return nil
}

// tellOwner records a staff decision for the work's owner and never names the staff member who made it.
func tellOwner(
	ctx context.Context, tx pgx.Tx, workID uuid.UUID, owner pgtype.UUID,
	noticeType notify.Type, words notify.Words,
) error {
	if !owner.Valid {
		return nil
	}
	account := uuid.UUID(owner.Bytes)
	return notify.Record(ctx, tx, notify.Event{
		Type: noticeType, Account: &account, Work: &workID, Words: words,
	})
}
