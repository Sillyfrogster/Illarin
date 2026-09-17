package asset_test

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/google/uuid"
)

func init() {
	bridge := &asset.BlockBridge
	bridge.SaveBlock = func(ctx context.Context, s *asset.Service, ownerID, assetID, blockID uuid.UUID, update asset.BlockUpdate, candidate *asset.Candidate) (asset.SavedBlock, error) {
		saved, err := blocks(s).SaveBlock(ctx, ownerID, assetID, blockID, edit.BlockUpdate(update), candidate)
		return asset.SavedBlock(saved), err
	}
	bridge.AddBlock = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID, definition block.DefinitionID, elementType block.Type, candidate *asset.Candidate) (asset.SavedBlock, error) {
		saved, err := blocks(s).AddBlock(ctx, ownerID, assetID, definition, elementType, candidate)
		return asset.SavedBlock(saved), err
	}
	bridge.ArrangeBlocks = func(ctx context.Context, s *asset.Service, ownerID, assetID uuid.UUID, arrangement []asset.BlockArrangement, candidate *asset.Candidate) (asset.SavedBlocks, error) {
		choices := make([]edit.BlockArrangement, len(arrangement))
		for i, choice := range arrangement {
			choices[i] = edit.BlockArrangement(choice)
		}
		saved, err := blocks(s).ArrangeBlocks(ctx, ownerID, assetID, choices, candidate)
		return asset.SavedBlocks(saved), err
	}
	bridge.RemoveBlock = func(ctx context.Context, s *asset.Service, ownerID, assetID, blockID uuid.UUID, candidate *asset.Candidate) error {
		return blocks(s).RemoveBlock(ctx, ownerID, assetID, blockID, candidate)
	}
	bridge.MoveBlockContent = func(ctx context.Context, s *asset.Service, ownerID, assetID, blockID, destinationID uuid.UUID, candidate *asset.Candidate) (asset.SavedBlocks, error) {
		saved, err := blocks(s).MoveBlockContent(ctx, ownerID, assetID, blockID, destinationID, candidate)
		return asset.SavedBlocks(saved), err
	}
}

func blocks(s *asset.Service) *edit.Service {
	return edit.NewService(asset.PoolOf(s), s)
}
