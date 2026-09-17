package asset

import (
	"context"

	"github.com/google/uuid"
)

// DownloadBridge reaches the download package, which imports this one, and is filled in by download_external_test.go
var DownloadBridge struct {
	OpenExport func(context.Context, *Service, uuid.UUID, *uuid.UUID, string) ([]byte, error)
}

type Export struct {
	Body []byte
}

func (s *Service) OpenExport(ctx context.Context, assetID uuid.UUID, viewerID *uuid.UUID, target string, _ any) (Export, error) {
	body, err := DownloadBridge.OpenExport(ctx, s, assetID, viewerID, target)
	return Export{Body: body}, err
}
