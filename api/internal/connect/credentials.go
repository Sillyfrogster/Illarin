package connect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Apps) Refresh(ctx context.Context, source, refreshToken string) (Credentials, error) {
	if err := s.takeRate(ctx, "refresh", source, 600, time.Hour); err != nil {
		return Credentials{}, err
	}
	oldHash, ok := credentialHash(refreshToken, refreshTokenType)
	if !ok {
		return Credentials{}, ErrNotLive
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Credentials{}, fmt.Errorf("begin token refresh: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	app, err := queries.LockConnectedAppByRefreshToken(ctx, oldHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, handleRefreshReuse(ctx, tx, queries, oldHash)
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("read refresh token: %w", err)
	}
	if refreshExpired(app) {
		return Credentials{}, revokeInactiveRefresh(ctx, tx, queries, app.ID)
	}
	rotated, err := rotateCredentials(ctx, queries, app, oldHash)
	if err != nil {
		return Credentials{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Credentials{}, fmt.Errorf("commit token refresh: %w", err)
	}
	return rotated, nil
}

func handleRefreshReuse(
	ctx context.Context,
	tx pgx.Tx,
	queries *db.Queries,
	oldHash []byte,
) error {
	used, err := queries.ConnectedAppForUsedRefreshToken(ctx, oldHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotLive
	}
	if err != nil {
		return fmt.Errorf("check refresh reuse: %w", err)
	}
	if _, err := queries.RevokeConnectedAppByID(ctx, used.ConnectedAppID); err != nil {
		return fmt.Errorf("revoke reused refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh reuse revocation: %w", err)
	}
	return ErrRefreshReuse
}

func refreshExpired(app db.LockConnectedAppByRefreshTokenRow) bool {
	lastActive := app.ConnectedAt.Time
	if app.LastSeenAt.Valid && app.LastSeenAt.Time.After(lastActive) {
		lastActive = app.LastSeenAt.Time
	}
	return time.Now().After(lastActive.Add(refreshIdleLifetime))
}

func revokeInactiveRefresh(
	ctx context.Context,
	tx pgx.Tx,
	queries *db.Queries,
	appID pgtype.UUID,
) error {
	if _, err := queries.RevokeConnectedAppByID(ctx, appID); err != nil {
		return fmt.Errorf("revoke inactive refresh token: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit inactive refresh revocation: %w", err)
	}
	return ErrNotLive
}

func rotateCredentials(
	ctx context.Context,
	queries *db.Queries,
	app db.LockConnectedAppByRefreshTokenRow,
	oldHash []byte,
) (Credentials, error) {
	accessToken, _, accessHash, err := newCredential(accessTokenType)
	if err != nil {
		return Credentials{}, err
	}
	refreshToken, refreshPrefix, refreshHash, err := newCredential(refreshTokenType)
	if err != nil {
		return Credentials{}, err
	}
	expiresAt := time.Now().Add(accessTokenLifetime)
	_, err = queries.RotateAppRefreshToken(ctx, db.RotateAppRefreshTokenParams{
		OldRefreshTokenHash:   oldHash,
		DetectableUntil:       timestamptz(time.Now().Add(refreshReuseWindow)),
		NewRefreshTokenHash:   refreshHash,
		NewRefreshTokenPrefix: refreshPrefix,
		ConnectedAppID:        app.ID,
	})
	if err != nil {
		return Credentials{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	if _, err := queries.InsertAppAccessToken(ctx, db.InsertAppAccessTokenParams{
		TokenHash: accessHash, ConnectedAppID: app.ID, ExpiresAt: timestamptz(expiresAt),
	}); err != nil {
		return Credentials{}, fmt.Errorf("store access token: %w", err)
	}
	_, _ = queries.DeleteExpiredAppAccessTokens(ctx, cleanupBatch)
	_, _ = queries.DeleteExpiredAppRefreshHistory(ctx, cleanupBatch)
	return Credentials{
		ConnectedApp: ConnectedApp{
			ID: uuid.UUID(app.ID.Bytes), UserID: uuid.UUID(app.UserID.Bytes),
			AppName: app.AppName, Name: app.Name,
			Capabilities: capabilitiesFrom(
				app.AppVersion, app.ProtocolVersion.Int32, app.Capabilities, app.AcceptedFormats,
			),
			Prefix: refreshPrefix, Permissions: permissionsFrom(app.Permissions),
			ConnectedAt: app.ConnectedAt.Time, LastSeenAt: optionalTime(app.LastSeenAt),
		},
		AccessToken: accessToken, AccessTokenExpiresAt: expiresAt,
		RefreshToken: refreshToken,
	}, nil
}

func (s *Apps) Authenticate(ctx context.Context, token string, needs Permission) (ConnectedApp, error) {
	hash, ok := credentialHash(token, accessTokenType)
	if !ok {
		return ConnectedApp{}, ErrNotLive
	}
	row, err := db.New(s.pool).TouchConnectedAppByAccessToken(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConnectedApp{}, ErrNotLive
	}
	if err != nil {
		return ConnectedApp{}, fmt.Errorf("read access token: %w", err)
	}
	app := ConnectedApp{
		ID: uuid.UUID(row.ID.Bytes), UserID: uuid.UUID(row.UserID.Bytes),
		AppName: row.AppName, Name: row.Name,
		Capabilities: capabilitiesFrom(
			row.AppVersion, row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedFormats,
		),
		Prefix: row.RefreshTokenPrefix, Permissions: permissionsFrom(row.Permissions),
		ConnectedAt: row.ConnectedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
	}
	if needs != "" && !app.Grants(needs) {
		return ConnectedApp{}, ErrMissingPermission
	}
	return app, nil
}

func issueCredentials(
	ctx context.Context,
	queries *db.Queries,
	userID pgtype.UUID,
	start StartInput,
) (Credentials, error) {
	accessToken, _, accessHash, err := newCredential(accessTokenType)
	if err != nil {
		return Credentials{}, err
	}
	refreshToken, refreshPrefix, refreshHash, err := newCredential(refreshTokenType)
	if err != nil {
		return Credentials{}, err
	}
	appID := uuid.New()
	row, err := queries.InsertConnectedApp(ctx, db.InsertConnectedAppParams{
		ID: uuidValue(appID), UserID: userID,
		AppName:          start.AppName,
		Name:             start.Name,
		AppVersion:       optionalText(start.AppVersion),
		ProtocolVersion:  pgtype.Int4{Int32: int32(start.ProtocolVersion), Valid: true},
		Capabilities:     start.Declared,
		AcceptedFormats:  start.AcceptedFormats,
		RefreshTokenHash: refreshHash, RefreshTokenPrefix: refreshPrefix,
		Permissions: permissionStrings(start.Permissions),
	})
	if err != nil {
		return Credentials{}, fmt.Errorf("create a connected app: %w", err)
	}
	expiresAt := time.Now().Add(accessTokenLifetime)
	if _, err := queries.InsertAppAccessToken(ctx, db.InsertAppAccessTokenParams{
		TokenHash: accessHash, ConnectedAppID: uuidValue(appID),
		ExpiresAt: timestamptz(expiresAt),
	}); err != nil {
		return Credentials{}, fmt.Errorf("store access token: %w", err)
	}
	_, _ = queries.DeleteExpiredAppAccessTokens(ctx, cleanupBatch)
	return Credentials{
		ConnectedApp: ConnectedApp{
			ID: appID, UserID: uuid.UUID(userID.Bytes),
			AppName: start.AppName, Name: start.Name,
			Capabilities: start.Capabilities, Prefix: row.RefreshTokenPrefix,
			Permissions: permissionsFrom(row.Permissions), ConnectedAt: row.ConnectedAt.Time,
		},
		AccessToken: accessToken, AccessTokenExpiresAt: expiresAt,
		RefreshToken: refreshToken,
	}, nil
}
