package blog

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
)

const EventVerification = "blog.endpoint.verification.v1"

var ErrNotProven = dispatch.ErrNotProven

func (s *Service) VerifyIntegration(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Integration, error) {
	current, err := s.Integration(ctx, id)
	if err != nil {
		return Integration{}, err
	}
	if current.Type == TypeDiscord {
		return s.provenByDiscord(ctx, actor, id)
	}
	address, secrets, err := s.endpointOf(ctx, id)
	if err != nil {
		return Integration{}, err
	}
	if err := dispatch.VerifyEndpoint(ctx, s.sender, EventVerification, address, secrets, s.now().UTC()); err != nil {
		return Integration{}, FieldError{
			Field: "address", Message: capitalize(err.Error()), Cause: err,
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Integration{}, fmt.Errorf("begin integration verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update blog_integrations
		   set state = $2, verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, IntegrationActive)
	if err != nil {
		return Integration{}, fmt.Errorf("activate the integration: %w", err)
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "integration.verified", IntegrationID: &id,
	})
	if err != nil {
		return Integration{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Integration{}, fmt.Errorf("commit integration verification: %w", err)
	}
	return s.Integration(ctx, id)
}
