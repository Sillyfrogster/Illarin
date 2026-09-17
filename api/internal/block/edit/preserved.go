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
	assetID uuid.UUID,
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
		delete from asset_preserved_data
		 where asset_id = $1 and owner_kind <> $2 and owner_id <> all($3)
	`, assetID, string(format.OwnerAsset), owners); err != nil {
		return fmt.Errorf("drop preserved data with no owner: %w", err)
	}
	return nil
}
