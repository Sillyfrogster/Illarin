package version

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/jackc/pgx/v5"
)

// TellFollowers tells the accounts following the work about a version that changed the file and was not published quietly.
func TellFollowers(ctx context.Context, tx pgx.Tx, published Version, choice Announcement) error {
	if !published.ContentChanged || !choice.Notify {
		return nil
	}
	var name string
	if err := tx.QueryRow(ctx, `select name from works where id = $1`, published.WorkID).Scan(&name); err != nil {
		return fmt.Errorf("read the name of the updated work: %w", err)
	}
	return notify.Record(ctx, tx, notify.Event{
		Type: notify.WorkUpdated, Work: &published.WorkID,
		Words: notify.Words{
			WorkName: name, UpdateNumber: published.Number,
			VersionLabel: published.VersionLabel, Summary: published.Summary,
		},
	})
}
