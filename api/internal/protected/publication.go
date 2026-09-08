package protected

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Querier is the read side of a pool or transaction a policy is applied through.
type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// ApplyPublishedPolicy keeps recorded text under the current protection rules.
func ApplyPublishedPolicy(ctx context.Context, q Querier, assetID uuid.UUID, blocks []block.Block) error {
	if err := restorePromptFragments(ctx, q, assetID, blocks, "asset_public.protected_content"); err != nil {
		return err
	}
	_, err := ApplyRecordedPolicy(ctx, q, assetID, nil, blocks)
	return err
}

// ApplyRecordedPolicy holds one recorded version's prompts to the asset's current protection, and says whether it left correspondence unsettled, which hides every prompt in that version.
func ApplyRecordedPolicy(
	ctx context.Context,
	q Querier,
	assetID uuid.UUID,
	snapshotID *uuid.UUID,
	blocks []block.Block,
) (bool, error) {
	current, err := currentPrompts(ctx, q, assetID)
	if err != nil {
		return false, err
	}
	settled, standsFor, err := correspondence(ctx, q, assetID, snapshotID)
	if err != nil {
		return false, err
	}
	recorded := map[uuid.UUID]bool{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		recorded[fragment.ID] = true
	})
	uncertain := false
	for id, prompt := range current {
		if prompt.sealed && !recorded[id] && !settled[id] {
			uncertain = true
		}
	}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		prompt, matched := current[fragment.ID]
		if stands, answered := standsFor[fragment.ID]; answered {
			prompt, matched = current[stands], true
		}
		fragment.Protected = uncertain || prompt.sealed || (!matched && fragment.Protected)
		if fragment.Protected {
			fragment.Text = ""
		}
	})
	return uncertain, nil
}

// RestoreRecordedPrompts puts a recorded version's own sealed text back for its owner.
func RestoreRecordedPrompts(payloads []byte, blocks []block.Block) error {
	var recorded []struct {
		OwnerKind string          `json:"owner_kind"`
		OwnerID   uuid.UUID       `json:"owner_id"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payloads, &recorded); err != nil {
		return fmt.Errorf("read recorded protected prompts: %w", err)
	}
	texts := map[uuid.UUID]string{}
	for _, item := range recorded {
		if item.OwnerKind != promptOwnerKind {
			continue
		}
		var held struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(item.Payload, &held); err != nil {
			return fmt.Errorf("decode a recorded protected prompt: %w", err)
		}
		texts[item.OwnerID] = held.Text
	}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		if text, held := texts[fragment.ID]; held {
			fragment.Text = text
		}
	})
	return nil
}

// UnsealedFragments names the sealed prompts a save would make public.
func UnsealedFragments(
	ctx context.Context,
	q Querier,
	assetID uuid.UUID,
	blocks []block.Block,
) ([]string, error) {
	rows, err := q.Query(ctx, `
		select owner_id from protected_content
		 where asset_id = $1 and owner_kind = $2 and payload_type = $3
	`, assetID, promptOwnerKind, promptPayload)
	if err != nil {
		return nil, fmt.Errorf("read sealed prompts: %w", err)
	}
	defer rows.Close()
	sealed := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a sealed prompt: %w", err)
		}
		sealed[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read sealed prompts: %w", err)
	}
	exposed := []string{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		if sealed[fragment.ID] && !fragment.Protected {
			exposed = append(exposed, PromptName(*fragment))
		}
	})
	return exposed, nil
}

// PromptName is what a person calls one prompt fragment.
func PromptName(fragment block.PromptFragment) string {
	if fragment.Name != "" {
		return fragment.Name
	}
	return "Untitled prompt"
}

// SealedPrompts names the asset's currently sealed prompts by fragment.
func SealedPrompts(ctx context.Context, q Querier, assetID uuid.UUID) (map[uuid.UUID]string, error) {
	current, err := currentPrompts(ctx, q, assetID)
	if err != nil {
		return nil, err
	}
	sealed := map[uuid.UUID]string{}
	for id, prompt := range current {
		if prompt.sealed {
			sealed[id] = prompt.name
		}
	}
	return sealed, nil
}

// promptState is what the working copy says about one prompt fragment now.
type promptState struct {
	sealed bool
	name   string
}

// currentPrompts reads the prompts the working copy holds, sealed or not.
func currentPrompts(ctx context.Context, q Querier, assetID uuid.UUID) (map[uuid.UUID]promptState, error) {
	rows, err := q.Query(ctx,
		`select elements from public.asset_blocks where asset_id = $1`, assetID)
	if err != nil {
		return nil, fmt.Errorf("read current prompt protection: %w", err)
	}
	defer rows.Close()
	current := map[uuid.UUID]promptState{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var elements []block.Element
		if err := json.Unmarshal(data, &elements); err != nil {
			return nil, err
		}
		for _, element := range elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				current[fragment.ID] = promptState{
					sealed: fragment.Protected, name: PromptName(fragment),
				}
			}
		}
	}
	return current, rows.Err()
}

// correspondence reads the owner's answers about which recorded prompt each current sealed prompt is, keyed both ways.
func correspondence(
	ctx context.Context,
	q Querier,
	assetID uuid.UUID,
	snapshotID *uuid.UUID,
) (map[uuid.UUID]bool, map[uuid.UUID]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		select match.current_fragment_id, match.recorded_fragment_id
		  from asset_snapshot_prompt_matches match
		  join public.assets owner on owner.id = $1
		 where match.snapshot_id = coalesce($2, owner.published_snapshot_id)
	`, assetID, snapshotID)
	if err != nil {
		return nil, nil, fmt.Errorf("read recorded prompt correspondence: %w", err)
	}
	defer rows.Close()
	settled := map[uuid.UUID]bool{}
	standsFor := map[uuid.UUID]uuid.UUID{}
	for rows.Next() {
		var current uuid.UUID
		var recorded *uuid.UUID
		if err := rows.Scan(&current, &recorded); err != nil {
			return nil, nil, fmt.Errorf("read a recorded prompt match: %w", err)
		}
		settled[current] = true
		if recorded != nil {
			standsFor[*recorded] = current
		}
	}
	return settled, standsFor, rows.Err()
}

// forEachFragment walks every prompt fragment a page holds, in place.
func forEachFragment(blocks []block.Block, visit func(*block.PromptFragment)) {
	for blockIndex := range blocks {
		for elementIndex := range blocks[blockIndex].Elements {
			element := &blocks[blockIndex].Elements[elementIndex]
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for fragmentIndex := range list.Fragments {
				visit(&list.Fragments[fragmentIndex])
			}
			element.Content = list
		}
	}
}
