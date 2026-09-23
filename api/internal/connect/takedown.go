package connect

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

// takeTakedownNotices returns the takedowns of installed extensions the app has not heard about yet, and records that it now has
func takeTakedownNotices(ctx context.Context, queries *db.Queries, appID uuid.UUID) ([]TakenDownWork, error) {
	rows, err := queries.TakeTakedownNotices(ctx, db.TakeTakedownNoticesParams{
		ConnectedAppID: uuidValue(appID), Types: format.InstalledTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("take takedown notices: %w", err)
	}
	notices := make([]TakenDownWork, 0, len(rows))
	for _, row := range rows {
		notices = append(notices, TakenDownWork{
			WorkID: uuid.UUID(row.WorkID.Bytes), Name: row.Name, TakenDownAt: row.TakenDownAt.Time,
		})
	}
	return notices, nil
}
