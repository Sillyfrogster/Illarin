package page

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

// PreservedNamespaces names the namespaces of preserved data a work carries.
func (s *Service) PreservedNamespaces(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) ([]work.PreservedNamespace, error) {
	return s.works.PreservedNamespaces(ctx, ownerID, workID)
}

// DeletePreservedNamespace drops one namespace of preserved data from a work.
func (s *Service) DeletePreservedNamespace(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	namespace string,
	candidate *work.Candidate,
) error {
	return s.works.DeletePreservedNamespace(ctx, ownerID, workID, namespace, candidate)
}
