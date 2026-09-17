package asset_test

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
)

func init() {
	asset.ErrSummaryRequired = version.ErrSummaryRequired
	asset.ErrNothingToPublish = version.ErrNothingToPublish
	asset.VersionBridge.PublishUpdate = func(ctx context.Context, s *asset.Service, in asset.UpdateRequest, candidate *asset.Candidate) (asset.Update, []asset.ReadinessItem, error) {
		return version.NewService(asset.PoolOf(s), s).PublishUpdate(ctx, version.UpdateRequest(in), candidate)
	}
}
