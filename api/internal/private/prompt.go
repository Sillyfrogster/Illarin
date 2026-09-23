package private

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	promptOwnerType = "prompt_fragment"
	promptPayload   = "prompt_fragment_text"
)

var ErrPolicyRequired = errors.New("choose at least one allowed app before making a prompt private")

func ImportPromptFragments(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	blocks []block.Block,
	carried map[uuid.UUID]string,
	imports []format.PrivatePrompt,
	initialApps []string,
) error {
	owners := make(map[uuid.UUID]bool, len(imports))
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				if fragment.Private && fragment.Text == "" {
					owners[fragment.ID] = true
				}
			}
		}
	}
	if len(owners) != len(imports) {
		return format.MalformedInput(errors.New(
			"private prompt metadata does not match the imported prompt fragments",
		))
	}

	apps, err := policy(ctx, tx, workID, nil)
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		apps, err = policy(ctx, tx, workID, &initialApps)
		if err != nil {
			return err
		}
		if len(apps) == 0 {
			return ErrPolicyRequired
		}
		for _, app := range apps {
			if _, err := tx.Exec(ctx, `
				insert into private_prompt_apps (work_id, app) values ($1, $2)
			`, workID, app); err != nil {
				return fmt.Errorf("save imported allowed apps: %w", err)
			}
		}
	}

	values := make(map[uuid.UUID]promptValue, len(imports))
	for _, imported := range imports {
		if !owners[imported.FragmentID] {
			return format.MalformedInput(errors.New(
				"private prompt metadata does not identify an imported prompt fragment",
			))
		}
		text := imported.Text
		if imported.ReuseExisting {
			held, err := heldPromptText(ctx, tx, workID, carried, imported)
			if err != nil {
				return err
			}
			text = held
		}
		values[imported.FragmentID] = promptValue{
			text: text, sourceKey: imported.SourceKey, replaceSourceKey: true,
		}
	}
	return replacePromptPayloads(ctx, tx, workID, values)
}

// UnfillablePrompts names the private fragments this work holds no wording for.
func UnfillablePrompts(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	carried map[uuid.UUID]string,
	imports []format.PrivatePrompt,
) ([]uuid.UUID, error) {
	missing := make([]uuid.UUID, 0)
	for _, imported := range imports {
		if !imported.ReuseExisting {
			continue
		}
		if _, err := heldPromptText(ctx, tx, workID, carried, imported); err != nil {
			if _, classified := format.FailureOf(err); !classified {
				return nil, err
			}
			missing = append(missing, imported.FragmentID)
		}
	}
	return missing, nil
}

// heldPromptText finds the text a placeholder stands in for, private or still public.
func heldPromptText(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	carried map[uuid.UUID]string,
	imported format.PrivatePrompt,
) (string, error) {
	if imported.SourceKey == "" {
		return "", format.MalformedInput(errors.New(
			"a reusable private prompt needs a source key",
		))
	}
	text, held, err := promptTextBySourceKey(ctx, tx, workID, imported.SourceKey)
	if err != nil {
		return "", err
	}
	if held {
		return text, nil
	}
	if public, found := carried[imported.FragmentID]; found {
		return public, nil
	}
	return "", format.MalformedInput(fmt.Errorf(
		"this file leaves out the text of private prompt %q, and nothing here holds it",
		imported.SourceKey,
	))
}

func promptTextBySourceKey(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	sourceKey string,
) (string, bool, error) {
	rows, err := tx.Query(ctx, `
		select payload
		  from private_prompts
		 where work_id = $1 and owner_type = $2 and payload_type = $3 and source_key = $4
		 for update
	`, workID, promptOwnerType, promptPayload, sourceKey)
	if err != nil {
		return "", false, fmt.Errorf("read existing private prompt: %w", err)
	}
	defer rows.Close()
	var texts []string
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return "", false, fmt.Errorf("read existing private prompt: %w", err)
		}
		var item struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return "", false, fmt.Errorf("decode existing private prompt: %w", err)
		}
		texts = append(texts, item.Text)
	}
	if err := rows.Err(); err != nil {
		return "", false, fmt.Errorf("read existing private prompt: %w", err)
	}
	if len(texts) > 1 {
		return "", false, format.MalformedInput(fmt.Errorf(
			"private prompt key %q stands for more than one saved value", sourceKey,
		))
	}
	if len(texts) == 0 {
		return "", false, nil
	}
	return texts[0], true, nil
}

// AllowsFormat says whether any of the apps keeps private prompts private in the format
func AllowsFormat(reg *format.Registry, apps []string, formatID string) bool {
	return slices.ContainsFunc(apps, func(app string) bool {
		return slices.Contains(reg.PrivatePromptFormats(app), formatID)
	})
}

// EligibleApps lists the apps that keep private prompts private in one of the offered formats
func EligibleApps(reg *format.Registry, offered []string) []string {
	apps := []string{}
	for _, app := range format.Apps() {
		if slices.ContainsFunc(offered, func(formatID string) bool {
			return AllowsFormat(reg, []string{app.ID}, formatID)
		}) {
			apps = append(apps, app.ID)
		}
	}
	return apps
}

func HasPromptFragments(blocks []block.Block) bool {
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				if fragment.Private {
					return true
				}
			}
		}
	}
	return false
}

func SyncPromptFragments(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	blocks []block.Block,
	allowedApps *[]string,
) error {
	privateValues := make(map[uuid.UUID]promptValue)
	for blockIndex := range blocks {
		for elementIndex := range blocks[blockIndex].Elements {
			element := &blocks[blockIndex].Elements[elementIndex]
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for itemIndex := range list.Fragments {
				fragment := &list.Fragments[itemIndex]
				if !fragment.Private {
					continue
				}
				if fragment.Marker != "" {
					return fmt.Errorf("a prompt marker cannot be private")
				}
				privateValues[fragment.ID] = promptValue{text: fragment.Text}
				fragment.Text = ""
			}
			element.Content = list
		}
	}

	if len(privateValues) == 0 {
		if _, err := tx.Exec(ctx, `delete from private_prompts where work_id = $1`, workID); err != nil {
			return fmt.Errorf("remove private prompts: %w", err)
		}
		if _, err := tx.Exec(ctx, `delete from private_prompt_apps where work_id = $1`, workID); err != nil {
			return fmt.Errorf("remove allowed apps: %w", err)
		}
		return nil
	}

	apps, err := policy(ctx, tx, workID, allowedApps)
	if err != nil {
		return err
	}
	if len(apps) == 0 {
		return ErrPolicyRequired
	}
	if allowedApps != nil {
		if _, err := tx.Exec(ctx, `delete from private_prompt_apps where work_id = $1`, workID); err != nil {
			return fmt.Errorf("replace allowed apps: %w", err)
		}
		for _, app := range apps {
			if _, err := tx.Exec(ctx, `insert into private_prompt_apps (work_id, app) values ($1, $2)`, workID, app); err != nil {
				return fmt.Errorf("save allowed apps: %w", err)
			}
		}
	}

	return replacePromptPayloads(ctx, tx, workID, privateValues)
}

type promptValue struct {
	text             string
	sourceKey        string
	replaceSourceKey bool
}

func replacePromptPayloads(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	values map[uuid.UUID]promptValue,
) error {
	for id, value := range values {
		payload, err := json.Marshal(map[string]string{"text": value.text})
		if err != nil {
			return fmt.Errorf("encode private prompt: %w", err)
		}
		digest := sha256.Sum256([]byte(value.text))
		var sourceKey any
		if value.sourceKey != "" {
			sourceKey = value.sourceKey
		}
		if _, err := tx.Exec(ctx, `
			insert into private_prompts (
				work_id, owner_type, owner_id, payload_type, payload, source_key, digest
			) values ($1, $2, $3, $4, $5, $6, $7)
			on conflict (work_id, owner_type, owner_id) do update
			set payload = excluded.payload,
				payload_type = excluded.payload_type,
				source_key = case when $8 then excluded.source_key else private_prompts.source_key end,
				digest = excluded.digest
		`, workID, promptOwnerType, id, promptPayload, payload, sourceKey, digest[:],
			value.replaceSourceKey); err != nil {
			return fmt.Errorf("save private prompt: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		delete from private_prompts
		 where work_id = $1 and owner_type = $2 and not (owner_id = any($3::uuid[]))
	`, workID, promptOwnerType, promptIDs(values)); err != nil {
		return fmt.Errorf("remove prompts made public: %w", err)
	}
	return nil
}

func policy(ctx context.Context, tx pgx.Tx, workID uuid.UUID, supplied *[]string) ([]string, error) {
	if supplied != nil {
		seen := map[string]bool{}
		apps := make([]string, 0, len(*supplied))
		for _, app := range *supplied {
			if seen[app] {
				return nil, fmt.Errorf("%q is listed twice", app)
			}
			seen[app] = true
			apps = append(apps, app)
		}
		return apps, nil
	}
	rows, err := tx.Query(ctx, `select app from private_prompt_apps where work_id = $1 order by app`, workID)
	if err != nil {
		return nil, fmt.Errorf("read allowed apps: %w", err)
	}
	defer rows.Close()
	var apps []string
	for rows.Next() {
		var app string
		if err := rows.Scan(&app); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func RestorePromptFragments(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, workID uuid.UUID, blocks []block.Block) error {
	return restorePromptFragments(ctx, q, workID, blocks, "private_prompts")
}

func restorePromptFragments(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, workID uuid.UUID, blocks []block.Block, table string) error {
	rows, err := q.Query(ctx, `
		select owner_id, payload from `+table+`
		 where work_id = $1 and owner_type = $2 and payload_type = $3
	`, workID, promptOwnerType, promptPayload)
	if err != nil {
		return fmt.Errorf("read private prompts: %w", err)
	}
	defer rows.Close()
	texts := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var payload []byte
		if err := rows.Scan(&id, &payload); err != nil {
			return fmt.Errorf("read private prompt: %w", err)
		}
		var item struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(payload, &item); err != nil {
			return fmt.Errorf("decode private prompt: %w", err)
		}
		texts[id] = item.Text
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read private prompts: %w", err)
	}
	for blockIndex := range blocks {
		for elementIndex := range blocks[blockIndex].Elements {
			element := &blocks[blockIndex].Elements[elementIndex]
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for itemIndex := range list.Fragments {
				fragment := &list.Fragments[itemIndex]
				if !fragment.Private {
					continue
				}
				text, found := texts[fragment.ID]
				if !found {
					return fmt.Errorf("private prompt %s has no payload", fragment.ID)
				}
				fragment.Text = text
			}
			element.Content = list
		}
	}
	return nil
}

func Apps(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, workID uuid.UUID) ([]string, error) {
	rows, err := q.Query(ctx, `select app from private_prompt_apps where work_id = $1 order by app`, workID)
	if err != nil {
		return nil, fmt.Errorf("read allowed apps: %w", err)
	}
	defer rows.Close()
	apps := []string{}
	for rows.Next() {
		var app string
		if err := rows.Scan(&app); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func promptIDs(values map[uuid.UUID]promptValue) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	for id := range values {
		result = append(result, id)
	}
	return result
}
