package protected

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ApplyPublishedPolicy keeps recorded text under the current protection rules.
func ApplyPublishedPolicy(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, assetID uuid.UUID, blocks []block.Block) error {
	if err := restorePromptFragments(ctx, q, assetID, blocks, "asset_public.protected_content"); err != nil {
		return err
	}
	rows, err := q.Query(ctx, `select elements from public.asset_blocks where asset_id = $1`, assetID)
	if err != nil {
		return fmt.Errorf("read current prompt protection: %w", err)
	}
	defer rows.Close()
	current := map[uuid.UUID]bool{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		var elements []block.Element
		if err := json.Unmarshal(data, &elements); err != nil {
			return err
		}
		for _, element := range elements {
			if list, ok := element.Content.(block.PromptList); ok {
				for _, fragment := range list.Fragments {
					current[fragment.ID] = fragment.Protected
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	recorded := map[uuid.UUID]bool{}
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			if list, ok := element.Content.(block.PromptList); ok {
				for _, fragment := range list.Fragments {
					recorded[fragment.ID] = true
				}
			}
		}
	}
	uncertain := false
	for id, sealed := range current {
		if sealed && !recorded[id] {
			uncertain = true
		}
	}
	for i := range blocks {
		for j := range blocks[i].Elements {
			element := &blocks[i].Elements[j]
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for k := range list.Fragments {
				fragment := &list.Fragments[k]
				sealed, matched := current[fragment.ID]
				fragment.Protected = uncertain || sealed || (!matched && fragment.Protected)
				if fragment.Protected {
					fragment.Text = ""
				}
			}
			element.Content = list
		}
	}
	return nil
}
