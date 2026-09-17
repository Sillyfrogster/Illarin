package asset

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

// BlockBridge reaches the block editing package, which imports this one, and is filled in by block_external_test.go
var BlockBridge struct {
	SaveBlock        func(context.Context, *Service, uuid.UUID, uuid.UUID, uuid.UUID, BlockUpdate, *Candidate) (SavedBlock, error)
	AddBlock         func(context.Context, *Service, uuid.UUID, uuid.UUID, block.DefinitionID, block.Type, *Candidate) (SavedBlock, error)
	ArrangeBlocks    func(context.Context, *Service, uuid.UUID, uuid.UUID, []BlockArrangement, *Candidate) (SavedBlocks, error)
	RemoveBlock      func(context.Context, *Service, uuid.UUID, uuid.UUID, uuid.UUID, *Candidate) error
	MoveBlockContent func(context.Context, *Service, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *Candidate) (SavedBlocks, error)
}

type BlockUpdate struct {
	Title           *string
	Layout          block.Layout
	Width           block.Width
	Elements        []block.Element
	AllowedApps     *[]string
	ExposeProtected bool
}

type SavedBlock struct {
	Kind  string
	Block block.Block
}

type BlockArrangement struct {
	ID     uuid.UUID
	Hidden bool
	Width  block.Width
}

func (s *Service) SaveBlock(ctx context.Context, ownerID, assetID, blockID uuid.UUID, update BlockUpdate, candidate *Candidate) (SavedBlock, error) {
	return BlockBridge.SaveBlock(ctx, s, ownerID, assetID, blockID, update, candidate)
}

func (s *Service) AddBlock(ctx context.Context, ownerID, assetID uuid.UUID, definition block.DefinitionID, elementType block.Type, candidate *Candidate) (SavedBlock, error) {
	return BlockBridge.AddBlock(ctx, s, ownerID, assetID, definition, elementType, candidate)
}

func (s *Service) ArrangeBlocks(ctx context.Context, ownerID, assetID uuid.UUID, arrangement []BlockArrangement, candidate *Candidate) (SavedBlocks, error) {
	return BlockBridge.ArrangeBlocks(ctx, s, ownerID, assetID, arrangement, candidate)
}

func (s *Service) RemoveBlock(ctx context.Context, ownerID, assetID, blockID uuid.UUID, candidate *Candidate) error {
	return BlockBridge.RemoveBlock(ctx, s, ownerID, assetID, blockID, candidate)
}

func (s *Service) MoveBlockContent(ctx context.Context, ownerID, assetID, blockID, destinationID uuid.UUID, candidate *Candidate) (SavedBlocks, error) {
	return BlockBridge.MoveBlockContent(ctx, s, ownerID, assetID, blockID, destinationID, candidate)
}
