package edit

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/google/uuid"
)

type BlockArrangement struct {
	ID     uuid.UUID
	Hidden bool
	Width  block.Width
}

func (s *Service) AddBlock(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	definition block.DefinitionID,
	elementType block.Type,
	candidate *asset.Candidate,
) (SavedBlock, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SavedBlock{}, err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return SavedBlock{}, err
	}
	var added block.Block
	if err := s.assets.ChangeContent(ctx, tx, workID, func() error {
		page, err := block.Read(ctx, tx, workID)
		if err != nil {
			return err
		}
		added, err = block.NewBlock(kind, definition, elementType, page)
		if err != nil {
			return invalid(err)
		}
		if err := block.ValidateStructure(added); err != nil {
			return invalid(err)
		}
		after := append(page, added)
		if err := block.ValidateBuilderConstraints(kind, after, after); err != nil {
			return invalid(err)
		}
		if err := block.Insert(ctx, tx, workID, []block.Block{added}); err != nil {
			return err
		}
		return s.writeSummary(ctx, tx, workID)
	}); err != nil {
		return SavedBlock{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return SavedBlock{}, err
	}
	return SavedBlock{Kind: kind, Block: added}, nil
}

func (s *Service) ArrangeBlocks(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	arrangement []BlockArrangement,
	candidate *asset.Candidate,
) (SavedBlocks, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SavedBlocks{}, err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return SavedBlocks{}, err
	}
	blocks, err := block.Read(ctx, tx, workID)
	if err != nil {
		return SavedBlocks{}, err
	}
	after, err := arranged(kind, blocks, arrangement)
	if err != nil {
		return SavedBlocks{}, err
	}
	for _, holder := range after {
		if _, err := tx.Exec(ctx, `
			update asset_blocks
			   set position = $3, hidden = $4, width = $5
			 where id = $1 and asset_id = $2
		`, holder.ID, workID, holder.Position, holder.Hidden, holder.Width); err != nil {
			return SavedBlocks{}, fmt.Errorf("save block arrangement: %w", err)
		}
	}
	if err := missingWork(summary.WriteFilters(ctx, tx, workID)); err != nil {
		return SavedBlocks{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return SavedBlocks{}, err
	}
	return SavedBlocks{Kind: kind, Blocks: after}, nil
}

func arranged(kind string, blocks []block.Block, arrangement []BlockArrangement) ([]block.Block, error) {
	if len(arrangement) != len(blocks) {
		return nil, fmt.Errorf("%w: include every block once before saving the arrangement", block.ErrInvalid)
	}
	byID := make(map[uuid.UUID]block.Block, len(blocks))
	for _, holder := range blocks {
		byID[holder.ID] = holder
	}
	after := make([]block.Block, len(arrangement))
	seen := make(map[uuid.UUID]struct{}, len(arrangement))
	for position, choice := range arrangement {
		holder, ok := byID[choice.ID]
		if !ok {
			return nil, fmt.Errorf("%w: the arrangement includes a block that is not on this page", block.ErrInvalid)
		}
		if _, duplicate := seen[choice.ID]; duplicate {
			return nil, fmt.Errorf("%w: include each block once before saving the arrangement", block.ErrInvalid)
		}
		seen[choice.ID] = struct{}{}
		definition, _ := holder.Definition.Definition(kind)
		if choice.Hidden && definition.Required && !definition.Hideable {
			return nil, fmt.Errorf("%w: %s is always shown and cannot be hidden", block.ErrInvalid, definition.Title)
		}
		holder.Position = position
		holder.Hidden = choice.Hidden
		holder.Width = choice.Width
		after[position] = holder
	}
	if err := block.ValidateBuilderConstraints(kind, blocks, after); err != nil {
		return nil, invalid(err)
	}
	return after, nil
}

func (s *Service) RemoveBlock(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	blockID uuid.UUID,
	candidate *asset.Candidate,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return err
	}
	if err := s.assets.ChangeContent(ctx, tx, workID, func() error {
		blocks, err := block.Read(ctx, tx, workID)
		if err != nil {
			return err
		}
		if err := protected.RestorePromptFragments(ctx, tx, workID, blocks); err != nil {
			return err
		}
		remaining, err := withoutBlock(kind, blocks, blockID)
		if err != nil {
			return err
		}
		if err := protected.SyncPromptFragments(ctx, tx, workID, remaining, nil); err != nil {
			return invalid(err)
		}
		if err := deleteBlockAndClosePositions(ctx, tx, workID, blockID, remaining); err != nil {
			return err
		}
		if err := dropUnownedPreservedData(ctx, tx, workID, remaining); err != nil {
			return err
		}
		return s.writeSummary(ctx, tx, workID)
	}); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, workID)
}

func withoutBlock(kind string, blocks []block.Block, blockID uuid.UUID) ([]block.Block, error) {
	remaining := make([]block.Block, 0, len(blocks))
	found := false
	for _, holder := range blocks {
		if holder.ID != blockID {
			remaining = append(remaining, holder)
			continue
		}
		found = true
		definition, _ := holder.Definition.Definition(kind)
		if definition.Required {
			return nil, fmt.Errorf("%w: %s is required and cannot be removed", block.ErrInvalid, definition.Title)
		}
	}
	if !found {
		return nil, asset.ErrNotFound
	}
	if err := block.ValidateBuilderConstraints(kind, blocks, remaining); err != nil {
		return nil, invalid(err)
	}
	return remaining, nil
}
