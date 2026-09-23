package work

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PreservedData struct {
	Name  string
	Bytes int
}

func (s *Service) PreservedData(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) ([]PreservedData, error) {
	originalFormat, err := s.originalFormatOf(ctx, ownerID, workID)
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
		return nil, fmt.Errorf("read preserved data: %w", err)
	}
	defer rows.Close()

	declaration, declared := s.reg.Declaration(originalFormat)
	found := make([]PreservedData, 0)
	for rows.Next() {
		var name, sample string
		var size int64
		if err := rows.Scan(&name, &size, &sample); err != nil {
			return nil, fmt.Errorf("read preserved data: %w", err)
		}
		if declared && declaration.RecordsNothing(name, []byte(sample)) {
			continue
		}
		found = append(found, PreservedData{Name: name, Bytes: int(size)})
	}
	return found, rows.Err()
}

func (s *Service) DeletePreservedData(
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

func (s *Service) originalFormatOf(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) (string, error) {
	var originalFormat *string
	err := s.pool.QueryRow(ctx, `
		select original_format
		  from works
		 where id = $1 and owner_id = $2 and deleted_at is null
	`, workID, ownerID).Scan(&originalFormat)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read the original format: %w", err)
	}
	if originalFormat == nil {
		return "", nil
	}
	return *originalFormat, nil
}
