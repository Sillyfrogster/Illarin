package asset

import (
	"context"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkBridge reaches the work package, which imports this one, and is filled in by work_external_test.go
var WorkBridge struct {
	SetIdentity    func(context.Context, *Service, Identity, *Candidate) error
	Publish        func(context.Context, *Service, uuid.UUID, uuid.UUID, *Candidate) ([]ReadinessItem, error)
	SetDiscovery   func(context.Context, *Service, uuid.UUID, uuid.UUID, Discovery) error
	Delete         func(context.Context, *Service, uuid.UUID, uuid.UUID) error
	Restore        func(context.Context, *Service, uuid.UUID, uuid.UUID) error
	Page           func(context.Context, *Service, uuid.UUID, *uuid.UUID, ContentVisibility, bool) (Page, error)
	Browse         func(context.Context, *Service, string, ContentVisibility) (BrowsePage, error)
	RecoveryWindow time.Duration
}

type Identity struct {
	OwnerID uuid.UUID
	AssetID uuid.UUID
	Name    string
	Blurb   string
	IsNSFW  *bool
}

type Page struct {
	WorkingCopyVersion *int64
	Name               string
	Blurb              string
	Discovery          Discovery
	Blocks             []block.Block
	Downloads          []format.Target
}

type ListFilter struct {
	Query string
}

type BrowsePage struct {
	Items []uuid.UUID
}

func (s *Service) SetIdentity(ctx context.Context, in Identity, candidate *Candidate) error {
	return WorkBridge.SetIdentity(ctx, s, in, candidate)
}

func (s *Service) Publish(ctx context.Context, ownerID, assetID uuid.UUID, candidate *Candidate) ([]ReadinessItem, error) {
	return WorkBridge.Publish(ctx, s, ownerID, assetID, candidate)
}

func (s *Service) SetDiscovery(ctx context.Context, ownerID, assetID uuid.UUID, discovery Discovery) error {
	return WorkBridge.SetDiscovery(ctx, s, ownerID, assetID, discovery)
}

func (s *Service) Delete(ctx context.Context, ownerID, assetID uuid.UUID) error {
	return WorkBridge.Delete(ctx, s, ownerID, assetID)
}

func (s *Service) Restore(ctx context.Context, ownerID, assetID uuid.UUID) error {
	return WorkBridge.Restore(ctx, s, ownerID, assetID)
}

func (s *Service) Detail(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility ContentVisibility) (Page, error) {
	return WorkBridge.Page(ctx, s, id, viewerID, visibility, false)
}

func (s *Service) WorkingCopy(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility ContentVisibility) (Page, error) {
	return WorkBridge.Page(ctx, s, id, viewerID, visibility, true)
}

func (s *Service) Browse(ctx context.Context, filter ListFilter, visibility ContentVisibility) (BrowsePage, error) {
	return WorkBridge.Browse(ctx, s, filter.Query, visibility)
}

func PoolOf(s *Service) *pgxpool.Pool {
	return s.pool
}
