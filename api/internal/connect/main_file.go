package connect

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrMainFileNotFound = errors.New("no such main file")

func (s *Sends) MainFile(
	ctx context.Context,
	deliveryID uuid.UUID,
	expires string,
	signature string,
) (uuid.UUID, string, error) {
	path := deliveryPathStart + deliveryID.String() + "/export"
	if !s.works.ValidSignature(path, expires, signature) {
		return uuid.Nil, "", ErrMainFileNotFound
	}
	row, err := db.New(s.pool).DeliveryForMainFile(ctx, uuidValue(deliveryID))
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrMainFileNotFound
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("read the main file of a send: %w", err)
	}
	return uuid.UUID(row.WorkID.Bytes), row.ChosenTarget.String, nil
}
