package connect

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrArtifactNotFound = errors.New("no such delivery artifact")

func (s *Sends) Artifact(
	ctx context.Context,
	deliveryID uuid.UUID,
	expires string,
	signature string,
) (uuid.UUID, string, error) {
	path := deliveryPathStart + deliveryID.String() + "/export"
	if !s.works.ValidSignature(path, expires, signature) {
		return uuid.Nil, "", ErrArtifactNotFound
	}
	row, err := db.New(s.pool).DeliveryForArtifact(ctx, uuidValue(deliveryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrArtifactNotFound
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("read a delivery artifact: %w", err)
	}
	return uuid.UUID(row.WorkID.Bytes), row.ChosenTarget.String, nil
}
