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

func (s *Apps) Live(ctx context.Context, userID uuid.UUID) ([]ConnectedApp, error) {
	rows, err := db.New(s.pool).LiveConnectedApps(ctx, uuidValue(userID))
	if err != nil {
		return nil, fmt.Errorf("list live connected apps: %w", err)
	}
	apps := make([]ConnectedApp, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, ConnectedApp{
			ID: uuid.UUID(row.ID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
			AppName: row.AppName, Name: row.Name,
			Capabilities: capabilitiesFrom(
				row.AppVersion, row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedFormats,
			),
			Prefix: row.RefreshTokenPrefix, Permissions: permissionsFrom(row.Permissions),
			ConnectedAt: row.ConnectedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
		})
	}
	return apps, nil
}

func (s *Apps) LiveByID(
	ctx context.Context,
	userID uuid.UUID,
	appID uuid.UUID,
) (ConnectedApp, error) {
	row, err := db.New(s.pool).LiveConnectedApp(ctx, db.LiveConnectedAppParams{
		ConnectedAppID: uuidValue(appID), UserID: uuidValue(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ConnectedApp{}, ErrAppNotFound
	}
	if err != nil {
		return ConnectedApp{}, fmt.Errorf("read a live connected app: %w", err)
	}
	return ConnectedApp{
		ID: uuid.UUID(row.ID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
		AppName: row.AppName, Name: row.Name,
		Capabilities: capabilitiesFrom(
			row.AppVersion, row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedFormats,
		),
		Prefix: row.RefreshTokenPrefix, Permissions: permissionsFrom(row.Permissions),
		ConnectedAt: row.ConnectedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
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
