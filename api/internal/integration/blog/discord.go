package blog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const roleNameLimit = 48

var ErrNotDiscord = errors.New("the integration is not a Discord channel")

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
) (Integration, error) {
	name := checkIntegrationName(in.Name)
	if name == "" {
		return Integration{}, FieldError{Field: "name", Message: "Give the integration a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Integration{}, err
	}
	capability, found, err := s.askDiscord(ctx, in.Address)
	if err != nil {
		return Integration{}, err
	}
	sealed, err := s.sealing.Seal([]byte(capability.URL))
	if err != nil {
		return Integration{}, fmt.Errorf("seal the Discord capability: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin Discord integration: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into blog_integrations
		       (id, type, name, host, address, announcements, state, verified_at,
		        guild_id, channel_id, webhook_name, role_id, role_name, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, now(), $8, $9, $10, $11, $12, $13)
	`, id, TypeDiscord, name, discordHost, sealed, []string{PostPublished}, IntegrationActive,
		found.GuildID, found.ChannelID, found.Name, role.id, role.name, actor)
	if err != nil {
		return Integration{}, fmt.Errorf("record the Discord integration: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.added", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit Discord integration: %w", err)
	}
	return s.Integration(ctx, id)
}

func (s *Service) UpdateChannel(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in ChannelEdit,
) (Integration, error) {
	current, err := s.Integration(ctx, id)
	if err != nil {
		return Integration{}, err
	}
	if current.Type != TypeDiscord {
		return Integration{}, ErrNotDiscord
	}
	name := checkIntegrationName(in.Name)
	if name == "" {
		return Integration{}, FieldError{Field: "name", Message: "Give the integration a name."}
	}
	role, err := checkRole(in.RoleID, in.RoleName)
	if err != nil {
		return Integration{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin Discord integration update: %w", err)
	}
	defer tx.Rollback(ctx)
	if strings.TrimSpace(in.Address) == "" {
		_, err = tx.Exec(ctx, `
			update blog_integrations
			   set name = $2, role_id = $3, role_name = $4, updated_at = now()
			 where id = $1
		`, id, name, role.id, role.name)
	} else {
		var capability discord.Capability
		var found discord.Webhook
		if capability, found, err = s.askDiscord(ctx, in.Address); err != nil {
			return Integration{}, err
		}
		var sealed []byte
		if sealed, err = s.sealing.Seal([]byte(capability.URL)); err != nil {
			return Integration{}, fmt.Errorf("seal the Discord capability: %w", err)
		}
		_, err = tx.Exec(ctx, `
			update blog_integrations
			   set name = $2, address = $3, guild_id = $4, channel_id = $5, webhook_name = $6,
			       role_id = $7, role_name = $8, state = $9, verified_at = now(),
			       disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, sealed, found.GuildID, found.ChannelID, found.Name,
			role.id, role.name, IntegrationActive)
	}
	if err != nil {
		return Integration{}, fmt.Errorf("update the Discord integration: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.updated", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit Discord integration update: %w", err)
	}
	return s.Integration(ctx, id)
}

func (s *Service) provenByDiscord(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Integration, error) {
	capability, _, err := s.channelOf(ctx, id)
	if err != nil {
		return Integration{}, err
	}
	_, found, err := s.askDiscord(ctx, capability.URL)
	if err != nil {
		return Integration{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin Discord verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update blog_integrations
		   set state = $2, guild_id = $3, channel_id = $4, webhook_name = $5,
		       verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, IntegrationActive, found.GuildID, found.ChannelID, found.Name)
	if err != nil {
		return Integration{}, fmt.Errorf("put the Discord integration back in service: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.verified", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit Discord verification: %w", err)
	}
	return s.Integration(ctx, id)
}

const discordHost = "discord.com"

func (s *Service) askDiscord(
	ctx context.Context,
	address string,
) (discord.Capability, discord.Webhook, error) {
	capability, found, err := discord.VerifyCapability(ctx, s.sender, address)
	if err != nil {
		return discord.Capability{}, discord.Webhook{}, FieldError{
			Field: "address", Message: capitalize(strings.TrimSuffix(err.Error(), ".")) + ".", Cause: err,
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
			Field: "roleId", Message: capitalize(err.Error()) + ".", Cause: err,
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
	integrationID uuid.UUID,
	now time.Time,
) error {
	if held.Reclaimed {
		return s.ledger.Record(ctx, held.Work, dispatch.DiscordUnconfirmed, now)
	}
	summary, err := s.Summary(ctx, s.pool, held.AnnouncementID)
	if err != nil {
		return err
	}
	capability, role, err := s.channelOf(ctx, integrationID)
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
		return s.ledger.Record(ctx, held.Work, dispatch.DiscordUnconfirmed, now)
	}
	said, message := dispatch.ReadAnnouncement(answer)
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
		return s.retireIntegration(ctx, integrationID)
	}
	return nil
}

func announcementOf(said Event, role string) discord.Announcement {
	return discord.Announcement{
		Title:    said.Post.Title,
		Summary:  said.Post.Summary,
		URL:      said.Post.URL,
		Image:    said.Post.LinkCardImage,
		Category: said.Post.Category.Label,
		Note:     said.Note,
		Role:     role,
		Author:   discord.Author{Name: said.Post.Byline.Name, URL: said.Post.Byline.URL},
		At:       said.Post.PublishedAt,
	}
}

func (s *Service) channelOf(
	ctx context.Context,
	id uuid.UUID,
) (discord.Capability, string, error) {
	var sealed []byte
	var role *string
	err := s.pool.QueryRow(ctx, `
		select address, role_id from blog_integrations where id = $1
	`, id).Scan(&sealed, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return discord.Capability{}, "", ErrIntegrationNotFound
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
