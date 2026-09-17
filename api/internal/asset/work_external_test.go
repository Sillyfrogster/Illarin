package asset_test

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func init() {
	bridge := &asset.WorkBridge
	bridge.RecoveryWindow = work.RecoveryWindow
	bridge.SetIdentity = func(ctx context.Context, s *asset.Service, in asset.Identity, candidate *asset.Candidate) error {
		return works(s).SetIdentity(ctx, work.Identity(in), candidate)
	}
	bridge.Publish = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID, candidate *asset.Candidate) ([]asset.ReadinessItem, error) {
		return works(s).Publish(ctx, ownerID, assetID, candidate)
	}
	bridge.SetDiscovery = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID, discovery asset.Discovery) error {
		return works(s).SetDiscovery(ctx, ownerID, assetID, discovery)
	}
	bridge.Delete = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID) error {
		return works(s).Delete(ctx, ownerID, assetID)
	}
	bridge.Restore = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID) error {
		return works(s).Restore(ctx, ownerID, assetID)
	}
	bridge.Page = func(ctx context.Context, s *asset.Service, id uuid.UUID, viewerID *uuid.UUID, visibility asset.ContentVisibility, working bool) (asset.Page, error) {
		read := works(s).Detail
		if working {
			read = works(s).WorkingCopy
		}
		found, err := read(ctx, id, viewerID, visibility)
		return asset.Page{
			WorkingCopyVersion: found.WorkingCopyVersion, Name: found.Name, Blurb: found.Blurb,
			Discovery: found.Discovery, Blocks: found.Blocks, Downloads: found.Downloads,
		}, err
	}
	bridge.Browse = func(ctx context.Context, s *asset.Service, query string, visibility asset.ContentVisibility) (asset.BrowsePage, error) {
		found, err := works(s).Browse(ctx, work.ListFilter{Query: query}, visibility)
		listed := asset.BrowsePage{}
		for _, item := range found.Items {
			listed.Items = append(listed.Items, item.ID)
		}
		return listed, err
	}
}

func works(s *asset.Service) *work.Service {
	return work.NewService(asset.PoolOf(s), s)
}
