package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const SecretOverlap = 24 * time.Hour

type RotatedSecret struct {
	Destination Destination
	Secret      string
	OldUntil    time.Time
}

func (s *Service) RotateSecret(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (RotatedSecret, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return RotatedSecret{}, err
	}
	if current.Kind == KindDiscord {
		return RotatedSecret{}, ErrNotWebhook
	}
	secret, err := webhook.MintSecret()
	if err != nil {
		return RotatedSecret{}, err
	}
	sealed, err := s.sealing.Seal([]byte(secret))
	if err != nil {
		return RotatedSecret{}, fmt.Errorf("seal the signing secret: %w", err)
	}
	now := s.now().UTC()
	until := now.Add(SecretOverlap)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return RotatedSecret{}, fmt.Errorf("begin secret rotation: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set signing_secret = $2, signing_secret_set_at = $3,
		       previous_secret = signing_secret, previous_secret_until = $4,
		       updated_at = $3
		 where id = $1
	`, id, sealed, now, until)
	if err != nil {
		return RotatedSecret{}, fmt.Errorf("rotate the signing secret: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.secret.rotated", DestinationID: &id,
	})
	if err != nil {
		return RotatedSecret{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RotatedSecret{}, fmt.Errorf("commit secret rotation: %w", err)
	}
	found, err := s.Destination(ctx, id)
	if err != nil {
		return RotatedSecret{}, err
	}
	return RotatedSecret{Destination: found, Secret: secret, OldUntil: until}, nil
}

func (s *Service) ForgetOldSecrets(ctx context.Context, now time.Time) (int64, error) {
	command, err := s.pool.Exec(ctx, `
		update publication_destinations
		   set previous_secret = null, previous_secret_until = null, updated_at = now()
		 where previous_secret_until is not null and previous_secret_until <= $1
	`, now)
	if err != nil {
		return 0, fmt.Errorf("forget the rotated secrets: %w", err)
	}
	return command.RowsAffected(), nil
}

func (s *Service) endpointOf(ctx context.Context, id uuid.UUID) (string, []string, error) {
	var sealedAddress, sealedSecret, sealedOld []byte
	var until *time.Time
	err := s.pool.QueryRow(ctx, `
		select address, signing_secret, previous_secret, previous_secret_until
		  from publication_destinations where id = $1
	`, id).Scan(&sealedAddress, &sealedSecret, &sealedOld, &until)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrDestinationNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("read the destination configuration: %w", err)
	}
	address, err := s.sealing.Open(sealedAddress)
	if err != nil {
		return "", nil, fmt.Errorf("open the endpoint address: %w", err)
	}
	secret, err := s.sealing.Open(sealedSecret)
	if err != nil {
		return "", nil, fmt.Errorf("open the signing secret: %w", err)
	}
	secrets := []string{string(secret)}
	if sealedOld != nil && until != nil && s.now().Before(*until) {
		old, err := s.sealing.Open(sealedOld)
		if err != nil {
			return "", nil, fmt.Errorf("open the rotated signing secret: %w", err)
		}
		secrets = append(secrets, string(old))
	}
	return string(address), secrets, nil
}

func (s *Service) retireDestination(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin retiring a destination: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, disabled_at = now(), updated_at = now()
		 where id = $1 and state <> $2
	`, id, DestinationDisabled)
	if err != nil {
		return fmt.Errorf("retire the destination: %w", err)
	}
	if err := stopDeliveriesTo(ctx, tx, id, SettledDisabled); err != nil {
		return err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: uuid.Nil, Credential: CredentialSystem,
		Action: "destination.gone", DestinationID: &id,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit retiring a destination: %w", err)
	}
	return nil
}
