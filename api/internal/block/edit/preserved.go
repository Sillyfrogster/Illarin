package edit

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func dropUnownedPreservedData(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	blocks []block.Block,
) error {
	owners := make([]uuid.UUID, 0)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			owners = append(owners, element.ID)
			owners = append(owners, block.ItemIDs(element.Content)...)
		}
	}
	if _, err := tx.Exec(ctx, `
		delete from work_preserved_data
		 where work_id = $1 and owner_type <> $2 and owner_id <> all($3)
	`, workID, string(format.OwnerWork), owners); err != nil {
		return fmt.Errorf("drop preserved data with no owner: %w", err)
	}
	return nil
}
