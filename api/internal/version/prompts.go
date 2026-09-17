package version

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrUnknownPrompt = errors.New("that prompt is not one of the choices")

type Prompt struct {
	ID   uuid.UUID
	Name string
}

type Mismatch struct {
	Version   asset.Version
	Unmatched []Prompt
	Recorded  []Prompt
}

type PromptCorrespondence struct {
	Current  uuid.UUID
	Recorded *uuid.UUID
}

func (s *Service) ProtectionMismatches(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
) ([]Mismatch, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := ownedAsset(ctx, tx, ownerID, assetID); err != nil {
		return nil, err
	}
	sealed, err := protected.SealedPrompts(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	numbers, err := recordedNumbers(ctx, tx, assetID)
	if err != nil {
		return nil, err
	}
	mismatches := make([]Mismatch, 0)
	for _, number := range numbers {
		version, err := asset.ReadVersion(ctx, tx, assetID, number)
		if err != nil {
			return nil, err
		}
		settled, err := settledPrompts(ctx, tx, version.ID)
		if err != nil {
			return nil, err
		}
		carried := recordedPrompts(version.Blocks)
		held := make(map[uuid.UUID]bool, len(carried))
		for _, prompt := range carried {
			held[prompt.ID] = true
		}
		unmatched := make([]Prompt, 0)
		for id, name := range sealed {
			if !held[id] && !settled[id] {
				unmatched = append(unmatched, Prompt{ID: id, Name: name})
			}
		}
		if len(unmatched) == 0 {
			continue
		}
		slices.SortFunc(unmatched, func(a, b Prompt) int {
			return strings.Compare(a.Name+a.ID.String(), b.Name+b.ID.String())
		})
		mismatches = append(mismatches, Mismatch{
			Version: version.Version, Unmatched: unmatched, Recorded: carried,
		})
	}
	return mismatches, nil
}

func (s *Service) ResolvePromptCorrespondence(
	ctx context.Context,
	ownerID uuid.UUID,
	assetID uuid.UUID,
	number int,
	answers []PromptCorrespondence,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ownedAsset(ctx, tx, ownerID, assetID); err != nil {
		return err
	}
	sealed, err := protected.SealedPrompts(ctx, tx, assetID)
	if err != nil {
		return err
	}
	version, err := asset.ReadVersion(ctx, tx, assetID, number)
	if err != nil {
		return err
	}
	held := make(map[uuid.UUID]bool)
	for _, prompt := range recordedPrompts(version.Blocks) {
		held[prompt.ID] = true
	}
	for _, answer := range answers {
		if _, current := sealed[answer.Current]; !current {
			return ErrUnknownPrompt
		}
		if answer.Recorded != nil && !held[*answer.Recorded] {
			return ErrUnknownPrompt
		}
		if _, err := tx.Exec(ctx, `
			insert into asset_snapshot_prompt_matches
				(snapshot_id, current_fragment_id, recorded_fragment_id)
			values ($1, $2, $3)
			on conflict (snapshot_id, current_fragment_id) do update
			set recorded_fragment_id = excluded.recorded_fragment_id,
				resolved_at = now()
		`, version.ID, answer.Current, answer.Recorded); err != nil {
			return fmt.Errorf("save the prompt correspondence: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func ownedAsset(ctx context.Context, tx pgx.Tx, ownerID, assetID uuid.UUID) error {
	var found bool
	err := tx.QueryRow(ctx, `
		select true from assets where id = $1 and owner_id = $2 and deleted_at is null
	`, assetID, ownerID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return asset.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read the asset to settle: %w", err)
	}
	return nil
}

func recordedNumbers(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) ([]int, error) {
	rows, err := tx.Query(ctx,
		`select number from asset_snapshots where asset_id = $1 order by number desc`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list the recorded versions: %w", err)
	}
	defer rows.Close()
	numbers := make([]int, 0)
	for rows.Next() {
		var number int
		if err := rows.Scan(&number); err != nil {
			return nil, fmt.Errorf("read a recorded version number: %w", err)
		}
		numbers = append(numbers, number)
	}
	return numbers, rows.Err()
}

func settledPrompts(ctx context.Context, tx pgx.Tx, snapshotID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := tx.Query(ctx,
		`select current_fragment_id from asset_snapshot_prompt_matches where snapshot_id = $1`,
		snapshotID)
	if err != nil {
		return nil, fmt.Errorf("read the settled prompts: %w", err)
	}
	defer rows.Close()
	settled := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a settled prompt: %w", err)
		}
		settled[id] = true
	}
	return settled, rows.Err()
}

func recordedPrompts(blocks []block.Block) []Prompt {
	prompts := make([]Prompt, 0)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				prompts = append(prompts, Prompt{
					ID: fragment.ID, Name: protected.PromptName(fragment),
				})
			}
		}
	}
	return prompts
}
