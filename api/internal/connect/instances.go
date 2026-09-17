package connect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Apps) Live(ctx context.Context, userID uuid.UUID) ([]Instance, error) {
	rows, err := db.New(s.pool).LiveLinkedInstances(ctx, uuidValue(userID))
	if err != nil {
		return nil, fmt.Errorf("list live instances: %w", err)
	}
	instances := make([]Instance, 0, len(rows))
	for _, row := range rows {
		instances = append(instances, Instance{
			ID: uuid.UUID(row.ID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
			Declaration: declarationFrom(
				row.ApplicationName, row.InstanceName, row.ApplicationVersion,
				row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedTargets,
			),
			Prefix: row.RefreshTokenPrefix, Scopes: scopesFrom(row.Scopes),
			LinkedAt: row.LinkedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
		})
	}
	return instances, nil
}

func (s *Apps) LiveByID(
	ctx context.Context,
	userID uuid.UUID,
	instanceID uuid.UUID,
) (Instance, error) {
	row, err := db.New(s.pool).LiveLinkedInstance(ctx, db.LiveLinkedInstanceParams{
		InstanceID: uuidValue(instanceID), UserID: uuidValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrInstanceNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("read live instance: %w", err)
	}
	return Instance{
		ID: uuid.UUID(row.ID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
		Declaration: declarationFrom(
			row.ApplicationName, row.InstanceName, row.ApplicationVersion,
			row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedTargets,
		),
		Prefix: row.RefreshTokenPrefix, Scopes: scopesFrom(row.Scopes),
		LinkedAt: row.LinkedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
	}, nil
}

func (s *Apps) Throttle(
	ctx context.Context,
	action string,
	source string,
	limit int32,
	window time.Duration,
) error {
	return s.takeRate(ctx, action, source, limit, window)
}
