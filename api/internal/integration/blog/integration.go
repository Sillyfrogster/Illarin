package blog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	TypeWebhook = "webhook"
	TypeDiscord = "discord"
)

const (
	IntegrationUnverified = "unverified"
	IntegrationActive     = "active"
	IntegrationDisabled   = "disabled"
)

const integrationNameLimit = 48

var AnnouncementTypes = []string{PostPublished, PostUpdated, PostUnpublished}

var ErrAnnouncementTypeUnknown = errors.New("no such announcement")

var ErrNotWebhook = errors.New("the integration is a Discord channel")

var (
	ErrIntegrationNotFound = errors.New("no such blog integration")
	ErrIntegrationRefused  = errors.New("the integration is not one this post may send to")
	ErrIntegrationInactive = errors.New("the integration has not been verified")
)

type Integration struct {
	ID            uuid.UUID
	Type          string
	Name          string
	Host          string
	Address       string
	State         string
	Announcements []string
	Channel       *Channel
	SecretSetAt   time.Time
	OldUntil      *time.Time
	VerifiedAt    *time.Time
	DisabledAt    *time.Time
	CreatedAt     time.Time
}

type Channel struct {
	GuildID   string
	ChannelID string
	Webhook   string
	RoleID    string
	RoleName  string
}

type IntegrationEdit struct {
	Name          string
	Address       string
	Announcements *[]string
}

type IntegrationUpdate struct {
	Name          *string
	Address       *string
	Announcements *[]string
}

type AddedIntegration struct {
	Integration Integration
	Secret      string
}

type Choice struct {
	ID            uuid.UUID
	Name          string
	Type          string
	State         string
	Announcements []string
	Role          string
	ByDefault     bool
}

func (s *Service) Integrations(ctx context.Context) ([]Integration, error) {
	rows, err := s.pool.Query(ctx, selectIntegrations+` order by held.name, held.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read blog integrations: %w", err)
	}
	return collectIntegrations(rows)
}

func (s *Service) AddIntegration(
	ctx context.Context,
	actor uuid.UUID,
	in IntegrationEdit,
) (AddedIntegration, error) {
	name, address, host, err := s.checkIntegration(in.Name, in.Address)
	if err != nil {
		return AddedIntegration{}, err
	}
	announced, err := checkAnnouncementTypes(in.Announcements)
	if err != nil {
		return AddedIntegration{}, err
	}
	secret, err := dispatch.MintSecret()
	if err != nil {
		return AddedIntegration{}, err
	}
	sealedAddress, err := s.sealing.Seal([]byte(address))
	if err != nil {
		return AddedIntegration{}, fmt.Errorf("seal the endpoint address: %w", err)
	}
	sealedSecret, err := s.sealing.Seal([]byte(secret))
	if err != nil {
		return AddedIntegration{}, fmt.Errorf("seal the signing secret: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AddedIntegration{}, fmt.Errorf("begin integration: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into blog_integrations
		       (id, type, name, host, address, signing_secret, announcements, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, TypeWebhook, name, host, sealedAddress, sealedSecret, announced, actor)
	if err != nil {
		return AddedIntegration{}, fmt.Errorf("record the integration: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.added", IntegrationID: &id,
	})
	if err != nil {
		return AddedIntegration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AddedIntegration{}, fmt.Errorf("commit integration: %w", err)
	}
	found, err := s.Integration(ctx, id)
	if err != nil {
		return AddedIntegration{}, err
	}
	return AddedIntegration{Integration: found, Secret: secret}, nil
}

func (s *Service) UpdateIntegration(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in IntegrationUpdate,
) (Integration, error) {
	current, err := s.Integration(ctx, id)
	if err != nil {
		return Integration{}, err
	}
	if current.Type == TypeDiscord {
		return Integration{}, ErrNotWebhook
	}
	name := current.Name
	if in.Name != nil {
		name = checkIntegrationName(*in.Name)
	}
	if name == "" {
		return Integration{}, FieldError{Field: "name", Message: "Give the integration a name."}
	}
	announced := current.Announcements
	if in.Announcements != nil {
		if announced, err = checkAnnouncementTypes(in.Announcements); err != nil {
			return Integration{}, err
		}
	}
	moved := in.Address != nil && *in.Address != ""
	var host string
	var sealedAddress []byte
	if moved {
		if host, err = s.sender.Check(*in.Address); err != nil {
			return Integration{}, FieldError{
				Field: "address", Message: capitalize(err.Error()) + ".",
			}
		}
		if sealedAddress, err = s.sealing.Seal([]byte(*in.Address)); err != nil {
			return Integration{}, fmt.Errorf("seal the endpoint address: %w", err)
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin integration update: %w", err)
	}
	defer tx.Rollback(ctx)
	if moved {
		_, err = tx.Exec(ctx, `
			update blog_integrations
			   set name = $2, announcements = $6, host = $3, address = $4, state = $5,
			       verified_at = null, disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, host, sealedAddress, IntegrationUnverified, announced)
	} else {
		_, err = tx.Exec(ctx, `
			update blog_integrations
			   set name = $2, announcements = $3, updated_at = now()
			 where id = $1
		`, id, name, announced)
	}
	if err != nil {
		return Integration{}, fmt.Errorf("update the integration: %w", err)
	}
	if moved {
		if err := s.stopAttemptsTo(ctx, tx, id, SettledMoved); err != nil {
			return Integration{}, err
		}
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.updated", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit integration update: %w", err)
	}
	return s.Integration(ctx, id)
}

func (s *Service) DisableIntegration(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Integration, error) {
	if _, err := s.Integration(ctx, id); err != nil {
		return Integration{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin integration disabling: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update blog_integrations
		   set state = $2, disabled_at = now(), updated_at = now()
		 where id = $1
	`, id, IntegrationDisabled)
	if err != nil {
		return Integration{}, fmt.Errorf("disable the integration: %w", err)
	}
	if err := s.stopAttemptsTo(ctx, tx, id, SettledDisabled); err != nil {
		return Integration{}, err
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.disabled", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit integration disabling: %w", err)
	}
	return s.Integration(ctx, id)
}

func (s *Service) RemoveIntegration(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	if _, err := s.Integration(ctx, id); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin integration removal: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.stopAttemptsTo(ctx, tx, id, SettledRemoved); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		delete from blog_integrations where id = $1
	`, id); err != nil {
		return fmt.Errorf("remove the integration: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.removed", IntegrationID: &id,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit integration removal: %w", err)
	}
	return nil
}

func (s *Service) Integration(ctx context.Context, id uuid.UUID) (Integration, error) {
	rows, err := s.pool.Query(ctx, selectIntegrations+` where held.id = $1`, id)
	if err != nil {
		return Integration{}, fmt.Errorf("read a blog integration: %w", err)
	}
	found, err := collectIntegrations(rows)
	if err != nil {
		return Integration{}, err
	}
	if len(found) == 0 {
		return Integration{}, ErrIntegrationNotFound
	}
	return found[0], nil
}

func (s *Service) checkIntegration(name, address string) (string, string, string, error) {
	checked := checkIntegrationName(name)
	if checked == "" {
		return "", "", "", FieldError{Field: "name", Message: "Give the integration a name."}
	}
	host, err := s.sender.Check(address)
	if err != nil {
		return "", "", "", FieldError{
			Field: "address", Message: capitalize(err.Error()) + ".",
		}
	}
	return checked, address, host, nil
}

func checkAnnouncementTypes(named *[]string) ([]string, error) {
	if named == nil {
		return []string{PostPublished}, nil
	}
	wanted := make(map[string]bool, len(*named))
	for _, one := range *named {
		if !slices.Contains(AnnouncementTypes, one) {
			return nil, FieldError{
				Field:   "announcements",
				Message: "Subscribe to published, updated or unpublished announcements.",
				Cause:   ErrAnnouncementTypeUnknown,
			}
		}
		wanted[one] = true
	}
	announced := make([]string, 0, len(AnnouncementTypes))
	for _, one := range AnnouncementTypes {
		if wanted[one] {
			announced = append(announced, one)
		}
	}
	return announced, nil
}

func checkIntegrationName(name string) string {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) > integrationNameLimit {
		return trimmed[:integrationNameLimit]
	}
	return trimmed
}

func capitalize(sentence string) string {
	if sentence == "" {
		return sentence
	}
	return strings.ToUpper(sentence[:1]) + sentence[1:]
}

func maskAddress(host string) string {
	return dispatch.Scheme + "://" + host + "/…"
}

func scanChannel(guildID, channelID, webhookName, roleID, roleName *string) *Channel {
	if guildID == nil || channelID == nil {
		return nil
	}
	held := &Channel{GuildID: *guildID, ChannelID: *channelID}
	if webhookName != nil {
		held.Webhook = *webhookName
	}
	if roleID != nil && roleName != nil {
		held.RoleID, held.RoleName = *roleID, *roleName
	}
	return held
}

const selectIntegrations = `
	select held.id, held.type, held.name, held.host, held.state, held.announcements,
	       held.guild_id, held.channel_id, held.webhook_name, held.role_id, held.role_name,
	       held.signing_secret_set_at, held.previous_secret_until,
	       held.verified_at, held.disabled_at, held.created_at
	  from blog_integrations held
	`

func collectIntegrations(rows pgx.Rows) ([]Integration, error) {
	defer rows.Close()
	found := make([]Integration, 0, 8)
	for rows.Next() {
		var one Integration
		var guildID, channelID, webhookName, roleID, roleName *string
		err := rows.Scan(
			&one.ID, &one.Type, &one.Name, &one.Host, &one.State, &one.Announcements,
			&guildID, &channelID, &webhookName, &roleID, &roleName,
			&one.SecretSetAt, &one.OldUntil,
			&one.VerifiedAt, &one.DisabledAt, &one.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("read a blog integration: %w", err)
		}
		one.Address = maskAddress(one.Host)
		one.Channel = scanChannel(guildID, channelID, webhookName, roleID, roleName)
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read blog integrations: %w", err)
	}
	return found, nil
}
