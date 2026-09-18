package connect

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

// takeWithheldNotices returns each withhold of an installed type the instance has not been told about, and records that it now has.
func takeWithheldNotices(ctx context.Context, queries *db.Queries, instanceID uuid.UUID) ([]WithheldWork, error) {
	rows, err := queries.TakeWithheldNotices(ctx, db.TakeWithheldNoticesParams{
		InstanceID: uuidValue(instanceID), Types: format.InstalledTypes(),
	})
	if err != nil {
		return nil, fmt.Errorf("take withheld notices: %w", err)
	}
	notices := make([]WithheldWork, 0, len(rows))
	for _, row := range rows {
		notices = append(notices, WithheldWork{
			WorkID: uuid.UUID(row.WorkID.Bytes), Name: row.Name, WithheldAt: row.WithheldAt.Time,
		})
	}
	return notices, nil
}
