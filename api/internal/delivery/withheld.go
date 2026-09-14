package delivery

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

// takeWithheldNotices returns each withhold of an installed kind the instance has not been told about, and records that it now has.
func takeWithheldNotices(ctx context.Context, queries *db.Queries, instanceID uuid.UUID) ([]WithheldNotice, error) {
	rows, err := queries.TakeWithheldNotices(ctx, db.TakeWithheldNoticesParams{
		InstanceID: uuidValue(instanceID), Kinds: format.InstalledKinds(),
	})
	if err != nil {
		return nil, fmt.Errorf("take withheld notices: %w", err)
	}
	notices := make([]WithheldNotice, 0, len(rows))
	for _, row := range rows {
		notices = append(notices, WithheldNotice{
			AssetID: uuid.UUID(row.AssetID.Bytes), Name: row.Name, WithheldAt: row.WithheldAt.Time,
		})
	}
	return notices, nil
}
