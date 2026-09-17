package asset_test

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/google/uuid"
)

func init() {
	asset.DownloadBridge.OpenExport = func(ctx context.Context, s *asset.Service, assetID uuid.UUID, viewerID *uuid.UUID, target string) ([]byte, error) {
		written, err := download.NewService(asset.PoolOf(s), s).OpenExport(ctx, assetID, viewerID, target, nil)
		return written.Body, err
	}
}
