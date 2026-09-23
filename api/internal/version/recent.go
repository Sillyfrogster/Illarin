package version

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

// RecentVersion is one version on a creator's profile feed, with the work it belongs to.
type RecentVersion struct {
	WorkID       uuid.UUID
	WorkName     string
	WorkType     string
	Number       int
	Initial      bool
	VersionLabel string
	Summary      string
	RecordedAt   time.Time
	CoverURL     string
}

// RecentByCreator lists the newest versions across a creator's works that are in Browse, under the reader's adult content setting.
func (s *Service) RecentByCreator(
	ctx context.Context,
	creatorID uuid.UUID,
	preference work.NSFWPreference,
	limit int,
) ([]RecentVersion, error) {
	rows, err := s.pool.Query(ctx, `
		select version.work_id, owned.name, owned.type, version.number, version.initial_recorded,
		       coalesce(version.version_label, ''), version.summary, version.recorded_at,
		       cover.id, owned.is_nsfw
		  from work_versions version
		  join works owned on owned.id = version.work_id
		  left join work_media cover
		         on cover.id = owned.cover_media_id and cover.work_id = owned.id
		        and cover.is_current and cover.blob_id is not null
		 where owned.owner_id = $1
		   and owned.lifecycle = 'published' and owned.visibility = 'listed'
		   and owned.deleted_at is null and owned.taken_down_at is null
		   and version.withdrawn_at is null
		   and ($2::text <> 'hidden' or not owned.is_nsfw)
		 order by version.recorded_at desc, version.number desc
		 limit $3
	`, creatorID, string(preference), limit)
	if err != nil {
		return nil, fmt.Errorf("read a creator's recent versions: %w", err)
	}
	defer rows.Close()
	recent := make([]RecentVersion, 0, limit)
	for rows.Next() {
		var version RecentVersion
		var coverID *uuid.UUID
		var isNSFW *bool
		if err := rows.Scan(
			&version.WorkID, &version.WorkName, &version.WorkType, &version.Number, &version.Initial,
			&version.VersionLabel, &version.Summary, &version.RecordedAt, &coverID, &isNSFW,
		); err != nil {
			return nil, fmt.Errorf("read a recent version: %w", err)
		}
		if coverID != nil {
			flagged := isNSFW != nil && *isNSFW
			version.CoverURL = s.works.ImageAddress(*coverID, "thumb", preference != work.NSFWShown && flagged, false)
		}
		recent = append(recent, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read a creator's recent versions: %w", err)
	}
	return recent, nil
}
