package work

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PreservedNamespace struct {
	Name  string
	Bytes int
}

func (s *Service) PreservedNamespaces(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
) ([]PreservedNamespace, error) {
	originFormat, err := s.preservedAssetOrigin(ctx, ownerID, assetID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select namespace, sum(length(payload::text))::bigint, min(payload::text)
		  from asset_preserved_data
		 where asset_id = $1
		 group by namespace
		 order by namespace
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("read preserved namespaces: %w", err)
	}
	defer rows.Close()

	declaration, declared := s.reg.Declaration(originFormat)
	found := make([]PreservedNamespace, 0)
	for rows.Next() {
		var name, sample string
		var size int64
		if err := rows.Scan(&name, &size, &sample); err != nil {
			return nil, fmt.Errorf("read preserved namespace: %w", err)
		}
		if declared && declaration.RecordsNothing(name, []byte(sample)) {
			continue
		}
		found = append(found, PreservedNamespace{Name: name, Bytes: int(size)})
	}
	return found, rows.Err()
}

func (s *Service) DeletePreservedNamespace(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
	namespace string,
	candidate *Candidate,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, assetID); err != nil {
		return err
	}
	fingerprint, err := s.contentFingerprint(ctx, tx, assetID)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		delete from asset_preserved_data where asset_id = $1 and namespace = $2
	`, assetID, namespace)
	if err != nil {
		return fmt.Errorf("delete preserved %s: %w", namespace, err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := s.moveContentGeneration(ctx, tx, assetID, fingerprint); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, assetID)
}

func (s *Service) preservedAssetOrigin(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
) (string, error) {
	var origin *string
	err := s.pool.QueryRow(ctx, `
		select origin_format
		  from assets
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, assetID, ownerID).Scan(&origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read asset origin: %w", err)
	}
	if origin == nil {
		return "", nil
	}
	return *origin, nil
}
