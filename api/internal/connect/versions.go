package connect

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
)

// minimumGroupSize is how many installations must report one app version before a page lists it.
const minimumGroupSize = 5

// InstalledAppVersions lists the app versions a work is installed on, counting only connected apps that declare one of the capabilities that install it.
func (s *Sends) InstalledAppVersions(ctx context.Context, workID uuid.UUID, installCapabilities []string) ([]string, error) {
	if len(installCapabilities) == 0 {
		return []string{}, nil
	}
	versions, err := db.New(s.pool).InstalledAppVersions(ctx, db.InstalledAppVersionsParams{
		WorkID: uuidValue(workID), Capabilities: installCapabilities, MinimumGroupSize: minimumGroupSize,
	})
	if err != nil {
		return nil, fmt.Errorf("read installed app versions: %w", err)
	}
	if versions == nil {
		return []string{}, nil
	}
	return versions, nil
}
