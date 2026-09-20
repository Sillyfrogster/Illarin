package connect

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const holdMargin = 2 * time.Second

func (s *Sends) Collect(
	ctx context.Context,
	app ConnectedApp,
	acknowledged []uuid.UUID,
) (Collected, error) {
	if len(acknowledged) > s.settings.MaxAcknowledged {
		return Collected{}, ErrAcknowledgement
	}
	if err := s.apps.Throttle(
		ctx, actionCollect, app.ID.String(), collectLimit, time.Hour,
	); err != nil {
		return Collected{}, err
	}
	if len(acknowledged) > 0 {
		if _, err := db.New(s.pool).AcknowledgeSends(ctx, db.AcknowledgeSendsParams{
			ConnectedAppID: uuidValue(app.ID), SendIds: uuidValues(acknowledged),
		}); err != nil {
			return Collected{}, fmt.Errorf("acknowledge sends: %w", err)
		}
	}

	held, admitted := s.waiting.hold(app.ID)
	if !admitted {
		return Collected{}, ErrTooManyCollectors
	}
	defer s.waiting.release(app.ID, held)

	recheck := time.NewTicker(s.settings.Recheck)
	defer recheck.Stop()
	waitedOut := time.NewTimer(s.hold(ctx))
	defer waitedOut.Stop()
	for {
		collected, err := s.claim(ctx, app)
		if err != nil {
			return Collected{}, err
		}
		if len(collected.Work) > 0 || len(collected.Takedowns) > 0 {
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

func (s *Sends) hold(ctx context.Context) time.Duration {
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

func (s *Sends) claim(ctx context.Context, app ConnectedApp) (Collected, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Collected{}, fmt.Errorf("begin a send claim: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	if _, err := queries.AbandonExhaustedSends(ctx, db.AbandonExhaustedSendsParams{
		ConnectedAppID: uuidValue(app.ID), MaxAttempts: int32(s.settings.MaxAttempts),
	}); err != nil {
		return Collected{}, fmt.Errorf("abandon exhausted sends: %w", err)
	}
	claimed, err := queries.ClaimSends(ctx, db.ClaimSendsParams{
		LeaseExpiresAt: timestamptz(s.now().Add(s.settings.Lease)),
		ConnectedAppID: uuidValue(app.ID),
		MaxAttempts:    int32(s.settings.MaxAttempts),
		BatchSize:      int32(s.settings.Batch),
	})
	if err != nil {
		return Collected{}, fmt.Errorf("claim sends: %w", err)
	}
	work := make([]Work, 0, len(claimed))
	for _, row := range claimed {
		released, err := s.release(ctx, tx, app, row)
		if err != nil {
			return Collected{}, err
		}
		if released != nil {
			work = append(work, *released)
		}
	}
	takedowns, err := takeTakedownNotices(ctx, queries, app.ID)
	if err != nil {
		return Collected{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Collected{}, fmt.Errorf("commit a send claim: %w", err)
	}
	return Collected{Work: work, Takedowns: takedowns}, nil
}

func (s *Sends) release(
	ctx context.Context,
	tx pgx.Tx,
	app ConnectedApp,
	row db.ClaimSendsRow,
) (*Work, error) {
	queries := db.New(tx)
	sendID := uuid.UUID(row.ID.Bytes)
	workID := uuid.UUID(row.WorkID.Bytes)
	sendable, err := s.works.SendableWork(ctx, tx, workID)
	if errors.Is(err, ErrNotSendable) || errors.Is(err, pgx.ErrNoRows) {
		return nil, stop(ctx, queries, row.ID, ReasonWithdrawn)
	}
	if err != nil {
		return nil, err
	}
	chosenFormat, label, chosen := chooseFormat(
		app.AcceptedFormats, sendable.Formats, sendable.HasOriginal,
	)
	if !chosen || !installs(app.Declared, sendable) {
		return nil, stop(ctx, queries, row.ID, ReasonUnsupported)
	}
	if err := queries.SetSendFormat(ctx, db.SetSendFormatParams{
		ChosenFormat: textValue(chosenFormat), ID: row.ID,
	}); err != nil {
		return nil, fmt.Errorf("record the chosen format: %w", err)
	}
	return &Work{
		ID: sendID, WorkID: workID,
		VersionNumber: sendable.VersionNumber,
		Type:          sendable.Type, Name: sendable.Name,
		Format: chosenFormat, Label: label,
		QueuedAt: row.QueuedAt.Time, LeaseExpiresAt: row.LeaseExpiresAt.Time,
		Files: s.files(sendID, sendable),
	}, nil
}

func stop(
	ctx context.Context,
	queries *db.Queries,
	id pgtype.UUID,
	reason Reason,
) error {
	if err := queries.FailSend(ctx, db.FailSendParams{
		SettledReason: textValue(string(reason)), ID: id,
	}); err != nil {
		return fmt.Errorf("stop a send: %w", err)
	}
	return nil
}

func (s *Sends) files(sendID uuid.UUID, sendable Sendable) []File {
	files := make([]File, 0, len(sendable.Pictures)+1)
	files = append(files, File{
		Type: FileExport,
		URL:  s.works.SignedURL(sendPathStart + sendID.String() + "/export"),
	})
	for _, picture := range sendable.Pictures {
		mediaID := picture.MediaID
		files = append(files, File{
			Type: FilePicture, URL: picture.URL, MediaID: &mediaID,
			Role: picture.Role, IsCover: picture.IsCover,
		})
	}
	return files
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
