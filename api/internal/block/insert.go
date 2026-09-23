package block

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrInvalid = errors.New("invalid block")

// Insert validates each block and writes it as a new row on a work's drafted page
func Insert(ctx context.Context, q db.DBTX, workID uuid.UUID, blocks []Block) error {
	queries := db.New(q)
	for _, b := range blocks {
		if err := ValidateStructure(b); err != nil {
			return fmt.Errorf("validate %s block: %w: %v", b.Definition, ErrInvalid, err)
		}
		elements, err := json.Marshal(b.Elements)
		if err != nil {
			return fmt.Errorf("write %s elements: %w", b.Definition, err)
		}
		var title pgtype.Text
		if b.Title != nil {
			title = pgtype.Text{String: *b.Title, Valid: true}
		}
		params := db.InsertWorkBlockParams{
			ID:         pgtype.UUID{Bytes: b.ID, Valid: true},
			WorkID:     pgtype.UUID{Bytes: workID, Valid: true},
			Definition: string(b.Definition),
			Title:      title,
			Position:   int32(b.Position),
			Hidden:     b.Hidden,
			Layout:     string(b.Layout),
			Width:      string(b.Width),
			Elements:   elements,
		}
		if err := queries.InsertWorkBlock(ctx, params); err != nil {
			return fmt.Errorf("insert %s block: %w", b.Definition, err)
		}
	}
	return nil
}
