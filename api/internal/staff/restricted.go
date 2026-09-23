package staff

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const restrictedReasonLimit = 500

type Restricted struct {
	Reason       string
	RestrictedBy string
	RestrictedAt time.Time
}

func (s *Service) RestrictedProfile(ctx context.Context, handle string) (Restricted, error) {
	var found Restricted
	var actor *string
	err := s.pool.QueryRow(ctx, `
		select restricted.reason, restricted.restricted_at, actor.username
		  from users account
		  join restricted_profiles restricted on restricted.user_id = account.id
		  left join users actor on actor.id = restricted.restricted_by
		 where account.username = $1
	`, handle).Scan(&found.Reason, &found.RestrictedAt, &actor)
	if errors.Is(err, pgx.ErrNoRows) {
		return Restricted{}, ErrNotRestricted
	}
	if err != nil {
		return Restricted{}, fmt.Errorf("read restricted profile: %w", err)
	}
	if actor != nil {
		found.RestrictedBy = *actor
	}
	return found, nil
}

func (s *Service) RestrictProfile(
	ctx context.Context,
	admin api.Account,
	handle, rawReason string,
) (Restricted, error) {
	reason := strings.TrimSpace(rawReason)
	if reason == "" || len([]rune(reason)) > restrictedReasonLimit {
		return Restricted{}, ErrInvalidReason
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Restricted{}, fmt.Errorf("begin restricting a profile: %w", err)
	}
	defer tx.Rollback(ctx)
	subject, err := lockAccountByHandle(ctx, tx, handle)
	if err != nil {
		return Restricted{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into restricted_profiles (user_id, restricted_by, reason)
		values ($1, $2, $3)
		on conflict (user_id) do update
		   set restricted_by = excluded.restricted_by,
		       reason = excluded.reason,
		       restricted_at = now()
	`, subject, admin.ID, reason)
	if err != nil {
		return Restricted{}, fmt.Errorf("restrict profile: %w", err)
	}
	if err := recordRestrictedProfileAudit(ctx, tx, admin.ID, subject, "restrict", reason); err != nil {
		return Restricted{}, err
	}
	if err := notify.Record(ctx, tx, notify.Event{
		Type: notify.ProfileRestricted, Account: &subject, Words: notify.Words{Reason: reason},
	}); err != nil {
		return Restricted{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Restricted{}, fmt.Errorf("commit restricting a profile: %w", err)
	}
	return s.RestrictedProfile(ctx, handle)
}

func (s *Service) RestoreProfile(ctx context.Context, admin api.Account, handle string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin profile restoration: %w", err)
	}
	defer tx.Rollback(ctx)
	subject, err := lockAccountByHandle(ctx, tx, handle)
	if err != nil {
		return err
	}
	lifted, err := tx.Exec(ctx, `delete from restricted_profiles where user_id = $1`, subject)
	if err != nil {
		return fmt.Errorf("restore profile: %w", err)
	}
	if lifted.RowsAffected() == 0 {
		return ErrNotRestricted
	}
	if err := recordRestrictedProfileAudit(ctx, tx, admin.ID, subject, "restore", ""); err != nil {
		return err
	}
	if err := notify.Record(ctx, tx, notify.Event{
		Type: notify.ProfileRestored, Account: &subject,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit profile restoration: %w", err)
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

func recordRestrictedProfileAudit(
	ctx context.Context,
	tx pgx.Tx,
	actor, subject uuid.UUID,
	action, reason string,
) error {
	_, err := tx.Exec(ctx, `
		insert into restricted_profile_audits (id, actor_id, subject_id, action, reason)
		values ($1, $2, $3, $4, $5)
	`, uuid.New(), actor, subject, action, reason)
	if err != nil {
		return fmt.Errorf("record restricted profile audit: %w", err)
	}
	return nil
}
