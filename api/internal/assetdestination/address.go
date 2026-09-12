package assetdestination

import (
	"context"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/discord"
)

type preparedAddress struct {
	host       string
	sealed     []byte
	state      string
	guildID    *string
	channelID  *string
	verifiedAt *time.Time
}

func (s *Service) prepareAddress(ctx context.Context, kind, address string) (preparedAddress, error) {
	prepared := preparedAddress{state: Unverified}
	if kind == Discord {
		capability, found, err := discord.VerifyCapability(ctx, s.sender, address)
		if err != nil {
			return prepared, FieldError{"address", err.Error()}
		}
		address = capability.URL
		now := time.Now().UTC()
		prepared.host, prepared.state = "discord.com", Active
		prepared.guildID, prepared.channelID, prepared.verifiedAt = &found.GuildID, &found.ChannelID, &now
	} else {
		host, err := s.sender.Check(address)
		if err != nil {
			return prepared, FieldError{"address", err.Error()}
		}
		prepared.host = host
	}
	sealed, err := s.sealing.Seal([]byte(address))
	prepared.sealed = sealed
	return prepared, err
}
