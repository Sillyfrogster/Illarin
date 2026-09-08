package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// VersionHistory lists the versions an asset has recorded, newest first.
func (s *Service) VersionHistory(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) ([]Version, error) {
	if _, err := s.readerRole(ctx, assetID, viewerID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select id, number, recorded_at, version_label, summary, notes
		  from asset_snapshots where asset_id = $1 order by number desc
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("read the update history: %w", err)
	}
	defer rows.Close()
	history := make([]Version, 0)
	for rows.Next() {
		var recorded Version
		if err := rows.Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
			&recorded.VersionLabel, &recorded.Summary, &recorded.Notes); err != nil {
			return nil, fmt.Errorf("read a recorded version: %w", err)
		}
		history = append(history, recorded)
	}
	return history, rows.Err()
}

// CompareVersions reports what changed between two recorded versions under the rules the reader is under now.
func (s *Service) CompareVersions(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	from, to int,
) (Comparison, error) {
	owner, err := s.readerRole(ctx, assetID, viewerID)
	if err != nil {
		return Comparison{}, err
	}
	return s.Compare(ctx, ComparisonRequest{
		AssetID: assetID, From: from, To: to, AsOwner: owner,
		Access: func(Version) string { return "" },
	})
}

// readerRole says whether this reader may open the asset's history, and whether they own it.
func (s *Service) readerRole(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) (bool, error) {
	var owner bool
	err := s.pool.QueryRow(ctx, `
		select coalesce(owner_id = $2, false)
		  from assets
		 where id = $1 and deleted_at is null
		   and (lifecycle = 'published' or owner_id = $2)
		   and (withheld_at is null or owner_id = $2)
	`, assetID, viewerID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("read the asset to compare: %w", err)
	}
	return owner, nil
}
