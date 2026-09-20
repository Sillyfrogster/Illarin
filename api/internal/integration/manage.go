package integration

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) Update(ctx context.Context, owner, id uuid.UUID, name, address *string) (Integration, error) {
	held, err := s.Get(ctx, owner, id)
	if err != nil {
		return Integration{}, err
	}
	if name != nil {
		held.Name = strings.TrimSpace(*name)
		if held.Name == "" || utf8.RuneCountInString(held.Name) > 48 {
			return Integration{}, FieldError{"name", "Name the integration in 1 to 48 characters."}
		}
	}
	var prepared preparedAddress
	if address != nil {
		prepared, err = s.prepareAddress(ctx, held.Type, *address)
		if err != nil {
			return Integration{}, err
		}
		held.Host = prepared.host
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `
		update work_integrations
		   set name = $4, host = $5, address = coalesce($6, address),
		       state = case when $6::bytea is null then state else $7 end,
		       verified_at = case when $6::bytea is null then verified_at else $8 end,
		       guild_id = coalesce($9, guild_id), channel_id = coalesce($10, channel_id),
		       disabled_at = case when $6::bytea is null then disabled_at else null end,
		       version = version+1, updated_at = now()
		 where owner_id = $1 and id = $2 and version = $3
	`, owner, id, held.version, held.Name, held.Host, prepared.sealed,
		prepared.state, prepared.verifiedAt, prepared.guildID, prepared.channelID)
	if err != nil {
		return Integration{}, err
	}
	if result.RowsAffected() != 1 {
		return Integration{}, ErrChanged
	}
	if address != nil {
		if err := s.ledger.StopTo(ctx, tx, id, dispatch.Stopped(dispatch.Moved)); err != nil {
			return Integration{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, err
	}
	return s.Get(ctx, owner, id)
}

func (s *Service) Disable(ctx context.Context, owner, id uuid.UUID) (Integration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `
		update work_integrations
		   set state = 'disabled', disabled_at = now(), version = version+1, updated_at = now()
		 where owner_id = $1 and id = $2
	`, owner, id)
	if err != nil {
		return Integration{}, err
	}
	if result.RowsAffected() != 1 {
		return Integration{}, ErrNotFound
	}
	if err := s.ledger.StopTo(ctx, tx, id, dispatch.Stopped(dispatch.Disabled)); err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, err
	}
	return s.Get(ctx, owner, id)
}

func (s *Service) Remove(ctx context.Context, owner, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var held bool
	err = tx.QueryRow(ctx, `
		select true from work_integrations where owner_id = $1 and id = $2 for update
	`, owner, id).Scan(&held)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := s.ledger.StopTo(ctx, tx, id, dispatch.Stopped(dispatch.Removed)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from work_integrations where id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) RotateSecret(ctx context.Context, owner, id uuid.UUID) (Added, error) {
	held, err := s.Get(ctx, owner, id)
	if err != nil {
		return Added{}, err
	}
	if held.Type != Webhook {
		return Added{}, FieldError{"type", "Discord credentials are changed by replacing the webhook address."}
	}
	if held.PreviousSecretUntil != nil && time.Now().Before(*held.PreviousSecretUntil) {
		return Added{}, FieldError{"secret", "Wait until the previous signing secret expires before rotating again."}
	}
	secret, err := dispatch.MintSecret()
	if err != nil {
		return Added{}, err
	}
	sealed, err := s.sealing.Seal([]byte(secret))
	if err != nil {
		return Added{}, err
	}
	now := time.Now().UTC()
	result, err := s.pool.Exec(ctx, `
		update work_integrations
		   set signing_secret = $4, signing_secret_set_at = $5, previous_secret = signing_secret,
		       previous_secret_until = $6, version = version+1, updated_at = $5
		 where owner_id = $1 and id = $2 and version = $3
	`, owner, id, held.version, sealed, now, now.Add(dispatch.SecretOverlap))
	if err != nil {
		return Added{}, err
	}
	if result.RowsAffected() != 1 {
		return Added{}, ErrChanged
	}
	found, err := s.Get(ctx, owner, id)
	return Added{Integration: found, Secret: secret}, err
}

func (s *Service) ForgetOldSecrets(ctx context.Context, now time.Time) (int64, error) {
	result, err := s.pool.Exec(ctx, `
		update work_integrations set previous_secret = null, previous_secret_until = null
		 where previous_secret_until <= $1
	`, now)
	return result.RowsAffected(), err
}

func (s *Service) RunCleanup(ctx context.Context, report func(error)) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if _, err := s.ForgetOldSecrets(ctx, time.Now()); err != nil && ctx.Err() == nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
