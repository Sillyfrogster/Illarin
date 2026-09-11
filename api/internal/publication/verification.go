package publication

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
)

const EventVerification = "publication.endpoint.verification.v1"

var ErrNotProven = webhook.ErrNotProven

func (s *Service) VerifyDestination(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Destination, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if current.Kind == KindDiscord {
		return s.provenByDiscord(ctx, actor, id)
	}
	address, secrets, err := s.endpointOf(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if err := webhook.VerifyEndpoint(ctx, s.sender, EventVerification, address, secrets, s.now().UTC()); err != nil {
		return Destination{}, FieldError{
			Field: "address", Message: capitalize(err.Error()), cause: err,
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin destination verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, DestinationActive)
	if err != nil {
		return Destination{}, fmt.Errorf("activate the destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.verified", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit destination verification: %w", err)
	}
	return s.Destination(ctx, id)
}
