package publication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const roleNameLimit = 48

var ErrNotDiscord = errors.New("the destination is not a Discord channel")

type ChannelEdit struct {
	Name     string
	Address  string
	RoleID   string
	RoleName string
}

func (s *Service) AddChannel(
	ctx context.Context,
	actor uuid.UUID,
	in ChannelEdit,
) (Destination, error) {
	name := checkDestinationName(in.Name)
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Destination{}, err
	}
	capability, found, err := s.askDiscord(ctx, in.Address)
	if err != nil {
		return Destination{}, err
	}
	sealed, err := s.sealing.Seal([]byte(capability.URL))
	if err != nil {
		return Destination{}, fmt.Errorf("seal the Discord capability: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord destination: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into publication_destinations
		       (id, kind, name, host, address, events, state, verified_at,
		        guild_id, channel_id, webhook_name, role_id, role_name, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, now(), $8, $9, $10, $11, $12, $13)
	`, id, KindDiscord, name, discordHost, sealed, []string{EventPublished}, DestinationActive,
		found.GuildID, found.ChannelID, found.Name, role.id, role.name, actor)
	if err != nil {
		return Destination{}, fmt.Errorf("record the Discord destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.added", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord destination: %w", err)
	}
	return s.Destination(ctx, id)
}

func (s *Service) UpdateChannel(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in ChannelEdit,
) (Destination, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if current.Kind != KindDiscord {
		return Destination{}, ErrNotDiscord
	}
	name := checkDestinationName(in.Name)
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Destination{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord destination update: %w", err)
	}
	defer tx.Rollback(ctx)
	if strings.TrimSpace(in.Address) == "" {
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, role_id = $3, role_name = $4, updated_at = now()
			 where id = $1
		`, id, name, role.id, role.name)
	} else {
		var capability discord.Capability
		var found discord.Webhook
		if capability, found, err = s.askDiscord(ctx, in.Address); err != nil {
			return Destination{}, err
		}
		var sealed []byte
		if sealed, err = s.sealing.Seal([]byte(capability.URL)); err != nil {
			return Destination{}, fmt.Errorf("seal the Discord capability: %w", err)
		}
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, address = $3, guild_id = $4, channel_id = $5, webhook_name = $6,
			       role_id = $7, role_name = $8, state = $9, verified_at = now(),
			       disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, sealed, found.GuildID, found.ChannelID, found.Name,
			role.id, role.name, DestinationActive)
	}
	if err != nil {
		return Destination{}, fmt.Errorf("update the Discord destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.updated", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord destination update: %w", err)
	}
	return s.Destination(ctx, id)
}

func (s *Service) provenByDiscord(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Destination, error) {
	capability, _, err := s.channelOf(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	_, found, err := s.askDiscord(ctx, capability.URL)
	if err != nil {
		return Destination{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin Discord verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, guild_id = $3, channel_id = $4, webhook_name = $5,
		       verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, DestinationActive, found.GuildID, found.ChannelID, found.Name)
	if err != nil {
		return Destination{}, fmt.Errorf("put the Discord destination back in service: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.verified", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit Discord verification: %w", err)
	}
	return s.Destination(ctx, id)
}

const discordHost = "discord.com"

func (s *Service) askDiscord(
	ctx context.Context,
	address string,
) (discord.Capability, discord.Webhook, error) {
	capability, found, err := discord.VerifyCapability(ctx, s.sender, address)
	if err != nil {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field: "address", Message: capitalize(strings.TrimSuffix(err.Error(), ".")) + ".", cause: err,
		}
	}
	return capability, found, nil
}

type approved struct {
	id   *string
	name *string
}

func checkRole(roleID, roleName string) (approved, error) {
	id := strings.TrimSpace(roleID)
	name := strings.TrimSpace(roleName)
	if id == "" && name == "" {
		return approved{}, nil
	}
	if err := discord.CheckRole(id); err != nil {
		return approved{}, FieldError{
			Field: "roleId", Message: capitalize(err.Error()) + ".", cause: err,
		}
	}
	if name == "" || len(name) > roleNameLimit {
		return approved{}, FieldError{
			Field:   "roleName",
			Message: fmt.Sprintf("Name the role in %d characters or fewer.", roleNameLimit),
		}
	}
	return approved{id: &id, name: &name}, nil
}

func (s *Service) announceOnDiscord(
	ctx context.Context,
	held carried,
	destinationID uuid.UUID,
	now time.Time,
) error {
	if held.Reclaimed {
		return s.ledger.Record(ctx, held.Work, outbox.DiscordUnconfirmed, now)
	}
	summary, err := s.summary(ctx, held.EventID)
	if err != nil {
		return err
	}
	capability, role, err := s.channelOf(ctx, destinationID)
	if err != nil {
		return err
	}
	if !held.Ping {
		role = ""
	}
	body, err := announcementOf(summary, role).Body()
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, capability.Confirming(), nil, body)
	if err != nil {
		return s.ledger.Record(ctx, held.Work, outbox.DiscordUnconfirmed, now)
	}
	said, message := outbox.ReadAnnouncement(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if message != "" {
		if err := s.ledger.KeepMessage(ctx, held.ID, message); err != nil {
			return err
		}
	}
	if err := s.ledger.Record(ctx, held.Work, said, now); err != nil {
		return err
	}
	if said.Gone {
		return s.retireDestination(ctx, destinationID)
	}
	return nil
}

func announcementOf(said sent, role string) discord.Announcement {
	one := discord.Announcement{
		Title:    said.Post.Title,
		Summary:  said.Post.Summary,
		URL:      said.Post.URL,
		Image:    said.Post.SocialImage,
		Category: said.Post.Category.Label,
		Note:     said.Note,
		Role:     role,
		Author:   discord.Author{Name: said.Post.Byline.Name, URL: said.Post.Byline.URL},
		At:       said.Post.PublishedAt,
	}
	if said.Post.Release != nil {
		one.Version = said.Post.Release.Version
	}
	return one
}

func (s *Service) channelOf(
	ctx context.Context,
	id uuid.UUID,
) (discord.Capability, string, error) {
	var sealed []byte
	var role *string
	err := s.pool.QueryRow(ctx, `
		select address, role_id from publication_destinations where id = $1
	`, id).Scan(&sealed, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return discord.Capability{}, "", ErrDestinationNotFound
	}
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("read the Discord configuration: %w", err)
	}
	address, err := s.sealing.Open(sealed)
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("open the Discord capability: %w", err)
	}
	capability, err := discord.ReadCapability(string(address))
	if err != nil {
		return discord.Capability{}, "", fmt.Errorf("read the Discord capability: %w", err)
	}
	if role == nil {
		return capability, "", nil
	}
	return capability, *role, nil
}
