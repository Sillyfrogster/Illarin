package version

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrUnknownPrompt = errors.New("that prompt is not one of the choices")

type Prompt struct {
	ID   uuid.UUID
	Name string
}

type Mismatch struct {
	Version   work.Version
	Unmatched []Prompt
	Recorded  []Prompt
}

type PromptCorrespondence struct {
	Current  uuid.UUID
	Recorded *uuid.UUID
}

func (s *Service) PrivatePromptMismatches(
	ctx context.Context,
	ownerID uuid.UUID,
	workID uuid.UUID,
) ([]Mismatch, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := ownedWork(ctx, tx, ownerID, workID); err != nil {
		return nil, err
	}
	privateNow, err := private.PrivatePromptNames(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	numbers, err := recordedNumbers(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	mismatches := make([]Mismatch, 0)
	for _, number := range numbers {
		version, err := work.ReadVersion(ctx, tx, workID, number)
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
		for id, name := range privateNow {
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
	workID uuid.UUID,
	number int,
	answers []PromptCorrespondence,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ownedWork(ctx, tx, ownerID, workID); err != nil {
		return err
	}
	privateNow, err := private.PrivatePromptNames(ctx, tx, workID)
	if err != nil {
		return err
	}
	version, err := work.ReadVersion(ctx, tx, workID, number)
	if err != nil {
		return err
	}
	held := make(map[uuid.UUID]bool)
	for _, prompt := range recordedPrompts(version.Blocks) {
		held[prompt.ID] = true
	}
	for _, answer := range answers {
		if _, current := privateNow[answer.Current]; !current {
			return ErrUnknownPrompt
		}
		if answer.Recorded != nil && !held[*answer.Recorded] {
			return ErrUnknownPrompt
		}
		if _, err := tx.Exec(ctx, `
			insert into work_version_prompt_matches
				(version_id, current_fragment_id, recorded_fragment_id)
			values ($1, $2, $3)
			on conflict (version_id, current_fragment_id) do update
			set recorded_fragment_id = excluded.recorded_fragment_id,
				resolved_at = now()
		`, version.ID, answer.Current, answer.Recorded); err != nil {
			return fmt.Errorf("save the prompt correspondence: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func ownedWork(ctx context.Context, tx pgx.Tx, ownerID, workID uuid.UUID) error {
	var found bool
	err := tx.QueryRow(ctx, `
		select true from works where id = $1 and owner_id = $2 and deleted_at is null
	`, workID, ownerID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return work.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("read the work to settle: %w", err)
	}
	return nil
}

func recordedNumbers(ctx context.Context, tx pgx.Tx, workID uuid.UUID) ([]int, error) {
	rows, err := tx.Query(ctx,
		`select number from work_versions where work_id = $1 order by number desc`, workID)
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

func settledPrompts(ctx context.Context, tx pgx.Tx, versionID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := tx.Query(ctx,
		`select current_fragment_id from work_version_prompt_matches where version_id = $1`,
		versionID)
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
					ID: fragment.ID, Name: private.PromptName(fragment),
				})
			}
		}
	}
	return prompts
}
