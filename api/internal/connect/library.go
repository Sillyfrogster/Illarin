package connect

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
)

func (s *Sends) Sync(
	ctx context.Context,
	app ConnectedApp,
	report ReportedLibrary,
) (LibraryResult, error) {
	entries, removed, err := s.readReport(report)
	if err != nil {
		return LibraryResult{}, err
	}
	version, err := checkAppVersion(report.AppVersion)
	if err != nil {
		return LibraryResult{}, ErrLibraryVersion
	}
	action, limit := actionSyncPart, int32(syncPartLimit)
	if report.Snapshot {
		action, limit = actionSyncWhole, int32(syncWholeLimit)
	}
	if err := s.apps.Throttle(
		ctx, action, app.ID.String(), limit, time.Hour,
	); err != nil {
		return LibraryResult{}, err
	}

	workIDs := make([]uuid.UUID, 0, len(entries))
	versions := make([]int32, 0, len(entries))
	for _, entry := range entries {
		workIDs = append(workIDs, entry.WorkID)
		reported := int32(0)
		if entry.VersionNumber != nil {
			reported = int32(*entry.VersionNumber)
		}
		versions = append(versions, reported)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LibraryResult{}, fmt.Errorf("begin a library report: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	accepted, err := queries.ReportLibraryEntries(ctx, db.ReportLibraryEntriesParams{
		ConnectedAppID: uuidValue(app.ID), WorkIds: uuidValues(workIDs),
		VersionNumbers: versions,
	})
	if err != nil {
		return LibraryResult{}, fmt.Errorf("record a library report: %w", err)
	}
	if err := queries.RecordLibraryAppVersion(ctx, db.RecordLibraryAppVersionParams{
		AppVersion: version, ConnectedAppID: uuidValue(app.ID),
	}); err != nil {
		return LibraryResult{}, fmt.Errorf("record the reported app version: %w", err)
	}
	var dropped int64
	if report.Snapshot {
		dropped, err = queries.PruneLibraryToWhole(ctx, db.PruneLibraryToWholeParams{
			ConnectedAppID: uuidValue(app.ID), WorkIds: uuidValues(workIDs),
		})
	} else if len(removed) > 0 {
		dropped, err = queries.RemoveLibraryEntries(ctx, db.RemoveLibraryEntriesParams{
			ConnectedAppID: uuidValue(app.ID), WorkIds: uuidValues(removed),
		})
	}
	if err != nil {
		return LibraryResult{}, fmt.Errorf("remove library entries: %w", err)
	}
	withheld, err := takeWithheldNotices(ctx, queries, app.ID)
	if err != nil {
		return LibraryResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LibraryResult{}, fmt.Errorf("commit a library report: %w", err)
	}
	return LibraryResult{
		Accepted: int(accepted), Removed: int(dropped),
		Ignored: len(entries) - int(accepted), Withheld: withheld,
	}, nil
}

func (s *Sends) readReport(report ReportedLibrary) ([]ReportedEntry, []uuid.UUID, error) {
	if len(report.Entries) > s.settings.MaxLibraryEntries ||
		len(report.Removed) > s.settings.MaxLibraryEntries {
		return nil, nil, ErrLibraryTooLarge
	}
	if report.Snapshot && len(report.Removed) > 0 {
		return nil, nil, ErrLibraryReport
	}
	entries := make([]ReportedEntry, 0, len(report.Entries))
	seen := make(map[uuid.UUID]struct{}, len(report.Entries))
	for _, entry := range report.Entries {
		if entry.VersionNumber != nil && *entry.VersionNumber < 1 {
			return nil, nil, ErrLibraryReport
		}
		if _, repeated := seen[entry.WorkID]; repeated {
			return nil, nil, ErrLibraryReport
		}
		seen[entry.WorkID] = struct{}{}
		entries = append(entries, entry)
	}
	removed := make([]uuid.UUID, 0, len(report.Removed))
	for _, workID := range report.Removed {
		if _, installed := seen[workID]; installed {
			return nil, nil, ErrLibraryReport
		}
		removed = append(removed, workID)
	}
	return entries, removed, nil
}

func (s *Sends) LibraryCountsByApp(
	ctx context.Context,
	userID uuid.UUID,
) (map[uuid.UUID]LibraryCounts, error) {
	rows, err := db.New(s.pool).AppLibraryCounts(ctx, uuidValue(userID))
	if err != nil {
		return nil, fmt.Errorf("count installed works: %w", err)
	}
	counts := make(map[uuid.UUID]LibraryCounts, len(rows))
	for _, row := range rows {
		counts[uuid.UUID(row.ConnectedAppID.Bytes)] = LibraryCounts{
			Installed: int(row.Installed), UpdatesAvailable: int(row.UpdatesAvailable),
		}
	}
	return counts, nil
}
