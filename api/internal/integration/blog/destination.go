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
	DestinationUnverified = "unverified"
	DestinationActive     = "active"
	DestinationDisabled   = "disabled"
)

const destinationNameLimit = 48

var PostEvents = []string{EventPublished, EventUpdated, EventWithdrawn}

var ErrEventUnknown = errors.New("no such publication event")

var ErrNotWebhook = errors.New("the destination is a Discord channel")

var (
	ErrDestinationNotFound = errors.New("no such publication destination")
	ErrDestinationRefused  = errors.New("the destination is not one this post may send to")
	ErrDestinationInactive = errors.New("the destination has not been verified")
)

type Destination struct {
	ID          uuid.UUID
	Type        string
	Name        string
	Host        string
	Address     string
	State       string
	Events      []string
	Channel     *Channel
	SecretSetAt time.Time
	OldUntil    *time.Time
	VerifiedAt  *time.Time
	DisabledAt  *time.Time
	CreatedAt   time.Time
}

type Channel struct {
	GuildID   string
	ChannelID string
	Webhook   string
	RoleID    string
	RoleName  string
}

type DestinationEdit struct {
	Name    string
	Address string
	Events  *[]string
}

type DestinationUpdate struct {
	Name    *string
	Address *string
	Events  *[]string
}

type AddedDestination struct {
	Destination Destination
	Secret      string
}

type Choice struct {
	ID        uuid.UUID
	Name      string
	Type      string
	State     string
	Events    []string
	Role      string
	ByDefault bool
}

func (s *Service) Destinations(ctx context.Context) ([]Destination, error) {
	rows, err := s.pool.Query(ctx, selectDestinations+` order by held.name, held.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read publication destinations: %w", err)
	}
	return collectDestinations(rows)
}

func (s *Service) AddDestination(
	ctx context.Context,
	actor uuid.UUID,
	in DestinationEdit,
) (AddedDestination, error) {
	name, address, host, err := s.checkDestination(in.Name, in.Address)
	if err != nil {
		return AddedDestination{}, err
	}
	events, err := checkEvents(in.Events)
	if err != nil {
		return AddedDestination{}, err
	}
	secret, err := dispatch.MintSecret()
	if err != nil {
		return AddedDestination{}, err
	}
	sealedAddress, err := s.sealing.Seal([]byte(address))
	if err != nil {
		return AddedDestination{}, fmt.Errorf("seal the endpoint address: %w", err)
	}
	sealedSecret, err := s.sealing.Seal([]byte(secret))
	if err != nil {
		return AddedDestination{}, fmt.Errorf("seal the signing secret: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AddedDestination{}, fmt.Errorf("begin destination: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into publication_destinations
		       (id, type, name, host, address, signing_secret, events, created_by)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, TypeWebhook, name, host, sealedAddress, sealedSecret, events, actor)
	if err != nil {
		return AddedDestination{}, fmt.Errorf("record the destination: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "destination.added", DestinationID: &id,
	})
	if err != nil {
		return AddedDestination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AddedDestination{}, fmt.Errorf("commit destination: %w", err)
	}
	found, err := s.Destination(ctx, id)
	if err != nil {
		return AddedDestination{}, err
	}
	return AddedDestination{Destination: found, Secret: secret}, nil
}

func (s *Service) UpdateDestination(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in DestinationUpdate,
) (Destination, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if current.Type == TypeDiscord {
		return Destination{}, ErrNotWebhook
	}
	name := current.Name
	if in.Name != nil {
		name = checkDestinationName(*in.Name)
	}
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
	}
	events := current.Events
	if in.Events != nil {
		if events, err = checkEvents(in.Events); err != nil {
			return Destination{}, err
		}
	}
	moved := in.Address != nil && *in.Address != ""
	var host string
	var sealedAddress []byte
	if moved {
		if host, err = s.sender.Check(*in.Address); err != nil {
			return Destination{}, FieldError{
				Field: "address", Message: capitalize(err.Error()) + ".",
			}
		}
		if sealedAddress, err = s.sealing.Seal([]byte(*in.Address)); err != nil {
			return Destination{}, fmt.Errorf("seal the endpoint address: %w", err)
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin destination update: %w", err)
	}
	defer tx.Rollback(ctx)
	if moved {
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, events = $6, host = $3, address = $4, state = $5,
			       verified_at = null, disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, host, sealedAddress, DestinationUnverified, events)
	} else {
		_, err = tx.Exec(ctx, `
			update publication_destinations
			   set name = $2, events = $3, updated_at = now()
			 where id = $1
		`, id, name, events)
	}
	if err != nil {
		return Destination{}, fmt.Errorf("update the destination: %w", err)
	}
	if moved {
		if err := s.stopDeliveriesTo(ctx, tx, id, SettledMoved); err != nil {
			return Destination{}, err
		}
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "destination.updated", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit destination update: %w", err)
	}
	return s.Destination(ctx, id)
}

func (s *Service) DisableDestination(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Destination, error) {
	if _, err := s.Destination(ctx, id); err != nil {
		return Destination{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin destination disabling: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, disabled_at = now(), updated_at = now()
		 where id = $1
	`, id, DestinationDisabled)
	if err != nil {
		return Destination{}, fmt.Errorf("disable the destination: %w", err)
	}
	if err := s.stopDeliveriesTo(ctx, tx, id, SettledDisabled); err != nil {
		return Destination{}, err
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "destination.disabled", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit destination disabling: %w", err)
	}
	return s.Destination(ctx, id)
}

func (s *Service) RemoveDestination(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	if _, err := s.Destination(ctx, id); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin destination removal: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.stopDeliveriesTo(ctx, tx, id, SettledRemoved); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		delete from publication_destinations where id = $1
	`, id); err != nil {
		return fmt.Errorf("remove the destination: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "destination.removed", DestinationID: &id,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit destination removal: %w", err)
	}
	return nil
}

func (s *Service) Destination(ctx context.Context, id uuid.UUID) (Destination, error) {
	rows, err := s.pool.Query(ctx, selectDestinations+` where held.id = $1`, id)
	if err != nil {
		return Destination{}, fmt.Errorf("read a publication destination: %w", err)
	}
	found, err := collectDestinations(rows)
	if err != nil {
		return Destination{}, err
	}
	if len(found) == 0 {
		return Destination{}, ErrDestinationNotFound
	}
	return found[0], nil
}

func (s *Service) checkDestination(name, address string) (string, string, string, error) {
	checked := checkDestinationName(name)
	if checked == "" {
		return "", "", "", FieldError{Field: "name", Message: "Give the destination a name."}
	}
	host, err := s.sender.Check(address)
	if err != nil {
		return "", "", "", FieldError{
			Field: "address", Message: capitalize(err.Error()) + ".",
		}
	}
	return checked, address, host, nil
}

func checkEvents(named *[]string) ([]string, error) {
	if named == nil {
		return []string{EventPublished}, nil
	}
	wanted := make(map[string]bool, len(*named))
	for _, one := range *named {
		if !slices.Contains(PostEvents, one) {
			return nil, FieldError{
				Field:   "events",
				Message: "Subscribe to published, updated or withdrawn events.",
				Cause:   ErrEventUnknown,
			}
		}
		wanted[one] = true
	}
	events := make([]string, 0, len(PostEvents))
	for _, one := range PostEvents {
		if wanted[one] {
			events = append(events, one)
		}
	}
	return events, nil
}

func checkDestinationName(name string) string {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) > destinationNameLimit {
		return trimmed[:destinationNameLimit]
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

const selectDestinations = `
	select held.id, held.type, held.name, held.host, held.state, held.events,
	       held.guild_id, held.channel_id, held.webhook_name, held.role_id, held.role_name,
	       held.signing_secret_set_at, held.previous_secret_until,
	       held.verified_at, held.disabled_at, held.created_at
	  from publication_destinations held
	`

func collectDestinations(rows pgx.Rows) ([]Destination, error) {
	defer rows.Close()
	found := make([]Destination, 0, 8)
	for rows.Next() {
		var one Destination
		var guildID, channelID, webhookName, roleID, roleName *string
		err := rows.Scan(
			&one.ID, &one.Type, &one.Name, &one.Host, &one.State, &one.Events,
			&guildID, &channelID, &webhookName, &roleID, &roleName,
			&one.SecretSetAt, &one.OldUntil,
			&one.VerifiedAt, &one.DisabledAt, &one.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication destination: %w", err)
		}
		one.Address = maskAddress(one.Host)
		one.Channel = scanChannel(guildID, channelID, webhookName, roleID, roleName)
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication destinations: %w", err)
	}
	return found, nil
}
