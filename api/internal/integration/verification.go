package integration

import (
	"context"
	"errors"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrChanged = errors.New("the integration changed while this request was running")

const EventVerification = "asset.endpoint.verification.v1"

type configuration struct {
	integrationType string
	version         int64
	address         string
	secrets         []string
}

func (s *Service) configuration(ctx context.Context, owner, id uuid.UUID) (configuration, error) {
	var found configuration
	var address, secret, old []byte
	var until *time.Time
	err := s.pool.QueryRow(ctx, `
		select type, version, address, signing_secret, previous_secret, previous_secret_until
		  from work_integrations where owner_id = $1 and id = $2
	`, owner, id).Scan(&found.integrationType, &found.version, &address, &secret, &old, &until)
	if errors.Is(err, pgx.ErrNoRows) {
		return found, ErrNotFound
	}
	if err != nil {
		return found, err
	}
	opened, err := s.sealing.Open(address)
	if err != nil {
		return found, err
	}
	found.address = string(opened)
	if secret != nil {
		opened, err = s.sealing.Open(secret)
		if err != nil {
			return found, err
		}
		found.secrets = append(found.secrets, string(opened))
	}
	if old != nil && until != nil && time.Now().Before(*until) {
		opened, err = s.sealing.Open(old)
		if err != nil {
			return found, err
		}
		found.secrets = append(found.secrets, string(opened))
	}
	return found, nil
}

func (s *Service) Verify(ctx context.Context, owner, id uuid.UUID) (Integration, error) {
	held, err := s.configuration(ctx, owner, id)
	if err != nil {
		return Integration{}, err
	}
	var guildID, channelID *string
	if held.integrationType == Discord {
		_, found, err := discord.VerifyCapability(ctx, s.sender, held.address)
		if err != nil {
			return Integration{}, FieldError{"address", err.Error()}
		}
		guildID, channelID = &found.GuildID, &found.ChannelID
	} else {
		if err := dispatch.VerifyEndpoint(ctx, s.sender, EventVerification, held.address, held.secrets, time.Now().UTC()); err != nil {
			return Integration{}, FieldError{"address", err.Error()}
		}
	}
	result, err := s.pool.Exec(ctx, `
		update work_integrations
		   set state = 'active', verified_at = now(), disabled_at = null, version = version+1,
		       updated_at = now(), guild_id = $4, channel_id = $5
		 where owner_id = $1 and id = $2 and version = $3
	`, owner, id, held.version, guildID, channelID)
	if err != nil {
		return Integration{}, err
	}
	if result.RowsAffected() != 1 {
		return Integration{}, ErrChanged
	}
	return s.Get(ctx, owner, id)
}
