package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AttemptStale is how long a claimed key waits for its request before another
// request may take it over. It outlasts the longest deadline a route carries.
const AttemptStale = 20 * time.Minute

// AttemptRetention is how long a finished outcome is kept for a retry.
const AttemptRetention = 24 * time.Hour

// MaxAttemptResponse is the largest answer worth keeping for a retry.
const MaxAttemptResponse = 1 << 20

// Attempt is what an idempotency key already knows about the request carrying it.
type Attempt struct {
	Fresh       bool
	Running     bool
	Fingerprint []byte
	Status      int
	Response    []byte
}

// ClaimAttempt takes one key for one credential and operation, or answers what
// the earlier request under that key did.
func (s *Service) ClaimAttempt(
	ctx context.Context,
	tokenID uuid.UUID,
	operation string,
	key string,
) (Attempt, error) {
	var claimed bool
	err := s.pool.QueryRow(ctx, `
		insert into publication_idempotency (token_id, operation, key)
		values ($1, $2, $3)
		on conflict (token_id, operation, key) do update
		   set claimed_at = now()
		 where publication_idempotency.completed_at is null
		   and publication_idempotency.claimed_at < $4
		returning true
	`, tokenID, operation, key, time.Now().Add(-AttemptStale)).Scan(&claimed)
	if err == nil {
		return Attempt{Fresh: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, fmt.Errorf("claim an idempotency key: %w", err)
	}
	var found Attempt
	var status *int
	err = s.pool.QueryRow(ctx, `
		select fingerprint, status, response, completed_at is null
		  from publication_idempotency
		 where token_id = $1 and operation = $2 and key = $3
	`, tokenID, operation, key).Scan(&found.Fingerprint, &status, &found.Response, &found.Running)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{Fresh: true}, nil
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("read an idempotency key: %w", err)
	}
	if status != nil {
		found.Status = *status
	}
	return found, nil
}

// FinishAttempt keeps what the claimed key produced so a retry gets it back.
func (s *Service) FinishAttempt(
	ctx context.Context,
	tokenID uuid.UUID,
	operation string,
	key string,
	fingerprint []byte,
	status int,
	response []byte,
) error {
	_, err := s.pool.Exec(ctx, `
		insert into publication_idempotency
		       (token_id, operation, key, fingerprint, status, response, completed_at)
		values ($1, $2, $3, $4, $5, $6, now())
		on conflict (token_id, operation, key) do update
		   set fingerprint = excluded.fingerprint,
		       status = excluded.status,
		       response = excluded.response,
		       completed_at = excluded.completed_at
	`, tokenID, operation, key, fingerprint, status, response)
	if err != nil {
		return fmt.Errorf("keep an idempotent outcome: %w", err)
	}
	return nil
}

// ReleaseAttempt hands a claimed key back when the request failed in a way the
// caller is meant to retry.
func (s *Service) ReleaseAttempt(
	ctx context.Context,
	tokenID uuid.UUID,
	operation string,
	key string,
) error {
	_, err := s.pool.Exec(ctx, `
		delete from publication_idempotency
		 where token_id = $1 and operation = $2 and key = $3 and completed_at is null
	`, tokenID, operation, key)
	if err != nil {
		return fmt.Errorf("release an idempotency key: %w", err)
	}
	return nil
}

// SweepInterval is how often the finished keys past their window are dropped.
const SweepInterval = time.Hour

// RunSweeper drops what the publication has outlived until the context is
// done: idempotency keys past their window, and rotated signing secrets past
// the overlap they were kept for.
func (s *Service) RunSweeper(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepAttempts(ctx); err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
			_, err := s.ForgetOldSecrets(ctx, s.now())
			if err != nil && ctx.Err() == nil && onError != nil {
				onError(err)
			}
		}
	}
}

// SweepAttempts drops the keys that have outlived the retry window.
func (s *Service) SweepAttempts(ctx context.Context) (int64, error) {
	command, err := s.pool.Exec(ctx, `
		delete from publication_idempotency where claimed_at < $1
	`, time.Now().Add(-AttemptRetention))
	if err != nil {
		return 0, fmt.Errorf("sweep idempotency keys: %w", err)
	}
	return command.RowsAffected(), nil
}
