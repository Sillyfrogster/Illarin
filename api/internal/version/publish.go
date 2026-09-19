package version

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	MaxSummaryRunes      = 200
	MaxNotesRunes        = 4000
	MaxVersionLabelRunes = 60
)

var (
	ErrSummaryRequired  = errors.New("a version needs a summary")
	ErrSummaryTooLong   = errors.New("the version text is too long")
	ErrNothingToPublish = errors.New("nothing has changed since the last version")
)

type PublishRequest struct {
	OwnerID      uuid.UUID
	WorkID       uuid.UUID
	Summary      string
	Notes        string
	VersionLabel string
	Announcement Announcement
}

func (s *Service) PublishVersion(
	ctx context.Context,
	in PublishRequest,
	candidate *work.Candidate,
) (Version, []work.ReadinessItem, error) {
	in.Summary = strings.TrimSpace(in.Summary)
	in.Notes = strings.TrimSpace(in.Notes)
	in.VersionLabel = strings.TrimSpace(in.VersionLabel)
	if in.Summary == "" {
		return Version{}, nil, ErrSummaryRequired
	}
	if utf8.RuneCountInString(in.Summary) > MaxSummaryRunes ||
		utf8.RuneCountInString(in.Notes) > MaxNotesRunes ||
		utf8.RuneCountInString(in.VersionLabel) > MaxVersionLabelRunes {
		return Version{}, nil, ErrSummaryTooLong
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Version{}, nil, err
	}
	defer tx.Rollback(ctx)

	workType, err := candidate.Lock(ctx, tx, in.OwnerID, in.WorkID)
	if err != nil {
		return Version{}, nil, err
	}
	var name, lifecycle string
	var isNSFW *bool
	err = tx.QueryRow(ctx, `
		select name, is_nsfw, lifecycle from works where id = $1
	`, in.WorkID).Scan(&name, &isNSFW, &lifecycle)
	if err != nil {
		return Version{}, nil, fmt.Errorf("read the work to publish a version of: %w", err)
	}
	if work.Lifecycle(lifecycle) != work.LifecyclePublished {
		return Version{}, nil, work.ErrWorkIsDraft
	}

	blocks, err := block.Read(ctx, tx, in.WorkID)
	if err != nil {
		return Version{}, nil, err
	}
	items, err := s.works.CandidateReadiness(ctx, tx, in.WorkID, workType, name, isNSFW, blocks)
	if err != nil {
		return Version{}, nil, err
	}
	if !work.Ready(items) {
		return Version{}, items, work.ErrPublishFloor
	}

	drafted, err := s.works.DraftedChanges(ctx, tx, in.WorkID)
	if err != nil {
		return Version{}, nil, err
	}
	if !drafted.Any {
		return Version{}, nil, ErrNothingToPublish
	}

	recorded, err := s.record(ctx, tx, in, drafted.Content)
	if err != nil {
		return Version{}, nil, err
	}
	for _, listen := range s.Listeners() {
		if err := listen(ctx, tx, recorded, in.Announcement); err != nil {
			return Version{}, nil, err
		}
	}
	if err := candidate.Commit(ctx, tx, in.WorkID); err != nil {
		return Version{}, nil, err
	}
	return recorded, items, nil
}

func (s *Service) record(
	ctx context.Context,
	tx pgx.Tx,
	in PublishRequest,
	contentChanged bool,
) (Version, error) {
	if _, err := tx.Exec(ctx, `update works set updated_at = now() where id = $1`, in.WorkID); err != nil {
		return Version{}, fmt.Errorf("mark the work as changed: %w", err)
	}
	var chosenLabel *string
	if in.VersionLabel != "" {
		chosenLabel = &in.VersionLabel
	}
	var versionID pgtype.UUID
	err := tx.QueryRow(ctx, `select record_work_version($1, false, $2, $3, $4)`,
		in.WorkID, in.Summary, in.Notes, chosenLabel).Scan(&versionID)
	if err != nil {
		return Version{}, fmt.Errorf("record the version: %w", err)
	}
	if !versionID.Valid {
		return Version{}, work.ErrNotFound
	}
	recorded := Version{
		ID: versionID.Bytes, WorkID: in.WorkID, ContentChanged: contentChanged,
	}
	err = tx.QueryRow(ctx, `
		select number, recorded_at, version_label, summary, notes
		  from work_versions where id = $1
	`, recorded.ID).Scan(&recorded.Number, &recorded.RecordedAt, &recorded.VersionLabel,
		&recorded.Summary, &recorded.Notes)
	if err != nil {
		return Version{}, fmt.Errorf("read the recorded version: %w", err)
	}
	return recorded, nil
}
