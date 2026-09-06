package publication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// KindWebhook is the one destination kind Illarin sends to so far.
const KindWebhook = "webhook"

// The states a destination passes through. Only an active one receives an event.
const (
	DestinationUnverified = "unverified"
	DestinationActive     = "active"
	DestinationDisabled   = "disabled"
)

const destinationNameLimit = 48

var (
	ErrDestinationNotFound = errors.New("no such publication destination")
	ErrDestinationRefused  = errors.New("the destination is not one this post may send to")
	ErrDestinationInactive = errors.New("the destination has not been verified")
)

// Destination is one endpoint the authority has configured. Nothing outside
// this package ever sees the address it stands for or the secret it signs with.
type Destination struct {
	ID         uuid.UUID
	Kind       string
	Name       string
	Host       string
	Address    string
	State      string
	VerifiedAt *time.Time
	DisabledAt *time.Time
	CreatedAt  time.Time
}

// DestinationEdit is what the authority supplies to add one endpoint.
type DestinationEdit struct {
	Name    string
	Address string
}

// DestinationUpdate carries only the parts of a destination a request named.
type DestinationUpdate struct {
	Name    *string
	Address *string
}

// AddedDestination is a destination and the one showing its secret ever gets.
type AddedDestination struct {
	Destination Destination
	Secret      string
}

// Choice is one safe destination identity a contributor may pick from. It
// carries no address and no secret, which is the whole point of it.
type Choice struct {
	ID        uuid.UUID
	Name      string
	Kind      string
	State     string
	ByDefault bool
}

// Destinations answers every configured destination, masked.
func (s *Service) Destinations(ctx context.Context) ([]Destination, error) {
	rows, err := s.pool.Query(ctx, selectDestinations+` order by held.name, held.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read publication destinations: %w", err)
	}
	return collectDestinations(rows)
}

// AddDestination records one endpoint and hands back the secret it will sign
// with. Illarin keeps a sealed copy and shows the value nowhere else.
func (s *Service) AddDestination(
	ctx context.Context,
	actor uuid.UUID,
	in DestinationEdit,
) (AddedDestination, error) {
	name, address, host, err := s.checkDestination(in.Name, in.Address)
	if err != nil {
		return AddedDestination{}, err
	}
	secret, err := webhook.MintSecret()
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
		       (id, kind, name, host, address, signing_secret, created_by)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, id, KindWebhook, name, host, sealedAddress, sealedSecret, actor)
	if err != nil {
		return AddedDestination{}, fmt.Errorf("record the destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
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

// UpdateDestination renames an endpoint or points it somewhere else. A new
// address takes the destination back to unverified, because control of the old
// one proves nothing about the new one.
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
	name := current.Name
	if in.Name != nil {
		name = checkDestinationName(*in.Name)
	}
	if name == "" {
		return Destination{}, FieldError{Field: "name", Message: "Give the destination a name."}
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
			   set name = $2, host = $3, address = $4, state = $5,
			       verified_at = null, disabled_at = null, updated_at = now()
			 where id = $1
		`, id, name, host, sealedAddress, DestinationUnverified)
	} else {
		_, err = tx.Exec(ctx, `
			update publication_destinations set name = $2, updated_at = now() where id = $1
		`, id, name)
	}
	if err != nil {
		return Destination{}, fmt.Errorf("update the destination: %w", err)
	}
	if moved {
		if err := stopDeliveriesTo(ctx, tx, id, stoppedByMoving); err != nil {
			return Destination{}, err
		}
	}
	err = recordPublicationAudit(ctx, tx, change{
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

// DisableDestination stops an endpoint receiving anything further. What it has
// already received stays where it is.
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
	if err := stopDeliveriesTo(ctx, tx, id, stoppedByDisabling); err != nil {
		return Destination{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
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

// RemoveDestination takes an endpoint out of the configuration. Every delivery
// it already holds keeps the name it was sent under.
func (s *Service) RemoveDestination(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	if _, err := s.Destination(ctx, id); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin destination removal: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := stopDeliveriesTo(ctx, tx, id, stoppedByRemoval); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		delete from publication_destinations where id = $1
	`, id); err != nil {
		return fmt.Errorf("remove the destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
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

// Destination answers one configured endpoint, masked.
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

// checkDestination refuses a name or address before anything is sealed.
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

// maskAddress is the whole of what any reader is told about an endpoint: which
// host it is, and nothing that would let them send to it.
func maskAddress(host string) string {
	return outbound.Scheme + "://" + host + "/…"
}

const selectDestinations = `
	select held.id, held.kind, held.name, held.host, held.state,
	       held.verified_at, held.disabled_at, held.created_at
	  from publication_destinations held
	`

func collectDestinations(rows pgx.Rows) ([]Destination, error) {
	defer rows.Close()
	found := make([]Destination, 0, 8)
	for rows.Next() {
		var one Destination
		err := rows.Scan(
			&one.ID, &one.Kind, &one.Name, &one.Host, &one.State,
			&one.VerifiedAt, &one.DisabledAt, &one.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication destination: %w", err)
		}
		one.Address = maskAddress(one.Host)
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication destinations: %w", err)
	}
	return found, nil
}
