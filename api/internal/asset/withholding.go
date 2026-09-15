package asset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalidWithholdReason = errors.New("invalid withhold reason")

func (s *Service) Withhold(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrInvalidWithholdReason
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin withholding an asset: %w", err)
	}
	defer tx.Rollback(ctx)
	withheld, err := db.New(tx).WithholdAsset(ctx, db.WithholdAssetParams{
		ID:             uuidToPgtype(id),
		WithheldBy:     uuidToPgtype(actorID),
		WithheldReason: textToNullable(&reason),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("withhold asset: %w", err)
	}
	if err := tellOwner(ctx, tx, id, withheld.OwnerID, notification.AssetWithheld, notification.Words{
		AssetName: withheld.PublicName, Reason: reason,
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
	cleared, err := db.New(tx).ClearAssetWithhold(ctx, uuidToPgtype(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("clear asset withhold: %w", err)
	}
	if err := tellOwner(ctx, tx, id, cleared.OwnerID, notification.AssetRestored, notification.Words{
		AssetName: cleared.PublicName,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit clearing the withhold: %w", err)
	}
	return nil
}

// tellOwner records a staff decision for the asset's owner and never names the staff member who made it.
func tellOwner(
	ctx context.Context, tx pgx.Tx, assetID uuid.UUID, owner pgtype.UUID,
	kind notification.Type, words notification.Words,
) error {
	if !owner.Valid {
		return nil
	}
	return notification.Record(ctx, tx, notification.Event{
		Type: kind, Account: uuid.UUID(owner.Bytes), Asset: &assetID, Words: words,
	})
}
