package delivery

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/linking"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const holdMargin = 2 * time.Second

func (s *Service) Collect(
	ctx context.Context,
	instance linking.Instance,
	acknowledged []uuid.UUID,
) (Collected, error) {
	if len(acknowledged) > s.settings.MaxAcknowledged {
		return Collected{}, ErrAcknowledgement
	}
	if err := s.instances.Throttle(
		ctx, actionCollect, instance.ID.String(), collectLimit, time.Hour,
	); err != nil {
		return Collected{}, throttled(err)
	}
	if len(acknowledged) > 0 {
		if _, err := db.New(s.pool).AcknowledgeDeliveries(ctx, db.AcknowledgeDeliveriesParams{
			InstanceID: uuidValue(instance.ID), DeliveryIds: uuidValues(acknowledged),
		}); err != nil {
			return Collected{}, fmt.Errorf("acknowledge deliveries: %w", err)
		}
	}

	held, admitted := s.waiting.hold(instance.ID)
	if !admitted {
		return Collected{}, ErrTooManyCollectors
	}
	defer s.waiting.release(instance.ID, held)

	recheck := time.NewTicker(s.settings.Recheck)
	defer recheck.Stop()
	waitedOut := time.NewTimer(s.hold(ctx))
	defer waitedOut.Stop()
	for {
		collected, err := s.claim(ctx, instance)
		if err != nil {
			return Collected{}, err
		}
		if len(collected.Work) > 0 || len(collected.Withheld) > 0 {
			return collected, nil
		}
		select {
		case <-held.work:
		case <-recheck.C:
		case <-held.superseded:
			return Collected{}, nil
		case <-waitedOut.C:
			return Collected{}, nil
		case <-ctx.Done():
			return Collected{}, ctx.Err()
		}
	}
}

func (s *Service) hold(ctx context.Context) time.Duration {
	spread := s.settings.HoldCeiling - s.settings.HoldFloor
	wait := s.settings.HoldFloor
	if spread > 0 {
		wait += rand.N(spread)
	}
	deadline, set := ctx.Deadline()
	if !set {
		return wait
	}
	allowed := time.Until(deadline) - holdMargin
	if allowed < wait {
		wait = allowed
	}
	if wait < 0 {
		wait = 0
	}
	return wait
}

func (s *Service) claim(ctx context.Context, instance linking.Instance) (Collected, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Collected{}, fmt.Errorf("begin a delivery claim: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	if _, err := queries.AbandonExhaustedDeliveries(ctx, db.AbandonExhaustedDeliveriesParams{
		InstanceID: uuidValue(instance.ID), MaxAttempts: int32(s.settings.MaxAttempts),
	}); err != nil {
		return Collected{}, fmt.Errorf("abandon exhausted deliveries: %w", err)
	}
	claimed, err := queries.ClaimDeliveries(ctx, db.ClaimDeliveriesParams{
		LeaseExpiresAt: timestamptz(s.now().Add(s.settings.Lease)),
		InstanceID:     uuidValue(instance.ID),
		MaxAttempts:    int32(s.settings.MaxAttempts),
		BatchSize:      int32(s.settings.Batch),
	})
	if err != nil {
		return Collected{}, fmt.Errorf("claim deliveries: %w", err)
	}
	work := make([]Work, 0, len(claimed))
	for _, row := range claimed {
		released, err := s.release(ctx, tx, instance, row)
		if err != nil {
			return Collected{}, err
		}
		if released != nil {
			work = append(work, *released)
		}
	}
	withheld, err := takeWithheldNotices(ctx, queries, instance.ID)
	if err != nil {
		return Collected{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Collected{}, fmt.Errorf("commit a delivery claim: %w", err)
	}
	return Collected{Work: work, Withheld: withheld}, nil
}

func (s *Service) release(
	ctx context.Context,
	tx pgx.Tx,
	instance linking.Instance,
	row db.ClaimDeliveriesRow,
) (*Work, error) {
	queries := db.New(tx)
	deliveryID := uuid.UUID(row.ID.Bytes)
	assetID := uuid.UUID(row.AssetID.Bytes)
	sendable, err := s.catalog.DeliverableAsset(ctx, tx, assetID)
	if errors.Is(err, asset.ErrNotDeliverable) || errors.Is(err, pgx.ErrNoRows) {
		return nil, stop(ctx, queries, row.ID, ReasonWithdrawn)
	}
	if err != nil {
		return nil, err
	}
	target, label, chosen := chooseTarget(
		instance.AcceptedTargets, sendable.Targets, sendable.HasOriginal,
	)
	if !chosen || !installs(instance.Capabilities, sendable) {
		return nil, stop(ctx, queries, row.ID, ReasonUnsupported)
	}
	if err := queries.SetDeliveryTarget(ctx, db.SetDeliveryTargetParams{
		ChosenTarget: textValue(target), ID: row.ID,
	}); err != nil {
		return nil, fmt.Errorf("record the chosen format: %w", err)
	}
	return &Work{
		ID: deliveryID, AssetID: assetID,
		ContentGeneration: sendable.ContentGeneration,
		Kind:              sendable.Kind, Name: sendable.Name,
		Format: target, Label: label,
		QueuedAt: row.QueuedAt.Time, LeaseExpiresAt: row.LeaseExpiresAt.Time,
		Artifacts: s.artifacts(deliveryID, sendable),
	}, nil
}

func stop(
	ctx context.Context,
	queries *db.Queries,
	id pgtype.UUID,
	reason Reason,
) error {
	if err := queries.FailDelivery(ctx, db.FailDeliveryParams{
		SettledReason: textValue(string(reason)), ID: id,
	}); err != nil {
		return fmt.Errorf("stop a delivery: %w", err)
	}
	return nil
}

func (s *Service) artifacts(deliveryID uuid.UUID, sendable asset.Deliverable) []Artifact {
	artifacts := make([]Artifact, 0, len(sendable.Pictures)+1)
	artifacts = append(artifacts, Artifact{
		Kind: ArtifactExport,
		URL:  s.catalog.SignedURL(deliveryPathStart + deliveryID.String() + "/export"),
	})
	for _, picture := range sendable.Pictures {
		mediaID := picture.MediaID
		artifacts = append(artifacts, Artifact{
			Kind: ArtifactPicture, URL: picture.URL, MediaID: &mediaID,
			Role: picture.Role, IsCover: picture.IsCover,
		})
	}
	return artifacts
}

func uuidValues(values []uuid.UUID) []pgtype.UUID {
	converted := make([]pgtype.UUID, len(values))
	for index, value := range values {
		converted[index] = uuidValue(value)
	}
	return converted
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
