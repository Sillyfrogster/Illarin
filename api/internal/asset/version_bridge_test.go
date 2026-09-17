package asset

import (
	"context"

	"github.com/google/uuid"
)

// VersionBridge reaches the version package, which imports this one, and is filled in by version_external_test.go
var VersionBridge struct {
	PublishUpdate func(context.Context, *Service, UpdateRequest, *Candidate) (Update, []ReadinessItem, error)
}

var (
	ErrSummaryRequired  error
	ErrNothingToPublish error
)

type UpdateRequest struct {
	OwnerID      uuid.UUID
	AssetID      uuid.UUID
	Summary      string
	Notes        string
	VersionLabel string
	Announcement UpdateAnnouncement
}

func (s *Service) PublishUpdate(ctx context.Context, in UpdateRequest, candidate *Candidate) (Update, []ReadinessItem, error) {
	return VersionBridge.PublishUpdate(ctx, s, in, candidate)
}
