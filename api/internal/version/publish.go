package version

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
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
	ErrSummaryRequired  = errors.New("an update needs a summary")
	ErrSummaryTooLong   = errors.New("the update text is too long")
	ErrNothingToPublish = errors.New("nothing has changed since the last update")
)

type UpdateRequest struct {
	OwnerID      uuid.UUID
	AssetID      uuid.UUID
	Summary      string
	Notes        string
	VersionLabel string
	Announcement asset.UpdateAnnouncement
}

func (s *Service) PublishUpdate(
	ctx context.Context,
	in UpdateRequest,
	candidate *asset.Candidate,
) (asset.Update, []asset.ReadinessItem, error) {
	in.Summary = strings.TrimSpace(in.Summary)
	in.Notes = strings.TrimSpace(in.Notes)
	in.VersionLabel = strings.TrimSpace(in.VersionLabel)
	if in.Summary == "" {
		return asset.Update{}, nil, ErrSummaryRequired
	}
	if utf8.RuneCountInString(in.Summary) > MaxSummaryRunes ||
		utf8.RuneCountInString(in.Notes) > MaxNotesRunes ||
		utf8.RuneCountInString(in.VersionLabel) > MaxVersionLabelRunes {
		return asset.Update{}, nil, ErrSummaryTooLong
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return asset.Update{}, nil, err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, in.OwnerID, in.AssetID)
	if err != nil {
		return asset.Update{}, nil, err
	}
	var name, lifecycle string
	var isNSFW *bool
	err = tx.QueryRow(ctx, `
		select name, is_nsfw, lifecycle from assets where id = $1
	`, in.AssetID).Scan(&name, &isNSFW, &lifecycle)
	if err != nil {
		return asset.Update{}, nil, fmt.Errorf("read the asset to update: %w", err)
	}
	if asset.Lifecycle(lifecycle) != asset.LifecyclePublished {
		return asset.Update{}, nil, asset.ErrAssetIsDraft
	}

	blocks, err := block.Read(ctx, tx, in.AssetID)
	if err != nil {
		return asset.Update{}, nil, err
	}
	items, err := s.assets.CandidateReadiness(ctx, tx, in.AssetID, kind, name, isNSFW, blocks)
	if err != nil {
		return asset.Update{}, nil, err
	}
	if !asset.Ready(items) {
		return asset.Update{}, items, asset.ErrPublishFloor
	}

	drafted, err := s.assets.DraftedChanges(ctx, tx, in.AssetID)
	if err != nil {
		return asset.Update{}, nil, err
	}
	if !drafted.Any {
		return asset.Update{}, nil, ErrNothingToPublish
	}

	recorded, err := s.recordUpdate(ctx, tx, in, drafted.Content)
	if err != nil {
		return asset.Update{}, nil, err
	}
	for _, listen := range s.assets.UpdateListeners() {
		if err := listen(ctx, tx, recorded, in.Announcement); err != nil {
			return asset.Update{}, nil, err
		}
	}
	if err := candidate.Commit(ctx, tx, in.AssetID); err != nil {
		return asset.Update{}, nil, err
	}
	return recorded, items, nil
}

func (s *Service) recordUpdate(
	ctx context.Context,
	tx pgx.Tx,
	in UpdateRequest,
	contentChanged bool,
) (asset.Update, error) {
	_, err := tx.Exec(ctx, `
		update assets
		   set content_generation = content_generation + case when $2 then 1 else 0 end,
		       updated_at = now()
		 where id = $1
	`, in.AssetID, contentChanged)
	if err != nil {
		return asset.Update{}, fmt.Errorf("move the content generation: %w", err)
	}
	var chosenLabel *string
	if in.VersionLabel != "" {
		chosenLabel = &in.VersionLabel
	}
	var snapshotID pgtype.UUID
	err = tx.QueryRow(ctx, `select record_asset_snapshot($1, false, $2, $3, $4)`,
		in.AssetID, in.Summary, in.Notes, chosenLabel).Scan(&snapshotID)
	if err != nil {
		return asset.Update{}, fmt.Errorf("record the update: %w", err)
	}
	if !snapshotID.Valid {
		return asset.Update{}, asset.ErrNotFound
	}
	recorded := asset.Update{
		ID: snapshotID.Bytes, AssetID: in.AssetID, ContentChanged: contentChanged,
	}
	err = tx.QueryRow(ctx, `
		select number, recorded_at, version_label, summary, notes, content_generation
		  from asset_snapshots where id = $1
	`, recorded.ID).Scan(&recorded.Number, &recorded.RecordedAt, &recorded.VersionLabel,
		&recorded.Summary, &recorded.Notes, &recorded.ContentGeneration)
	if err != nil {
		return asset.Update{}, fmt.Errorf("read the recorded update: %w", err)
	}
	return recorded, nil
}
