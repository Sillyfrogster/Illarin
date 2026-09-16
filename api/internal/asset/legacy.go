package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type LegacyAsset struct {
	ID   uuid.UUID
	Name string
}

func (s *Service) ResolveLegacyAddress(ctx context.Context, address string) (LegacyAsset, error) {
	tx, err := s.beginReadSnapshot(ctx)
	if err != nil {
		return LegacyAsset{}, err
	}
	defer tx.Rollback(ctx)
	row, err := db.New(tx).LegacyPathTarget(ctx, address)
	if errors.Is(err, pgx.ErrNoRows) {
		return LegacyAsset{}, ErrNotFound
	}
	if err != nil {
		return LegacyAsset{}, fmt.Errorf("resolve the v1 address: %w", err)
	}
	return LegacyAsset{ID: uuid.UUID(row.ID.Bytes), Name: row.Name}, nil
}
