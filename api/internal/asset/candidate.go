package asset

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

func (c *Candidate) Lock(ctx context.Context, tx pgx.Tx, ownerID, assetID uuid.UUID) (string, error) {
	kind, err := lockEditableAsset(ctx, tx, ownerID, assetID)
	if err != nil {
		return "", err
	}
	if c == nil || c.Version < 1 {
		return "", ErrVersionRequired
	}
	var current int64
	if err := tx.QueryRow(ctx, `select working_copy_version from assets where id = $1`, assetID).Scan(&current); err != nil {
		return "", fmt.Errorf("read working-copy version: %w", err)
	}
	if c.Version != current {
		return "", &VersionConflict{CurrentVersion: current}
	}
	return kind, nil
}

func (c *Candidate) commit(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) error {
	var version int64
	if err := tx.QueryRow(ctx, `update assets set working_copy_version = working_copy_version + 1 where id = $1 returning working_copy_version`, assetID).Scan(&version); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	c.SavedVersion = version
	return nil
}
