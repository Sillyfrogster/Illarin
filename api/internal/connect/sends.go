package connect

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Works interface {
	SendableWork(ctx context.Context, q db.DBTX, workID uuid.UUID) (Sendable, error)
	SignedURL(path string) string
	ValidSignature(path, expires, signature string) bool
}

type Settings struct {
	HoldFloor         time.Duration
	HoldCeiling       time.Duration
	Recheck           time.Duration
	Lease             time.Duration
	Retention         time.Duration
	CleanupInterval   time.Duration
	Batch             int
	MaxAttempts       int
	PendingPerApp     int
	ConcurrentHolds   int
	MaxAcknowledged   int
	MaxLibraryEntries int
}

func DefaultSettings() Settings {
	return Settings{
		HoldFloor:         25 * time.Second,
		HoldCeiling:       30 * time.Second,
		Recheck:           5 * time.Second,
		Lease:             dispatch.Life,
		Retention:         7 * 24 * time.Hour,
		CleanupInterval:   5 * time.Minute,
		Batch:             10,
		MaxAttempts:       5,
		PendingPerApp:     100,
		ConcurrentHolds:   512,
		MaxAcknowledged:   32,
		MaxLibraryEntries: 2000,
	}
}

const (
	actionQueue     = "send-queue"
	actionCollect   = "send-collect"
	actionSyncPart  = "library-sync"
	actionSyncWhole = "library-snapshot"
	queueLimit      = 120
	collectLimit    = 400
	syncPartLimit   = 120
	syncWholeLimit  = 24
	sendPathStart   = "/send/"
)

type Sends struct {
	pool     *pgxpool.Pool
	works    Works
	apps     *Apps
	settings Settings
	waiting  *hub
	now      func() time.Time
}

func NewSends(
	pool *pgxpool.Pool,
	works Works,
	apps *Apps,
	settings Settings,
) *Sends {
	return &Sends{
		pool: pool, works: works, apps: apps, settings: settings,
		waiting: newHub(settings.ConcurrentHolds), now: time.Now,
	}
}

func (s *Sends) Queue(
	ctx context.Context,
	userID uuid.UUID,
	appID uuid.UUID,
	workID uuid.UUID,
) (Send, error) {
	if err := s.apps.Throttle(
		ctx, actionQueue, userID.String(), queueLimit, time.Hour,
	); err != nil {
		return Send{}, err
	}
	app, err := s.apps.LiveByID(ctx, userID, appID)
	if errors.Is(err, ErrAppNotFound) {
		return Send{}, ErrNoAppOfYours
	}
	if err != nil {
		return Send{}, err
	}
	if !app.Grants(PermissionReceiveWorks) {
		return Send{}, ErrMissingPermission
	}
	sendable, err := s.works.SendableWork(ctx, s.pool, workID)
	if errors.Is(err, ErrNotSendable) {
		return Send{}, ErrWorkNotSendable
	}
	if err != nil {
		return Send{}, err
	}
	if !installs(app.Declared, sendable) {
		return Send{}, ErrCannotInstall
	}
	if _, _, chosen := chooseFormat(
		app.AcceptedFormats, sendable.Formats, sendable.HasOriginal,
	); !chosen {
		return Send{}, ErrNoFormat
	}

	queries := db.New(s.pool)
	waiting, err := queries.CountLiveSends(ctx, uuidValue(appID))
	if err != nil {
		return Send{}, fmt.Errorf("count waiting sends: %w", err)
	}
	if waiting >= int64(s.settings.PendingPerApp) {
		return Send{}, ErrQueueFull
	}
	row, err := queries.QueueSend(ctx, db.QueueSendParams{
		ID: uuidValue(uuid.New()), ConnectedAppID: uuidValue(appID),
		WorkID:    uuidValue(workID),
		ExpiresAt: timestamptz(s.now().Add(s.settings.Retention)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return s.liveSend(ctx, appID, workID)
	}
	if err != nil {
		return Send{}, fmt.Errorf("queue a send: %w", err)
	}
	s.waiting.signal(appID)
	return sendFrom(
		row.ID, row.ConnectedAppID, row.WorkID, row.State, row.SettledReason,
		row.QueuedAt, row.SettledAt, row.ExpiresAt, row.UpdatesInstall,
	), nil
}

func (s *Sends) liveSend(
	ctx context.Context,
	appID uuid.UUID,
	workID uuid.UUID,
) (Send, error) {
	row, err := db.New(s.pool).LiveSendForWork(ctx, db.LiveSendForWorkParams{
		ConnectedAppID: uuidValue(appID), WorkID: uuidValue(workID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Send{}, ErrSendNotFound
	}
	if err != nil {
		return Send{}, fmt.Errorf("read the waiting send: %w", err)
	}
	return sendFrom(
		row.ID, row.ConnectedAppID, row.WorkID, row.State, row.SettledReason,
		row.QueuedAt, row.SettledAt, row.ExpiresAt, row.UpdatesInstall,
	), nil
}

func (s *Sends) Discard(ctx context.Context, userID, sendID uuid.UUID) error {
	discarded, err := db.New(s.pool).DiscardSend(ctx, db.DiscardSendParams{
		SendID: uuidValue(sendID), UserID: uuidValue(userID),
	})
	if err != nil {
		return fmt.Errorf("discard a send: %w", err)
	}
	if discarded == 0 {
		return ErrSendNotFound
	}
	return nil
}

func (s *Sends) WorkApps(
	ctx context.Context,
	userID uuid.UUID,
	workID uuid.UUID,
) (WorkApps, error) {
	queries := db.New(s.pool)
	number, err := queries.SendableWorkVersion(ctx, uuidValue(workID))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkApps{}, ErrWorkNotFound
	}
	if err != nil {
		return WorkApps{}, fmt.Errorf("read the work to send: %w", err)
	}
	sendable, err := s.works.SendableWork(ctx, s.pool, workID)
	if errors.Is(err, ErrNotSendable) {
		return WorkApps{}, ErrWorkNotFound
	}
	if err != nil {
		return WorkApps{}, fmt.Errorf("read the formats a send can use: %w", err)
	}
	rows, err := queries.WorkConnectedAppStates(ctx, db.WorkConnectedAppStatesParams{
		WorkID: uuidValue(workID), UserID: uuidValue(userID),
	})
	if err != nil {
		return WorkApps{}, fmt.Errorf("read connected app state for a work: %w", err)
	}
	found := WorkApps{VersionNumber: int(number), Items: []AppState{}}
	for _, row := range rows {
		_, _, canReceive := chooseFormat(
			row.AcceptedFormats, sendable.Formats, sendable.HasOriginal,
		)
		state := AppState{
			ConnectedAppID: uuid.UUID(row.ID.Bytes),
			AppName:        row.AppName,
			Name:           row.Name,
			LastSeenAt:     optionalTime(row.LastSeenAt),
			CanReceive: holdsPermission(row.Permissions, PermissionReceiveWorks) &&
				installs(row.Capabilities, sendable) && canReceive,
			ReportsLibrary: holdsPermission(row.Permissions, PermissionSyncLibrary),
		}
		if row.SendID.Valid {
			waiting := sendFrom(
				row.SendID, row.ID, uuidValue(workID), row.SendState,
				row.SettledReason, row.QueuedAt, row.SettledAt, row.ExpiresAt,
				row.UpdatesInstall,
			)
			state.Send = &waiting
		}
		if row.InstalledVersion.Valid {
			installed := int(row.InstalledVersion.Int32)
			state.InstalledVersion = &installed
			state.UpdateAvailable = installed < found.VersionNumber
		}
		found.Items = append(found.Items, state)
	}
	return found, nil
}

// UpdatableApps names the account's connected apps that hold an older copy of a work and can receive it.
func (s *Sends) UpdatableApps(
	ctx context.Context,
	userID uuid.UUID,
	workIDs []uuid.UUID,
) (map[uuid.UUID][]AppState, error) {
	behind, err := s.worksInstalledBehind(ctx, userID, workIDs)
	if err != nil {
		return nil, err
	}
	offered := make(map[uuid.UUID][]AppState, len(behind))
	for _, workID := range behind {
		found, err := s.WorkApps(ctx, userID, workID)
		if errors.Is(err, ErrWorkNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, state := range found.Items {
			if state.UpdateAvailable && state.CanReceive {
				offered[workID] = append(offered[workID], state)
			}
		}
	}
	return offered, nil
}

// worksInstalledBehind keeps only the works one of the account's connected apps holds an older copy of.
func (s *Sends) worksInstalledBehind(
	ctx context.Context,
	userID uuid.UUID,
	workIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	if len(workIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		select distinct entry.work_id
		  from app_library_entries entry
		  join connected_apps app on app.id = entry.connected_app_id
		  join works subject on subject.id = entry.work_id
		  join work_versions published on published.id = subject.published_version_id
		 where app.user_id = $1
		   and app.revoked_at is null
		   and entry.work_id = any($2::uuid[])
		   and entry.version_number < published.number
		   and subject.deleted_at is null
		   and subject.taken_down_at is null
		   and subject.lifecycle = 'published'
	`, userID, workIDs)
	if err != nil {
		return nil, fmt.Errorf("find the works a connected app holds an older copy of: %w", err)
	}
	defer rows.Close()
	behind := make([]uuid.UUID, 0, len(workIDs))
	for rows.Next() {
		var workID uuid.UUID
		if err := rows.Scan(&workID); err != nil {
			return nil, fmt.Errorf("read a work a connected app holds an older copy of: %w", err)
		}
		behind = append(behind, workID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find the works a connected app holds an older copy of: %w", err)
	}
	return behind, nil
}

func holdsPermission(permissions []string, wanted Permission) bool {
	return slices.Contains(permissions, string(wanted))
}

func sendFrom(
	id pgtype.UUID,
	appID pgtype.UUID,
	workID pgtype.UUID,
	state string,
	reason pgtype.Text,
	queuedAt pgtype.Timestamptz,
	settledAt pgtype.Timestamptz,
	expiresAt pgtype.Timestamptz,
	updatesInstall bool,
) Send {
	return Send{
		ID: uuid.UUID(id.Bytes), ConnectedAppID: uuid.UUID(appID.Bytes),
		WorkID: uuid.UUID(workID.Bytes), State: State(state),
		Reason: Reason(reason.String), QueuedAt: queuedAt.Time,
		SettledAt: optionalTime(settledAt), ExpiresAt: expiresAt.Time,
		UpdatesInstall: updatesInstall,
	}
}
