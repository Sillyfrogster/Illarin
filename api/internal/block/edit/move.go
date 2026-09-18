package edit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) MoveBlockContent(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	blockID uuid.UUID,
	destinationID uuid.UUID,
	candidate *work.Candidate,
) (SavedBlocks, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SavedBlocks{}, err
	}
	defer tx.Rollback(ctx)

	workType, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return SavedBlocks{}, err
	}
	var after []block.Block
	if err := s.works.ChangeContent(ctx, tx, workID, func() error {
		after, err = s.moveContent(ctx, tx, workType, workID, blockID, destinationID)
		return err
	}); err != nil {
		return SavedBlocks{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return SavedBlocks{}, err
	}
	return SavedBlocks{Type: workType, Blocks: after}, nil
}

func (s *Service) moveContent(
	ctx context.Context,
	tx pgx.Tx,
	workType string,
	workID uuid.UUID,
	blockID uuid.UUID,
	destinationID uuid.UUID,
) ([]block.Block, error) {
	before, err := block.Read(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	if err := private.RestorePromptFragments(ctx, tx, workID, before); err != nil {
		return nil, err
	}
	var source *block.Block
	var destination *block.Block
	for i := range before {
		switch before[i].ID {
		case blockID:
			source = &before[i]
		case destinationID:
			destination = &before[i]
		}
	}
	if source == nil || destination == nil || source.ID == destination.ID {
		return nil, work.ErrNotFound
	}
	if err := moveElements(workType, source, destination); err != nil {
		return nil, err
	}
	after := make([]block.Block, 0, len(before)-1)
	for _, holder := range before {
		if holder.ID == source.ID {
			continue
		}
		holder.Position = len(after)
		after = append(after, holder)
	}
	if err := block.ValidateStructure(*destination); err != nil {
		return nil, invalid(err)
	}
	if err := block.ValidateBuilderConstraints(workType, before, after); err != nil {
		return nil, invalid(err)
	}
	if err := private.SyncPromptFragments(ctx, tx, workID, after, nil); err != nil {
		return nil, invalid(err)
	}
	elements, err := json.Marshal(destination.Elements)
	if err != nil {
		return nil, fmt.Errorf("write moved elements: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		update work_blocks set elements = $3 where id = $1 and work_id = $2
	`, destination.ID, workID, elements); err != nil {
		return nil, fmt.Errorf("save moved elements: %w", err)
	}
	if err := deleteBlockAndClosePositions(ctx, tx, workID, source.ID, after); err != nil {
		return nil, err
	}
	if err := dropUnownedPreservedData(ctx, tx, workID, after); err != nil {
		return nil, err
	}
	return after, s.writeSummary(ctx, tx, workID)
}

func moveElements(workType string, source, destination *block.Block) error {
	definition, _ := source.Definition.Definition(workType)
	if definition.Required {
		return fmt.Errorf("%w: %s is required and cannot be removed", block.ErrInvalid, definition.Title)
	}
	for _, element := range source.Elements {
		if source.Pinned(element.Role, workType) {
			return fmt.Errorf("%w: %s is pinned and cannot move", block.ErrInvalid, element.Role.Label())
		}
	}
	if len(source.Elements) == 0 {
		return fmt.Errorf("%w: this block has no content to move", block.ErrInvalid)
	}
	occupied := make(map[block.Slot]struct{}, len(destination.Elements))
	for _, element := range destination.Elements {
		occupied[element.Slot] = struct{}{}
	}
	free := make([]block.Slot, 0)
	for _, slot := range destination.Layout.Slots() {
		if _, used := occupied[slot]; !used {
			free = append(free, slot)
		}
	}
	if len(free) < len(source.Elements) {
		return fmt.Errorf("%w: %s does not have room for this content", block.ErrInvalid, destination.Definition)
	}
	for i, element := range source.Elements {
		element.Slot = free[i]
		destination.Elements = append(destination.Elements, element)
	}
	return nil
}
