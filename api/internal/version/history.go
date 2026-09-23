package version

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) VersionHistory(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
) ([]work.Version, error) {
	owner, err := s.readerRole(ctx, workID, viewerID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select id, number, recorded_at, initial_recorded, version_label, summary, notes,
		       notes_edited_at, withdrawn_at, coalesce(withdrawal_explanation, '')
		  from work_versions where work_id = $1 order by number desc
	`, workID)
	if err != nil {
		return nil, fmt.Errorf("read the version history: %w", err)
	}
	defer rows.Close()
	history := make([]work.Version, 0)
	for rows.Next() {
		var recorded work.Version
		if err := rows.Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
			&recorded.Initial, &recorded.VersionLabel, &recorded.Summary,
			&recorded.Notes, &recorded.NotesEditedAt, &recorded.WithdrawnAt,
			&recorded.WithdrawalExplanation); err != nil {
			return nil, fmt.Errorf("read a recorded version: %w", err)
		}
		if recorded.WithdrawnAt != nil && !owner {
			work.RedactWithdrawn(&recorded)
		}
		history = append(history, recorded)
	}
	return history, rows.Err()
}

func (s *Service) CompareVersions(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
	from, to int,
	preference work.NSFWPreference,
) (Comparison, error) {
	owner, err := s.readerRole(ctx, workID, viewerID)
	if err != nil {
		return Comparison{}, err
	}
	return s.Compare(ctx, ComparisonRequest{
		WorkID: workID, From: from, To: to, AsOwner: owner, NSFWPreference: preference,
		Access: func(version work.Version) string {
			if version.WithdrawnAt != nil && !owner {
				return "This version was withdrawn."
			}
			return ""
		},
	})
}

func (s *Service) readerRole(
	ctx context.Context,
	workID uuid.UUID,
	viewerID *uuid.UUID,
) (bool, error) {
	var owner bool
	err := s.pool.QueryRow(ctx, `
		select coalesce(owner_id = $2, false)
		  from works
		 where id = $1 and deleted_at is null
		   and (lifecycle = 'published' or owner_id = $2)
		   and (taken_down_at is null or owner_id = $2)
	`, workID, viewerID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, work.ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("read the work to compare: %w", err)
	}
	return owner, nil
}
