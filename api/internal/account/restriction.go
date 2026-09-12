package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const restrictionReasonLimit = 500

var (
	ErrProfileRestricted = errors.New("profile is restricted")
	ErrNotRestricted     = errors.New("profile is not restricted")
)

type Restriction struct {
	Reason       string
	RestrictedBy string
	RestrictedAt time.Time
}

func (s *Service) ProfileRestriction(ctx context.Context, handle string) (Restriction, error) {
	var found Restriction
	var actor *string
	err := s.pool.QueryRow(ctx, `
		select restriction.reason, restriction.restricted_at, actor.username
		  from users account
		  join profile_restrictions restriction on restriction.user_id = account.id
		  left join users actor on actor.id = restriction.restricted_by
		 where account.username = $1
	`, handle).Scan(&found.Reason, &found.RestrictedAt, &actor)
	if errors.Is(err, pgx.ErrNoRows) {
		return Restriction{}, ErrNotRestricted
	}
	if err != nil {
		return Restriction{}, fmt.Errorf("read profile restriction: %w", err)
	}
	if actor != nil {
		found.RestrictedBy = *actor
	}
	return found, nil
}

func (s *Service) RestrictProfile(
	ctx context.Context,
	admin Account,
	handle, rawReason string,
) (Restriction, error) {
	reason := strings.TrimSpace(rawReason)
	if reason == "" || len([]rune(reason)) > restrictionReasonLimit {
		return Restriction{}, FieldError{
			Field:   "reason",
			Message: fmt.Sprintf("Give a reason of up to %d characters.", restrictionReasonLimit),
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Restriction{}, fmt.Errorf("begin profile restriction: %w", err)
	}
	defer tx.Rollback(ctx)
	subject, err := lockAccountByHandle(ctx, tx, handle)
	if err != nil {
		return Restriction{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into profile_restrictions (user_id, restricted_by, reason)
		values ($1, $2, $3)
		on conflict (user_id) do update
		   set restricted_by = excluded.restricted_by,
		       reason = excluded.reason,
		       restricted_at = now()
	`, subject, admin.ID, reason)
	if err != nil {
		return Restriction{}, fmt.Errorf("restrict profile: %w", err)
	}
	if err := recordRestrictionAudit(ctx, tx, admin.ID, subject, "restrict", reason); err != nil {
		return Restriction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Restriction{}, fmt.Errorf("commit profile restriction: %w", err)
	}
	return s.ProfileRestriction(ctx, handle)
}

func (s *Service) RestoreProfile(ctx context.Context, admin Account, handle string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin profile restoration: %w", err)
	}
	defer tx.Rollback(ctx)
	subject, err := lockAccountByHandle(ctx, tx, handle)
	if err != nil {
		return err
	}
	lifted, err := tx.Exec(ctx, `delete from profile_restrictions where user_id = $1`, subject)
	if err != nil {
		return fmt.Errorf("restore profile: %w", err)
	}
	if lifted.RowsAffected() == 0 {
		return ErrNotRestricted
	}
	if err := recordRestrictionAudit(ctx, tx, admin.ID, subject, "restore", ""); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit profile restoration: %w", err)
	}
	return nil
}

func (s *Service) refuseWhileRestricted(ctx context.Context, ownerID uuid.UUID) error {
	var restricted bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from profile_restrictions where user_id = $1)
	`, ownerID).Scan(&restricted)
	if err != nil {
		return fmt.Errorf("read profile restriction state: %w", err)
	}
	if restricted {
		return ErrProfileRestricted
	}
	return nil
}

func lockAccountByHandle(ctx context.Context, tx pgx.Tx, handle string) (uuid.UUID, error) {
	var accountID uuid.UUID
	err := tx.QueryRow(ctx, `
		select id from users where username = $1 for update
	`, handle).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrProfileNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("lock the account being restricted: %w", err)
	}
	return accountID, nil
}

func recordRestrictionAudit(
	ctx context.Context,
	tx pgx.Tx,
	actor, subject uuid.UUID,
	action, reason string,
) error {
	_, err := tx.Exec(ctx, `
		insert into profile_restriction_audits (id, actor_id, subject_id, action, reason)
		values ($1, $2, $3, $4, $5)
	`, uuid.New(), actor, subject, action, reason)
	if err != nil {
		return fmt.Errorf("record profile restriction audit: %w", err)
	}
	return nil
}
