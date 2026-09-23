package page

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

// PreservedData names the namespaces of preserved data a work carries.
func (s *Service) PreservedData(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) ([]work.PreservedData, error) {
	return s.works.PreservedData(ctx, ownerID, workID)
}

// DeletePreservedData drops one namespace of preserved data from a work.
func (s *Service) DeletePreservedData(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	namespace string,
	candidate *work.Candidate,
) error {
	return s.works.DeletePreservedData(ctx, ownerID, workID, namespace, candidate)
}
