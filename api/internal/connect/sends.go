package connect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Works interface {
	DeliverableWork(ctx context.Context, q db.DBTX, workID uuid.UUID) (Deliverable, error)
	SignedURL(path string) string
	ValidSignature(path, expires, signature string) bool
}

type Instances interface {
	Live(ctx context.Context, userID uuid.UUID) ([]Instance, error)
	LiveByID(ctx context.Context, userID, instanceID uuid.UUID) (Instance, error)
	Throttle(ctx context.Context, action, source string, limit int32, window time.Duration) error
}

type Settings struct {
	HoldFloor          time.Duration
	HoldCeiling        time.Duration
	Recheck            time.Duration
	Lease              time.Duration
	Retention          time.Duration
	SweepInterval      time.Duration
	Batch              int
	MaxAttempts        int
	PendingPerInstance int
	ConcurrentHolds    int
	MaxAcknowledged    int
	MaxLibraryEntries  int
}

func DefaultSettings() Settings {
	return Settings{
		HoldFloor:          25 * time.Second,
		HoldCeiling:        30 * time.Second,
		Recheck:            5 * time.Second,
		Lease:              dispatch.Life,
		Retention:          7 * 24 * time.Hour,
		SweepInterval:      5 * time.Minute,
		Batch:              10,
		MaxAttempts:        5,
		PendingPerInstance: 100,
		ConcurrentHolds:    512,
		MaxAcknowledged:    32,
		MaxLibraryEntries:  2000,
	}
}

const (
	actionQueue       = "delivery-queue"
	actionCollect     = "delivery-collect"
	actionSyncPart    = "library-sync"
	actionSyncWhole   = "library-snapshot"
	queueLimit        = 120
	collectLimit      = 400
	syncPartLimit     = 120
	syncWholeLimit    = 24
	deliveryPathStart = "/delivery/"
)

type Sends struct {
	pool      *pgxpool.Pool
	works     Works
	instances Instances
	settings  Settings
	waiting   *hub
	now       func() time.Time
}

func NewSends(
	pool *pgxpool.Pool,
	works Works,
	instances Instances,
	settings Settings,
) *Sends {
	return &Sends{
		pool: pool, works: works, instances: instances, settings: settings,
		waiting: newHub(settings.ConcurrentHolds), now: time.Now,
	}
}

func (s *Sends) Queue(
	ctx context.Context,
	userID uuid.UUID,
	instanceID uuid.UUID,
	workID uuid.UUID,
) (Delivery, error) {
	if err := s.instances.Throttle(
		ctx, actionQueue, userID.String(), queueLimit, time.Hour,
	); err != nil {
		return Delivery{}, err
	}
	instance, err := s.instances.LiveByID(ctx, userID, instanceID)
	if errors.Is(err, ErrInstanceNotFound) {
		return Delivery{}, ErrNoInstanceOfYours
	}
	if err != nil {
		return Delivery{}, err
	}
	if !instance.Grants(ScopeWorkReceive) {
		return Delivery{}, ErrMissingScope
	}
	sendable, err := s.works.DeliverableWork(ctx, s.pool, workID)
	if errors.Is(err, ErrNotDeliverable) {
		return Delivery{}, ErrWorkNotSendable
	}
	if err != nil {
		return Delivery{}, err
	}
	if !installs(instance.Capabilities, sendable) {
		return Delivery{}, ErrCannotInstall
	}
	if _, _, chosen := chooseTarget(
		instance.AcceptedTargets, sendable.Targets, sendable.HasOriginal,
	); !chosen {
		return Delivery{}, ErrNoTarget
	}

	queries := db.New(s.pool)
	waiting, err := queries.CountLiveDeliveries(ctx, uuidValue(instanceID))
	if err != nil {
		return Delivery{}, fmt.Errorf("count waiting deliveries: %w", err)
	}
	if waiting >= int64(s.settings.PendingPerInstance) {
		return Delivery{}, ErrQueueFull
	}
	row, err := queries.QueueDelivery(ctx, db.QueueDeliveryParams{
		ID: uuidValue(uuid.New()), InstanceID: uuidValue(instanceID),
		WorkID:    uuidValue(workID),
		ExpiresAt: timestamptz(s.now().Add(s.settings.Retention)),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return s.liveDelivery(ctx, instanceID, workID)
	}
	if err != nil {
		return Delivery{}, fmt.Errorf("queue a delivery: %w", err)
	}
	s.waiting.signal(instanceID)
	return deliveryFrom(
		row.ID, row.InstanceID, row.WorkID, row.State, row.SettledReason,
		row.QueuedAt, row.SettledAt, row.ExpiresAt, row.UpdatesInstall,
	), nil
}

func (s *Sends) liveDelivery(
	ctx context.Context,
	instanceID uuid.UUID,
	workID uuid.UUID,
) (Delivery, error) {
	row, err := db.New(s.pool).LiveDeliveryForWork(ctx, db.LiveDeliveryForWorkParams{
		InstanceID: uuidValue(instanceID), WorkID: uuidValue(workID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Delivery{}, ErrDeliveryNotFound
	}
	if err != nil {
		return Delivery{}, fmt.Errorf("read the waiting delivery: %w", err)
	}
	return deliveryFrom(
		row.ID, row.InstanceID, row.WorkID, row.State, row.SettledReason,
		row.QueuedAt, row.SettledAt, row.ExpiresAt, row.UpdatesInstall,
	), nil
}

func (s *Sends) Discard(ctx context.Context, userID, deliveryID uuid.UUID) error {
	discarded, err := db.New(s.pool).DiscardDelivery(ctx, db.DiscardDeliveryParams{
		DeliveryID: uuidValue(deliveryID), UserID: uuidValue(userID),
	})
	if err != nil {
		return fmt.Errorf("discard a delivery: %w", err)
	}
	if discarded == 0 {
		return ErrDeliveryNotFound
	}
	return nil
}

func (s *Sends) WorkInstances(
	ctx context.Context,
	userID uuid.UUID,
	workID uuid.UUID,
) (WorkInstances, error) {
	queries := db.New(s.pool)
	generation, err := queries.SendableWorkGeneration(ctx, uuidValue(workID))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkInstances{}, ErrWorkNotFound
	}
	if err != nil {
		return WorkInstances{}, fmt.Errorf("read the work to send: %w", err)
	}
	sendable, err := s.works.DeliverableWork(ctx, s.pool, workID)
	if errors.Is(err, ErrNotDeliverable) {
		return WorkInstances{}, ErrWorkNotFound
	}
	if err != nil {
		return WorkInstances{}, fmt.Errorf("read available delivery formats: %w", err)
	}
	rows, err := queries.WorkInstanceStates(ctx, db.WorkInstanceStatesParams{
		WorkID: uuidValue(workID), UserID: uuidValue(userID),
	})
	if err != nil {
		return WorkInstances{}, fmt.Errorf("read instance state for a work: %w", err)
	}
	found := WorkInstances{ContentGeneration: int(generation), Items: []InstanceState{}}
	for _, row := range rows {
		_, _, canReceive := chooseTarget(
			row.AcceptedTargets, sendable.Targets, sendable.HasOriginal,
		)
		state := InstanceState{
			InstanceID:      uuid.UUID(row.ID.Bytes),
			ApplicationName: row.ApplicationName,
			InstanceName:    row.InstanceName,
			LastSeenAt:      optionalTime(row.LastSeenAt),
			CanReceive: holdsScope(row.Scopes, ScopeWorkReceive) &&
				installs(row.Capabilities, sendable) && canReceive,
			ReportsLibrary: holdsScope(row.Scopes, ScopeLibrarySync),
		}
		if row.DeliveryID.Valid {
			waiting := deliveryFrom(
				row.DeliveryID, row.ID, uuidValue(workID), row.DeliveryState,
				row.SettledReason, row.QueuedAt, row.SettledAt, row.ExpiresAt,
				row.UpdatesInstall,
			)
			state.Delivery = &waiting
		}
		if row.InstalledGeneration.Valid {
			installed := int(row.InstalledGeneration.Int32)
			state.InstalledGeneration = &installed
			state.UpdateAvailable = installed < found.ContentGeneration
		}
		found.Items = append(found.Items, state)
	}
	return found, nil
}

// UpdatableInstances names the account's instances that hold an older copy of a work and can receive it.
func (s *Sends) UpdatableInstances(
	ctx context.Context,
	userID uuid.UUID,
	workIDs []uuid.UUID,
) (map[uuid.UUID][]InstanceState, error) {
	behind, err := s.worksInstalledBehind(ctx, userID, workIDs)
	if err != nil {
		return nil, err
	}
	offered := make(map[uuid.UUID][]InstanceState, len(behind))
	for _, workID := range behind {
		found, err := s.WorkInstances(ctx, userID, workID)
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

// worksInstalledBehind keeps only the works one of the account's instances holds an older copy of.
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
		  from instance_library_entries entry
		  join linked_instances instance on instance.id = entry.instance_id
		  join works subject on subject.id = entry.work_id
		 where instance.user_id = $1
		   and instance.revoked_at is null
		   and entry.work_id = any($2::uuid[])
		   and entry.content_generation < subject.content_generation
		   and subject.deleted_at is null
		   and subject.withheld_at is null
		   and subject.lifecycle = 'published'
	`, userID, workIDs)
	if err != nil {
		return nil, fmt.Errorf("find the works an instance holds an older copy of: %w", err)
	}
	defer rows.Close()
	behind := make([]uuid.UUID, 0, len(workIDs))
	for rows.Next() {
		var workID uuid.UUID
		if err := rows.Scan(&workID); err != nil {
			return nil, fmt.Errorf("read a work an instance holds an older copy of: %w", err)
		}
		behind = append(behind, workID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find the works an instance holds an older copy of: %w", err)
	}
	return behind, nil
}

func holdsScope(scopes []string, wanted Scope) bool {
	for _, scope := range scopes {
		if Scope(scope) == wanted {
			return true
		}
	}
	return false
}

func deliveryFrom(
	id pgtype.UUID,
	instanceID pgtype.UUID,
	workID pgtype.UUID,
	state string,
	reason pgtype.Text,
	queuedAt pgtype.Timestamptz,
	settledAt pgtype.Timestamptz,
	expiresAt pgtype.Timestamptz,
	updatesInstall bool,
) Delivery {
	return Delivery{
		ID: uuid.UUID(id.Bytes), InstanceID: uuid.UUID(instanceID.Bytes),
		WorkID: uuid.UUID(workID.Bytes), State: State(state),
		Reason: Reason(reason.String), QueuedAt: queuedAt.Time,
		SettledAt: optionalTime(settledAt), ExpiresAt: expiresAt.Time,
		UpdatesInstall: updatesInstall,
	}
}
