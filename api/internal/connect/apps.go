package connect

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	requestLifetime       = 10 * time.Minute
	authorizationLifetime = 5 * time.Minute
	accessTokenLifetime   = 15 * time.Minute
	refreshIdleLifetime   = 90 * 24 * time.Hour
	refreshReuseWindow    = 90 * 24 * time.Hour

	pollInterval = 5 * time.Second

	codeAttemptLimit = 5
	cleanupBatch     = 100
)

type PollDelayError struct {
	After time.Duration
}

func (e *PollDelayError) Error() string { return ErrPollTooSoon.Error() }
func (e *PollDelayError) Unwrap() error { return ErrPollTooSoon }

type RateLimitError struct {
	After time.Duration
}

func (e *RateLimitError) Error() string { return ErrTooManyRequests.Error() }
func (e *RateLimitError) Unwrap() error { return ErrTooManyRequests }

type Apps struct {
	pool          *pgxpool.Pool
	siteURL       string
	browserOrigin string
	hmacKey       []byte
}

func NewApps(pool *pgxpool.Pool, siteURL string, hmacKey []byte) *Apps {
	trimmed := strings.TrimRight(siteURL, "/")
	origin := trimmed
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		origin = parsed.Scheme + "://" + parsed.Host
	}
	return &Apps{
		pool: pool, siteURL: trimmed, browserOrigin: origin,
		hmacKey: append([]byte(nil), hmacKey...),
	}
}

func (s *Apps) BrowserOrigin() string { return s.browserOrigin }

func (s *Apps) Start(ctx context.Context, source string, input StartInput) (Request, error) {
	in, err := validateStart(input)
	if err != nil {
		return Request{}, err
	}
	if err := s.takeRate(ctx, "start", source, 30, time.Hour); err != nil {
		return Request{}, err
	}
	queries := db.New(s.pool)
	if _, err := queries.DeleteExpiredConnectionRequests(ctx, cleanupBatch); err != nil {
		return Request{}, fmt.Errorf("clear device requests: %w", err)
	}
	if err := deleteExpiredRates(ctx, queries); err != nil {
		return Request{}, err
	}

	expiresAt := time.Now().Add(requestLifetime)
	for attempt := 0; attempt < 8; attempt++ {
		deviceCode, deviceHash, err := newOpaqueCode()
		if err != nil {
			return Request{}, err
		}
		userCode, err := newCode(codeLength)
		if err != nil {
			return Request{}, err
		}
		err = queries.InsertConnectionRequest(ctx, db.InsertConnectionRequestParams{
			DeviceCodeHash:  deviceHash,
			UserCodeHash:    s.digest("user-code", userCode),
			AppName:         in.AppName,
			Name:            in.Name,
			AppVersion:      optionalText(in.AppVersion),
			ProtocolVersion: int32(in.ProtocolVersion),
			Capabilities:    in.Declared,
			AcceptedFormats: in.AcceptedFormats,
			Permissions:     permissionStrings(in.Permissions),
			ExpiresAt:       timestamptz(expiresAt),
		})
		if err == nil {
			return Request{
				DeviceCode: deviceCode, UserCode: FormatUserCode(userCode),
				VerifyURL: s.siteURL + "/connect", ExpiresAt: expiresAt,
				Interval: pollInterval,
			}, nil
		}
		if !isUniqueViolation(err) {
			return Request{}, fmt.Errorf("store device request: %w", err)
		}
	}
	return Request{}, errors.New("could not allocate a connection code")
}

func (s *Apps) StartAuthorization(
	ctx context.Context,
	source string,
	input AuthorizationInput,
) (Authorization, error) {
	in, err := validateAuthorization(input)
	if err != nil {
		return Authorization{}, err
	}
	if err := s.takeRate(ctx, "start", source, 30, time.Hour); err != nil {
		return Authorization{}, err
	}
	queries := db.New(s.pool)
	if _, err := queries.DeleteExpiredConnectionAuthorizations(ctx, cleanupBatch); err != nil {
		return Authorization{}, fmt.Errorf("clear browser authorizations: %w", err)
	}
	if err := deleteExpiredRates(ctx, queries); err != nil {
		return Authorization{}, err
	}
	expiresAt := time.Now().Add(authorizationLifetime)
	for attempt := 0; attempt < 8; attempt++ {
		requestCode, requestHash, err := newOpaqueCode()
		if err != nil {
			return Authorization{}, err
		}
		err = queries.InsertConnectionAuthorization(ctx, db.InsertConnectionAuthorizationParams{
			RequestHash: requestHash, RedirectUri: in.RedirectURI,
			State: in.State, CodeChallenge: in.CodeChallenge,
			AppName: in.AppName, Name: in.Name,
			AppVersion:      optionalText(in.AppVersion),
			ProtocolVersion: int32(in.ProtocolVersion),
			Capabilities:    in.Declared, AcceptedFormats: in.AcceptedFormats,
			Permissions: permissionStrings(in.Permissions), ExpiresAt: timestamptz(expiresAt),
		})
		if err == nil {
			return Authorization{
				URL:       s.siteURL + "/connect?request=" + url.QueryEscape(requestCode),
				ExpiresAt: expiresAt,
			}, nil
		}
		if !isUniqueViolation(err) {
			return Authorization{}, fmt.Errorf("store browser authorization: %w", err)
		}
	}
	return Authorization{}, errors.New("could not allocate an authorization request")
}

func (s *Apps) Pending(ctx context.Context, userID uuid.UUID, rawCode string) (Pending, error) {
	if err := s.takeRate(ctx, "user-code", userID.String(), codeAttemptLimit, time.Hour); err != nil {
		return Pending{}, ErrTooManyCodes
	}
	code, ok := normalizeUserCode(rawCode)
	if !ok {
		return Pending{}, ErrRequestNotFound
	}
	row, err := db.New(s.pool).ReviewConnectionRequest(ctx, db.ReviewConnectionRequestParams{
		ReviewedBy:   uuidValue(userID),
		UserCodeHash: s.digest("user-code", code),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Pending{}, ErrRequestNotFound
	}
	if err != nil {
		return Pending{}, fmt.Errorf("review device request: %w", err)
	}
	return pendingFromDeviceReview(row, s.deviceApprovalProof(userID, code)), nil
}

func (s *Apps) Approve(
	ctx context.Context,
	userID uuid.UUID,
	rawCode string,
	approvalToken string,
) (Pending, error) {
	code, ok := normalizeUserCode(rawCode)
	tokenHash, tokenOK := s.deviceApprovalProofHash(userID, code, approvalToken)
	if !ok || !tokenOK {
		return Pending{}, ErrRequestNotFound
	}
	row, err := db.New(s.pool).ApproveConnectionRequest(ctx, db.ApproveConnectionRequestParams{
		ReviewedBy: uuidValue(userID), ReviewTokenHash: tokenHash,
		UserCodeHash: s.digest("user-code", code),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Pending{}, ErrRequestNotFound
	}
	if err != nil {
		return Pending{}, fmt.Errorf("approve device request: %w", err)
	}
	return pendingFromDeviceApproval(row), nil
}

func (s *Apps) Deny(
	ctx context.Context,
	userID uuid.UUID,
	rawCode string,
	approvalToken string,
) error {
	code, ok := normalizeUserCode(rawCode)
	tokenHash, tokenOK := s.deviceApprovalProofHash(userID, code, approvalToken)
	if !ok || !tokenOK {
		return ErrRequestNotFound
	}
	denied, err := db.New(s.pool).DenyConnectionRequest(ctx, db.DenyConnectionRequestParams{
		ReviewedBy: uuidValue(userID), ReviewTokenHash: tokenHash,
		UserCodeHash: s.digest("user-code", code),
	})
	if err != nil {
		return fmt.Errorf("deny device request: %w", err)
	}
	if denied == 0 {
		return ErrRequestNotFound
	}
	return nil
}

func (s *Apps) PendingAuthorization(
	ctx context.Context,
	userID uuid.UUID,
	requestCode string,
) (Pending, error) {
	hash, ok := opaqueCodeHash(requestCode)
	if !ok {
		return Pending{}, ErrRequestNotFound
	}
	row, err := db.New(s.pool).ReviewConnectionAuthorization(ctx, db.ReviewConnectionAuthorizationParams{
		ReviewedBy: uuidValue(userID), RequestHash: hash,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Pending{}, ErrRequestNotFound
	}
	if err != nil {
		return Pending{}, fmt.Errorf("review browser authorization: %w", err)
	}
	return Pending{
		StartInput: startFrom(
			row.AppName, row.Name, row.AppVersion, row.ProtocolVersion,
			row.Capabilities, row.AcceptedFormats, row.Permissions,
		),
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (s *Apps) ApproveAuthorization(
	ctx context.Context,
	userID uuid.UUID,
	requestCode string,
) (Redirect, error) {
	requestHash, ok := opaqueCodeHash(requestCode)
	if !ok {
		return Redirect{}, ErrRequestNotFound
	}
	code, codeHash, err := newOpaqueCode()
	if err != nil {
		return Redirect{}, err
	}
	row, err := db.New(s.pool).ApproveConnectionAuthorization(ctx, db.ApproveConnectionAuthorizationParams{
		AuthorizationCodeHash: codeHash, ReviewedBy: uuidValue(userID),
		RequestHash: requestHash,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Redirect{}, ErrRequestNotFound
	}
	if err != nil {
		return Redirect{}, fmt.Errorf("approve browser authorization: %w", err)
	}
	destination, err := redirectWith(row.RedirectUri, "code", code, row.State)
	if err != nil {
		return Redirect{}, err
	}
	return Redirect{URL: destination}, nil
}

func (s *Apps) DenyAuthorization(
	ctx context.Context,
	userID uuid.UUID,
	requestCode string,
) (Redirect, error) {
	hash, ok := opaqueCodeHash(requestCode)
	if !ok {
		return Redirect{}, ErrRequestNotFound
	}
	row, err := db.New(s.pool).DenyConnectionAuthorization(ctx, db.DenyConnectionAuthorizationParams{
		ReviewedBy: uuidValue(userID), RequestHash: hash,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Redirect{}, ErrRequestNotFound
	}
	if err != nil {
		return Redirect{}, fmt.Errorf("deny browser authorization: %w", err)
	}
	destination, err := redirectWith(row.RedirectUri, "error", "access_denied", row.State)
	if err != nil {
		return Redirect{}, err
	}
	return Redirect{URL: destination}, nil
}

func (s *Apps) Poll(
	ctx context.Context,
	source string,
	deviceCode string,
) (Credentials, bool, error) {
	if err := s.takeRate(ctx, "poll", source, 600, time.Minute); err != nil {
		return Credentials{}, false, err
	}
	hash, ok := opaqueCodeHash(deviceCode)
	if !ok {
		return Credentials{}, false, ErrRequestNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Credentials{}, false, fmt.Errorf("begin device poll: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	request, err := queries.LockConnectionRequest(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, false, ErrRequestNotFound
	}
	if err != nil {
		return Credentials{}, false, fmt.Errorf("read device request: %w", err)
	}
	if request.RedeemedAt.Valid {
		return Credentials{}, false, ErrRequestNotFound
	}
	if time.Now().After(request.ExpiresAt.Time) {
		return Credentials{}, false, ErrRequestExpired
	}
	if request.DeniedAt.Valid {
		return Credentials{}, false, ErrAccessDenied
	}
	currentInterval := time.Duration(request.PollIntervalSeconds) * time.Second
	tooSoon := request.LastPolledAt.Valid &&
		time.Since(request.LastPolledAt.Time) < currentInterval
	nextInterval, err := queries.RecordConnectionPoll(ctx, db.RecordConnectionPollParams{
		SlowDown: tooSoon, DeviceCodeHash: hash,
	})
	if err != nil {
		return Credentials{}, false, fmt.Errorf("record device poll: %w", err)
	}
	if tooSoon {
		if err := tx.Commit(ctx); err != nil {
			return Credentials{}, false, fmt.Errorf("commit slow down: %w", err)
		}
		return Credentials{}, false, &PollDelayError{After: time.Duration(nextInterval) * time.Second}
	}
	if !request.ApprovedBy.Valid {
		if err := tx.Commit(ctx); err != nil {
			return Credentials{}, false, fmt.Errorf("commit pending poll: %w", err)
		}
		return Credentials{}, false, nil
	}
	issued, err := issueCredentials(ctx, queries, request.ApprovedBy, startFrom(
		request.AppName, request.Name, request.AppVersion, request.ProtocolVersion,
		request.Capabilities, request.AcceptedFormats, request.Permissions,
	))
	if err != nil {
		return Credentials{}, false, err
	}
	redeemed, err := queries.RedeemConnectionRequest(ctx, hash)
	if err != nil || redeemed != 1 {
		if err == nil {
			err = ErrRequestNotFound
		}
		return Credentials{}, false, fmt.Errorf("redeem device request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Credentials{}, false, fmt.Errorf("commit device grant: %w", err)
	}
	return issued, true, nil
}

func (s *Apps) Exchange(
	ctx context.Context,
	source string,
	authorizationCode string,
	verifier string,
	redirectURI string,
) (Credentials, error) {
	if err := s.takeRate(ctx, "exchange", source, 60, time.Hour); err != nil {
		return Credentials{}, err
	}
	codeHash, ok := opaqueCodeHash(authorizationCode)
	if !ok || !validLoopbackRedirect(redirectURI) {
		return Credentials{}, ErrInvalidPKCE
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Credentials{}, fmt.Errorf("begin code exchange: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	request, err := queries.LockConnectionAuthorization(ctx, codeHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrInvalidPKCE
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("read authorization code: %w", err)
	}
	if request.RedeemedAt.Valid || request.DeniedAt.Valid ||
		!request.ApprovedBy.Valid || time.Now().After(request.ExpiresAt.Time) ||
		request.RedirectUri != redirectURI ||
		!challengeMatches(verifier, request.CodeChallenge) {
		return Credentials{}, ErrInvalidPKCE
	}
	issued, err := issueCredentials(ctx, queries, request.ApprovedBy, startFrom(
		request.AppName, request.Name, request.AppVersion, request.ProtocolVersion,
		request.Capabilities, request.AcceptedFormats, request.Permissions,
	))
	if err != nil {
		return Credentials{}, err
	}
	redeemed, err := queries.RedeemConnectionAuthorization(ctx, codeHash)
	if err != nil || redeemed != 1 {
		if err == nil {
			err = ErrInvalidPKCE
		}
		return Credentials{}, fmt.Errorf("redeem authorization code: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Credentials{}, fmt.Errorf("commit code exchange: %w", err)
	}
	return issued, nil
}

func (s *Apps) List(ctx context.Context, userID uuid.UUID) ([]ConnectedApp, error) {
	rows, err := db.New(s.pool).ListConnectedApps(ctx, uuidValue(userID))
	if err != nil {
		return nil, fmt.Errorf("list connected apps: %w", err)
	}
	apps := make([]ConnectedApp, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, ConnectedApp{
			ID: uuid.UUID(row.ID.Bytes), UserID: userID,
			AppName: row.AppName, Name: row.Name,
			Capabilities: capabilitiesFrom(
				row.AppVersion, row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedFormats,
			),
			Prefix: row.RefreshTokenPrefix, Permissions: permissionsFrom(row.Permissions),
			ConnectedAt: row.ConnectedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
			RevokedAt: optionalTime(row.RevokedAt),
		})
	}
	return apps, nil
}

func (s *Apps) Revoke(ctx context.Context, userID, appID uuid.UUID) error {
	revoked, err := db.New(s.pool).RevokeConnectedApp(ctx, db.RevokeConnectedAppParams{
		ConnectedAppID: uuidValue(appID), UserID: uuidValue(userID),
	})
	if err != nil {
		return fmt.Errorf("revoke a connected app: %w", err)
	}
	if !revoked {
		return ErrAppNotFound
	}
	return nil
}

func (s *Apps) UpdateCapabilities(
	ctx context.Context,
	app ConnectedApp,
	capabilities Capabilities,
) (ConnectedApp, error) {
	validated, err := validateCapabilities(capabilities)
	if err != nil {
		return ConnectedApp{}, err
	}
	row, err := db.New(s.pool).UpdateConnectedAppCapabilities(
		ctx,
		db.UpdateConnectedAppCapabilitiesParams{
			AppVersion:      optionalText(validated.AppVersion),
			ProtocolVersion: pgtype.Int4{Int32: int32(validated.ProtocolVersion), Valid: true},
			Capabilities:    validated.Declared, AcceptedFormats: validated.AcceptedFormats,
			ConnectedAppID: uuidValue(app.ID),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConnectedApp{}, ErrAppNotFound
	}
	if err != nil {
		return ConnectedApp{}, fmt.Errorf("update the capabilities of a connected app: %w", err)
	}
	return ConnectedApp{
		ID: uuid.UUID(row.ID.Bytes), UserID: app.UserID,
		AppName: row.AppName, Name: row.Name,
		Capabilities: capabilitiesFrom(
			row.AppVersion, row.ProtocolVersion.Int32, row.Capabilities, row.AcceptedFormats,
		),
		Prefix: row.RefreshTokenPrefix, Permissions: permissionsFrom(row.Permissions),
		ConnectedAt: row.ConnectedAt.Time, LastSeenAt: optionalTime(row.LastSeenAt),
	}, nil
}

func pendingFromDeviceReview(row db.ReviewConnectionRequestRow, token string) Pending {
	return Pending{
		StartInput: startFrom(
			row.AppName, row.Name, row.AppVersion, row.ProtocolVersion,
			row.Capabilities, row.AcceptedFormats, row.Permissions,
		),
		ExpiresAt: row.ExpiresAt.Time, ApprovalToken: token,
	}
}

func pendingFromDeviceApproval(row db.ApproveConnectionRequestRow) Pending {
	return Pending{
		StartInput: startFrom(
			row.AppName, row.Name, row.AppVersion, row.ProtocolVersion,
			row.Capabilities, row.AcceptedFormats, row.Permissions,
		),
		ExpiresAt: row.ExpiresAt.Time,
	}
}

func startFrom(
	appName string,
	name string,
	appVersion pgtype.Text,
	protocol int32,
	declared []string,
	formats []string,
	permissions []string,
) StartInput {
	return StartInput{
		AppName: appName, Name: name,
		Capabilities: capabilitiesFrom(appVersion, protocol, declared, formats),
		Permissions:  permissionsFrom(permissions),
	}
}

func capabilitiesFrom(
	appVersion pgtype.Text,
	protocol int32,
	declared []string,
	formats []string,
) Capabilities {
	return Capabilities{
		AppVersion: textFrom(appVersion), ProtocolVersion: int(protocol),
		Declared: declared, AcceptedFormats: formats,
	}
}

func (s *Apps) takeRate(
	ctx context.Context,
	action string,
	source string,
	limit int32,
	window time.Duration,
) error {
	if source == "" {
		source = "unknown"
	}
	row, err := db.New(s.pool).TakeConnectionRateLimit(ctx, db.TakeConnectionRateLimitParams{
		KeyHash: s.digest("rate:"+action, source), Action: action,
		WindowCutoff: timestamptz(time.Now().Add(-window)),
	})
	if err != nil {
		return fmt.Errorf("rate a connection request: %w", err)
	}
	if row.Attempts > limit {
		after := time.Until(row.WindowStart.Time.Add(window))
		if after < time.Second {
			after = time.Second
		}
		return &RateLimitError{After: after}
	}
	return nil
}

func deleteExpiredRates(ctx context.Context, queries *db.Queries) error {
	_, err := queries.DeleteExpiredConnectionRateLimits(ctx, db.DeleteExpiredConnectionRateLimitsParams{
		WindowCutoff: timestamptz(time.Now().Add(-24 * time.Hour)),
		BatchSize:    cleanupBatch,
	})
	if err != nil {
		return fmt.Errorf("clear connection rate limits: %w", err)
	}
	return nil
}

func (s *Apps) digest(purpose, value string) []byte {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(purpose))
	mac.Write([]byte{0})
	mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (s *Apps) deviceApprovalProof(userID uuid.UUID, code string) string {
	digest := s.digest("device-approval", userID.String()+"\x00"+code)
	return base64.RawURLEncoding.EncodeToString(digest)
}

func (s *Apps) deviceApprovalProofHash(
	userID uuid.UUID,
	code string,
	proof string,
) ([]byte, bool) {
	hash, ok := opaqueCodeHash(proof)
	if !ok || !hmac.Equal([]byte(proof), []byte(s.deviceApprovalProof(userID, code))) {
		return nil, false
	}
	return hash, true
}

func PollInterval() time.Duration { return pollInterval }

func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func optionalTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	moment := value.Time
	return &moment
}

func textFrom(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func optionalText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func uuidValue(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
