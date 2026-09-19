package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type BlockUpdate struct {
	Title           *string
	Layout          block.Layout
	Width           block.Width
	Elements        []block.Element
	AllowedApps     *[]string
	ExposeProtected bool
}

func (s *Service) SaveBlock(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
	blockID uuid.UUID,
	update BlockUpdate,
	candidate *work.Candidate,
) (SavedBlock, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SavedBlock{}, err
	}
	defer tx.Rollback(ctx)

	workType, err := candidate.Lock(ctx, tx, ownerID, workID)
	if err != nil {
		return SavedBlock{}, err
	}
	blocks, index, err := s.writeBlock(ctx, tx, workType, workID, blockID, update)
	if err != nil {
		return SavedBlock{}, err
	}
	if err := private.RestorePromptFragments(ctx, tx, workID, blocks); err != nil {
		return SavedBlock{}, err
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return SavedBlock{}, err
	}
	return SavedBlock{Type: workType, Block: blocks[index]}, nil
}

func (s *Service) writeBlock(
	ctx context.Context,
	tx pgx.Tx,
	workType string,
	workID uuid.UUID,
	blockID uuid.UUID,
	update BlockUpdate,
) ([]block.Block, int, error) {
	blocks, err := block.Read(ctx, tx, workID)
	if err != nil {
		return nil, 0, err
	}
	if err := private.RestorePromptFragments(ctx, tx, workID, blocks); err != nil {
		return nil, 0, err
	}
	before := append([]block.Block(nil), blocks...)
	index := slices.IndexFunc(blocks, func(holder block.Block) bool { return holder.ID == blockID })
	if index < 0 {
		return nil, 0, work.ErrNotFound
	}
	blocks[index].Title = update.Title
	blocks[index].Layout = update.Layout
	blocks[index].Width = update.Width
	blocks[index].Elements = update.Elements
	if err := block.ValidateStructure(blocks[index]); err != nil {
		return nil, 0, invalid(err)
	}
	if err := block.ValidateBuilderConstraints(workType, before, blocks); err != nil {
		return nil, 0, invalid(err)
	}
	if err := s.validateProtectedApps(ctx, tx, workID, workType, blocks, update.AllowedApps); err != nil {
		return nil, 0, invalid(err)
	}
	if !update.ExposeProtected {
		exposed, err := private.UnsealedFragments(ctx, tx, workID, blocks)
		if err != nil {
			return nil, 0, err
		}
		if len(exposed) > 0 {
			return nil, 0, private.ExposureRefusal{Prompts: exposed}
		}
	}
	if err := private.SyncPromptFragments(ctx, tx, workID, blocks, update.AllowedApps); err != nil {
		return nil, 0, invalid(err)
	}
	saved := blocks[index]
	elements, err := json.Marshal(saved.Elements)
	if err != nil {
		return nil, 0, fmt.Errorf("write %s elements: %w", saved.Definition, err)
	}
	result, err := tx.Exec(ctx, `
		update work_blocks
		   set title = $3, layout = $4, width = $5, elements = $6
		 where id = $1 and work_id = $2
	`, blockID, workID, saved.Title, saved.Layout, saved.Width, elements)
	if err != nil {
		return nil, 0, fmt.Errorf("save block: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, 0, work.ErrNotFound
	}
	if err := dropUnownedPreservedData(ctx, tx, workID, blocks); err != nil {
		return nil, 0, err
	}
	return blocks, index, s.writeSummary(ctx, tx, workID)
}

func (s *Service) validateProtectedApps(
	ctx context.Context,
	q db.DBTX,
	workID uuid.UUID,
	workType string,
	blocks []block.Block,
	allowedApps *[]string,
) error {
	if allowedApps == nil || !private.HasPromptFragments(blocks) {
		return nil
	}
	var origin string
	if err := q.QueryRow(ctx, `select coalesce(origin_format, '') from works where id = $1`, workID).Scan(&origin); err != nil {
		return fmt.Errorf("read the work origin for protected delivery: %w", err)
	}
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		elements = append(elements, holder.Elements...)
	}
	offered := s.reg.OfferedTargets(format.CapabilitySubject{
		Type: workType, Origin: origin, Elements: elements,
	})
	for _, app := range *allowedApps {
		available := slices.ContainsFunc(private.AppTargets(workType, app), func(wanted string) bool {
			return slices.ContainsFunc(offered, func(target format.Target) bool { return target.Format == wanted })
		})
		if !available {
			return fmt.Errorf("%q has no usable export target for this work", app)
		}
	}
	return nil
}
