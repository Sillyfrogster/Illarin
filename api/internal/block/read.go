package block

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Read returns a work's drafted blocks in page order
func Read(ctx context.Context, q db.DBTX, workID uuid.UUID) ([]Block, error) {
	rows, err := db.New(q).WorkBlocks(ctx, pgtype.UUID{Bytes: workID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("read work blocks: %w", err)
	}
	blocks := make([]Block, 0, len(rows))
	for _, row := range rows {
		var elements []Element
		if err := json.Unmarshal(row.Elements, &elements); err != nil {
			return nil, fmt.Errorf("read %s elements: %w", row.Definition, err)
		}
		var title *string
		if row.Title.Valid {
			title = &row.Title.String
		}
		blocks = append(blocks, Block{
			ID:         row.ID.Bytes,
			Definition: DefinitionID(row.Definition),
			Title:      title,
			Position:   int(row.Position),
			Hidden:     row.Hidden,
			Layout:     Layout(row.Layout),
			Width:      Width(row.Width),
			Elements:   elements,
		})
	}
	return blocks, nil
}
