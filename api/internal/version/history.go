package version

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) VersionHistory(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
) ([]asset.Version, error) {
	owner, err := s.readerRole(ctx, assetID, viewerID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select id, number, recorded_at, initial_recorded, version_label, summary, notes,
		       notes_edited_at, withdrawn_at, coalesce(withdrawal_explanation, '')
		  from asset_snapshots where asset_id = $1 order by number desc
	`, assetID)
	if err != nil {
		return nil, fmt.Errorf("read the update history: %w", err)
	}
	defer rows.Close()
	history := make([]asset.Version, 0)
	for rows.Next() {
		var recorded asset.Version
		if err := rows.Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
			&recorded.Initial, &recorded.VersionLabel, &recorded.Summary,
			&recorded.Notes, &recorded.NotesEditedAt, &recorded.WithdrawnAt,
			&recorded.WithdrawalExplanation); err != nil {
			return nil, fmt.Errorf("read a recorded version: %w", err)
		}
		if recorded.WithdrawnAt != nil && !owner {
			asset.RedactWithdrawn(&recorded)
		}
		history = append(history, recorded)
	}
	return history, rows.Err()
}

func (s *Service) CompareVersions(
	ctx context.Context,
	assetID uuid.UUID,
	viewerID *uuid.UUID,
	from, to int,
	visibility asset.ContentVisibility,
) (asset.Comparison, error) {
	owner, err := s.readerRole(ctx, assetID, viewerID)
	if err != nil {
		return asset.Comparison{}, err
	}
	return s.assets.Compare(ctx, asset.ComparisonRequest{
		AssetID: assetID, From: from, To: to, AsOwner: owner, Visibility: visibility,
		Access: func(version asset.Version) string {
			if version.WithdrawnAt != nil && !owner {
				return "This version was withdrawn."
			}
			return ""
		},
	})
}

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
		return false, asset.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("read the asset to compare: %w", err)
	}
	return owner, nil
}
