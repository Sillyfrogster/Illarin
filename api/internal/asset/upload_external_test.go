package asset_test

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/google/uuid"
)

func init() {
	bridge := &asset.UploadBridge
	bridge.StartFromNothing = func(ctx context.Context, s *asset.Service, ownerID uuid.UUID, kind, app string) (uuid.UUID, error) {
		return uploads(s).StartFromNothing(ctx, ownerID, kind, app)
	}
	bridge.Create = func(ctx context.Context, s *asset.Service, in asset.CreateInput) (asset.Asset, error) {
		return uploads(s).Create(ctx, upload.CreateInput(in))
	}
	bridge.AcceptIngest = func(ctx context.Context, s *asset.Service, in asset.IngestInput) (asset.IngestOperation, error) {
		return operation(uploads(s).AcceptIngest(ctx, upload.IngestInput(in)))
	}
	bridge.ProcessNextIngest = func(ctx context.Context, s *asset.Service) (bool, error) {
		return uploads(s).ProcessNextIngest(ctx)
	}
	bridge.GetIngest = func(ctx context.Context, s *asset.Service, ownerID, id uuid.UUID) (asset.IngestOperation, error) {
		return operation(uploads(s).GetIngest(ctx, ownerID, id))
	}
	bridge.AcceptRevision = func(ctx context.Context, s *asset.Service, in asset.RevisionInput, candidate *asset.Candidate) (asset.IngestOperation, error) {
		return operation(uploads(s).AcceptRevision(ctx, upload.RevisionInput(in), candidate))
	}
	bridge.AcceptReplacement = func(ctx context.Context, s *asset.Service, ownerID, assetID, operationID uuid.UUID, candidate *asset.Candidate, decisions map[string]string, exposeProtected bool) (asset.IngestOperation, error) {
		return operation(uploads(s).AcceptReplacement(ctx, ownerID, assetID, operationID, candidate, decisions, exposeProtected))
	}
}

func uploads(s *asset.Service) *upload.Service {
	return upload.NewService(asset.PoolOf(s), s)
}

func operation(found upload.Operation, err error) (asset.IngestOperation, error) {
	converted := asset.IngestOperation{ID: found.ID, Status: asset.IngestStatus(found.Status), Asset: found.Asset}
	if found.Failure != nil {
		converted.Failure = &asset.IngestFailure{Reason: found.Failure.Reason, Message: found.Failure.Message}
	}
	return converted, err
}
