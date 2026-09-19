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
	workID uuid.UUID,
) ([]PreservedNamespace, error) {
	originFormat, err := s.preservedWorkOrigin(ctx, ownerID, workID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select namespace, sum(length(payload::text))::bigint, min(payload::text)
		  from work_preserved_data
		 where work_id = $1
		 group by namespace
		 order by namespace
	`, workID)
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
	workID uuid.UUID,
	namespace string,
	candidate *Candidate,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		delete from work_preserved_data where work_id = $1 and namespace = $2
	`, workID, namespace)
	if err != nil {
		return fmt.Errorf("delete preserved %s: %w", namespace, err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return candidate.Commit(ctx, tx, workID)
}

func (s *Service) preservedWorkOrigin(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) (string, error) {
	var origin *string
	err := s.pool.QueryRow(ctx, `
		select origin_format
		  from works
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, workID, ownerID).Scan(&origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read work origin: %w", err)
	}
	if origin == nil {
		return "", nil
	}
	return *origin, nil
}
