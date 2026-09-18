package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) AcceptRevision(ctx context.Context, in RevisionInput, candidate *work.Candidate) (Operation, error) {
	var withheldAt pgtype.Timestamptz
	err := s.pool.QueryRow(ctx, `
		select withheld_at
		  from assets
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, in.AssetID, in.OwnerID).Scan(&withheldAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, work.ErrNotFound
	}
	if err != nil {
		return Operation{}, fmt.Errorf("check revision owner: %w", err)
	}
	if withheldAt.Valid {
		return Operation{}, work.ErrAssetFrozen
	}

	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return Operation{}, fmt.Errorf("store revision: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Operation{}, fmt.Errorf("begin revision acceptance: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.assets.EnsureAccountStorage(ctx, tx, in.OwnerID, []uuid.UUID{stored.ID}); err != nil {
		return Operation{}, err
	}
	if _, err := candidate.Lock(ctx, tx, in.OwnerID, in.AssetID); err != nil {
		return Operation{}, err
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into ingest_operations
			(id, owner_id, blob_id, filename, status, target_asset_id, candidate_version)
		values ($1, $2, $3, $4, 'pending', $5, $6)
	`, id, in.OwnerID, stored.ID, in.Filename, in.AssetID, candidate.Version)
	if err != nil {
		return Operation{}, fmt.Errorf("record revision ingest: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Operation{}, fmt.Errorf("commit revision acceptance: %w", err)
	}
	return Operation{ID: id, Status: IngestPending}, nil
}
