package asset

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/jackc/pgx/v5"
)

// TellWatchers records an update for the accounts watching the asset when it changed the file and was not published quietly.
func TellWatchers(ctx context.Context, tx pgx.Tx, published Update, choice UpdateAnnouncement) error {
	if !published.ContentChanged || !choice.Notify {
		return nil
	}
	var name string
	if err := tx.QueryRow(ctx, `select name from assets where id = $1`, published.AssetID).Scan(&name); err != nil {
		return fmt.Errorf("read the name of the updated asset: %w", err)
	}
	return notify.Record(ctx, tx, notify.Event{
		Type: notify.AssetUpdated, Asset: &published.AssetID,
		Words: notify.Words{
			AssetName: name, UpdateNumber: published.Number,
			VersionLabel: published.VersionLabel, Summary: published.Summary,
		},
	})
}
