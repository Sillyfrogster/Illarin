package work

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrVersionRequired = errors.New("a reviewed working-copy version is required")

type Candidate struct {
	Version      int64
	SavedVersion int64
}

type VersionConflict struct {
	CurrentVersion int64
}

func (e *VersionConflict) Error() string {
	return "This asset changed since you opened it. Keep your edits and reload the working copy to reconcile them."
}

func (c *Candidate) Lock(ctx context.Context, tx pgx.Tx, ownerID, workID uuid.UUID) (string, error) {
	workType, err := LockEditable(ctx, tx, ownerID, workID)
	if err != nil {
		return "", err
	}
	if c == nil || c.Version < 1 {
		return "", ErrVersionRequired
	}
	var current int64
	if err := tx.QueryRow(ctx, `select working_copy_version from works where id = $1`, workID).Scan(&current); err != nil {
		return "", fmt.Errorf("read working-copy version: %w", err)
	}
	if c.Version != current {
		return "", &VersionConflict{CurrentVersion: current}
	}
	return workType, nil
}

func (c *Candidate) Commit(ctx context.Context, tx pgx.Tx, workID uuid.UUID) error {
	var version int64
	if err := tx.QueryRow(ctx, `update works set working_copy_version = working_copy_version + 1 where id = $1 returning working_copy_version`, workID).Scan(&version); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	c.SavedVersion = version
	return nil
}

func LockEditable(
	ctx context.Context,
	tx pgx.Tx,
	ownerID uuid.UUID,
	workID uuid.UUID,
) (string, error) {
	var workType string
	var withheld bool
	err := tx.QueryRow(ctx, `
		select type, withheld_at is not null
		  from works
		 where id = $1 and owner_id = $2 and deleted_at is null
		 for update
	`, workID, ownerID).Scan(&workType, &withheld)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read block owner: %w", err)
	}
	if withheld {
		return "", ErrWorkFrozen
	}
	return workType, nil
}
