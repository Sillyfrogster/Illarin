package work

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/google/uuid"
)

// PreservedNamespaces names the namespaces of preserved data a work carries.
func (s *Service) PreservedNamespaces(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) ([]asset.PreservedNamespace, error) {
	return s.assets.PreservedNamespaces(ctx, ownerID, workID)
}

// DeletePreservedNamespace drops one namespace of preserved data from a work.
func (s *Service) DeletePreservedNamespace(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	namespace string,
	candidate *asset.Candidate,
) error {
	return s.assets.DeletePreservedNamespace(ctx, ownerID, workID, namespace, candidate)
}
