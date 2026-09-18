package page

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const RecoveryWindow = 30 * 24 * time.Hour

func (s *Service) Delete(ctx context.Context, ownerID, id uuid.UUID) error {
	queries := db.New(s.pool)
	state, err := queries.WorkDeletionState(ctx, db.WorkDeletionStateParams{
		ID: uuidToPgtype(id), OwnerID: uuidToPgtype(ownerID),
	})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && state.DeletedAt.Valid) {
		return work.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read work deletion state: %w", err)
	}
	if state.WithheldAt.Valid {
		return work.ErrWorkFrozen
	}
	now := s.works.Now()
	changed, err := queries.SoftDeleteWork(ctx, db.SoftDeleteWorkParams{
		ID: uuidToPgtype(id), OwnerID: uuidToPgtype(ownerID),
		DeletedAt: timeToNullable(&now), RecoverableUntil: timeToNullable(timePointer(now.Add(RecoveryWindow))),
	})
	if err != nil {
		return fmt.Errorf("delete work: %w", err)
	}
	if changed == 0 {
		return work.ErrNotFound
	}
	return nil
}

func (s *Service) Restore(ctx context.Context, ownerID, id uuid.UUID) error {
	now := s.works.Now()
	changed, err := db.New(s.pool).RestoreWork(ctx, db.RestoreWorkParams{
		ID: uuidToPgtype(id), OwnerID: uuidToPgtype(ownerID), UpdatedAt: timeToNullable(&now),
	})
	if err != nil {
		return fmt.Errorf("restore work: %w", err)
	}
	if changed == 0 {
		return work.ErrNotFound
	}
	return nil
}

func (s *Service) Deleted(ctx context.Context, ownerID uuid.UUID, handle string) ([]DeletedWork, error) {
	rows, err := db.New(s.pool).ListDeletedWorks(ctx, db.ListDeletedWorksParams{
		OwnerID: uuidToPgtype(ownerID), Username: handle,
		RecoverableUntil: timeToNullable(timePointer(s.works.Now())),
	})
	if err != nil {
		return nil, fmt.Errorf("list deleted works: %w", err)
	}
	items := make([]DeletedWork, len(rows))
	for i, row := range rows {
		items[i] = DeletedWork{
			Id: uuidFromPgtype(row.ID), Name: row.Name, Type: DeletedWorkType(row.Type),
			DeletedAt: timeFromPgtype(row.DeletedAt), RecoverableUntil: timeFromPgtype(row.RecoverableUntil),
		}
	}
	return items, nil
}

func timePointer(value time.Time) *time.Time { return &value }
