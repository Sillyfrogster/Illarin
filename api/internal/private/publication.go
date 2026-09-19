package private

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func ApplyPublishedPolicy(ctx context.Context, q Querier, workID uuid.UUID, blocks []block.Block) error {
	if err := restorePromptFragments(ctx, q, workID, blocks, "work_public.private_prompts"); err != nil {
		return err
	}
	_, err := ApplyRecordedPolicy(ctx, q, workID, nil, blocks)
	return err
}

func ApplyRecordedPolicy(
	ctx context.Context,
	q Querier,
	workID uuid.UUID,
	versionID *uuid.UUID,
	blocks []block.Block,
) (bool, error) {
	uncertain, err := markCurrentPrivacy(ctx, q, workID, versionID, blocks)
	if err != nil {
		return false, err
	}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		if fragment.Private {
			fragment.Text = ""
		}
	})
	return uncertain, nil
}

func markCurrentPrivacy(
	ctx context.Context,
	q Querier,
	workID uuid.UUID,
	versionID *uuid.UUID,
	blocks []block.Block,
) (bool, error) {
	current, err := currentPrompts(ctx, q, workID)
	if err != nil {
		return false, err
	}
	settled, standsFor, err := correspondence(ctx, q, workID, versionID)
	if err != nil {
		return false, err
	}
	recorded := map[uuid.UUID]bool{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		recorded[fragment.ID] = true
	})
	uncertain := false
	for id, prompt := range current {
		if prompt.isPrivate && !recorded[id] && !settled[id] {
			uncertain = true
		}
	}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		prompt, matched := current[fragment.ID]
		if stands, answered := standsFor[fragment.ID]; answered {
			prompt, matched = current[stands], true
		}
		fragment.Private = uncertain || prompt.isPrivate || (!matched && fragment.Private)
	})
	return uncertain, nil
}

func RestoreRecordedPrompts(payloads []byte, blocks []block.Block) error {
	var recorded []struct {
		OwnerType string          `json:"owner_type"`
		OwnerID   uuid.UUID       `json:"owner_id"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payloads, &recorded); err != nil {
		return fmt.Errorf("read recorded private prompts: %w", err)
	}
	texts := map[uuid.UUID]string{}
	for _, item := range recorded {
		if item.OwnerType != promptOwnerType {
			continue
		}
		var held struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(item.Payload, &held); err != nil {
			return fmt.Errorf("decode a recorded private prompt: %w", err)
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

// PrepareRestoration applies today's private prompts to historical content without changing allowed apps.
func PrepareRestoration(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	versionID uuid.UUID,
	payloads []byte,
	blocks []block.Block,
) error {
	if err := RestoreRecordedPrompts(payloads, blocks); err != nil {
		return err
	}
	if _, err := markCurrentPrivacy(ctx, tx, workID, &versionID, blocks); err != nil {
		return err
	}
	privateValues := map[uuid.UUID]promptValue{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		if fragment.Private {
			privateValues[fragment.ID] = promptValue{text: fragment.Text}
			fragment.Text = ""
		}
	})
	if len(privateValues) == 0 {
		_, err := tx.Exec(ctx, `delete from private_prompts where work_id = $1`, workID)
		return err
	}
	return replacePromptPayloads(ctx, tx, workID, privateValues)
}

func PromptsMadePublic(
	ctx context.Context,
	q Querier,
	workID uuid.UUID,
	blocks []block.Block,
) ([]string, error) {
	rows, err := q.Query(ctx, `
		select owner_id from private_prompts
		 where work_id = $1 and owner_type = $2 and payload_type = $3
	`, workID, promptOwnerType, promptPayload)
	if err != nil {
		return nil, fmt.Errorf("read private prompts: %w", err)
	}
	defer rows.Close()
	isPrivate := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a private prompt: %w", err)
		}
		isPrivate[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read private prompts: %w", err)
	}
	exposed := []string{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		if isPrivate[fragment.ID] && !fragment.Private {
			exposed = append(exposed, PromptName(*fragment))
		}
	})
	return exposed, nil
}

func PromptName(fragment block.PromptFragment) string {
	if fragment.Name != "" {
		return fragment.Name
	}
	return "Untitled prompt"
}

// ReplacementMakesPublic names the private prompts a replacement would make public.
func ReplacementMakesPublic(
	ctx context.Context,
	q Querier,
	workID uuid.UUID,
	blocks []block.Block,
	identities map[uuid.UUID]string,
	imports []format.PrivatePrompt,
) ([]string, error) {
	current, err := currentPrompts(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	isPrivate := map[uuid.UUID]bool{}
	byIdentity := map[string]uuid.UUID{}
	duplicate := map[string]bool{}
	forEachFragment(blocks, func(fragment *block.PromptFragment) {
		isPrivate[fragment.ID] = fragment.Private
		if identity := identities[fragment.ID]; identity != "" {
			if _, found := byIdentity[identity]; found {
				duplicate[identity] = true
			}
			byIdentity[identity] = fragment.ID
		}
	})
	oldIdentities := map[string]bool{}
	for id := range current {
		if identity := identities[id]; identity != "" {
			if oldIdentities[identity] {
				duplicate[identity] = true
			}
			oldIdentities[identity] = true
		}
	}
	keys := map[string]bool{}
	for _, imported := range imports {
		if isPrivate[imported.FragmentID] && imported.SourceKey != "" {
			keys[imported.SourceKey] = true
		}
	}
	rows, err := q.Query(ctx, `
		select owner_id, coalesce(source_key, '') from private_prompts
		 where work_id = $1 and owner_type = $2 and payload_type = $3
	`, workID, promptOwnerType, promptPayload)
	if err != nil {
		return nil, fmt.Errorf("read the replacement's private prompts: %w", err)
	}
	defer rows.Close()
	exposed := []string{}
	for rows.Next() {
		var id uuid.UUID
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			return nil, err
		}
		identity := identities[id]
		matched := identity != "" && !duplicate[identity] && isPrivate[byIdentity[identity]]
		if isPrivate[id] || matched || keys[key] {
			continue
		}
		name := current[id].name
		if name == "" {
			name = "Untitled prompt"
		}
		exposed = append(exposed, name)
	}
	sort.Strings(exposed)
	return exposed, rows.Err()
}

func PrivatePromptNames(ctx context.Context, q Querier, workID uuid.UUID) (map[uuid.UUID]string, error) {
	current, err := currentPrompts(ctx, q, workID)
	if err != nil {
		return nil, err
	}
	isPrivate := map[uuid.UUID]string{}
	for id, prompt := range current {
		if prompt.isPrivate {
			isPrivate[id] = prompt.name
		}
	}
	return isPrivate, nil
}

type promptState struct {
	isPrivate bool
	name      string
}

func currentPrompts(ctx context.Context, q Querier, workID uuid.UUID) (map[uuid.UUID]promptState, error) {
	rows, err := q.Query(ctx,
		`select elements from public.work_blocks where work_id = $1`, workID)
	if err != nil {
		return nil, fmt.Errorf("read current private prompts: %w", err)
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
					isPrivate: fragment.Private, name: PromptName(fragment),
				}
			}
		}
	}
	return current, rows.Err()
}

func correspondence(
	ctx context.Context,
	q Querier,
	workID uuid.UUID,
	versionID *uuid.UUID,
) (map[uuid.UUID]bool, map[uuid.UUID]uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		select match.current_fragment_id, match.recorded_fragment_id
		  from work_version_prompt_matches match
		  join public.works owner on owner.id = $1
		 where match.version_id = coalesce($2, owner.published_version_id)
	`, workID, versionID)
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

type ExposureRefusal struct {
	Prompts []string
}

func (refusal ExposureRefusal) Error() string {
	return "making a private prompt public needs an explicit confirmation"
}
