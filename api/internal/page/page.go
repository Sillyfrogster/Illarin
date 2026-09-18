package page

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DetailTag struct {
	Label string
	Value string
}

type Detail struct {
	WorkingCopyVersion  *int64
	UnpublishedChanges  *bool
	ID                  uuid.UUID
	Kind                string
	Name                string
	Blurb               string
	Tags                []DetailTag
	Creator             string
	Identifier          *string
	Dependencies        []Dependency
	IsNSFW              *bool
	Discovery           work.Discovery
	Lifecycle           work.Lifecycle
	IsOwner             bool
	Downloads           []format.Target
	AppTargets          []format.AppTarget
	Original            *work.OriginalUpload
	CreatedAt           time.Time
	Blocks              []block.Block
	Media               []work.DetailImage
	Preview             *string
	LatestUpdate        *work.Version
	Readiness           []work.ReadinessItem
	SealedBlocks        int
	LinkedInstallOnly   bool
	AllowedApps         []string
	EligibleApps        []string
	InstallCapabilities []string
	Withhold            *Withhold
}

func (s *Service) Detail(
	ctx context.Context,
	id uuid.UUID,
	viewerID *uuid.UUID,
	visibility work.ContentVisibility,
) (Detail, error) {
	return s.detail(ctx, id, viewerID, visibility, false)
}

// WorkingCopy reads the page its owner edits, drafted changes included
func (s *Service) WorkingCopy(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility work.ContentVisibility) (Detail, error) {
	if viewerID == nil {
		return Detail{}, work.ErrNotFound
	}
	return s.detail(ctx, id, viewerID, visibility, true)
}

func (s *Service) detail(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility work.ContentVisibility, working bool) (Detail, error) {
	tx, err := s.assets.BeginReadSnapshot(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin asset page snapshot: %w", err)
	}
	defer tx.Rollback(ctx)
	if working {
		if _, err := tx.Exec(ctx, `set local search_path = public`); err != nil {
			return Detail{}, err
		}
	}

	queries := db.New(tx)
	row, err := queries.AssetPage(ctx, db.AssetPageParams{
		ID: uuidToPgtype(id), ViewerID: uuidToNullable(viewerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, work.ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read asset page: %w", err)
	}
	if working && !row.IsOwner {
		return Detail{}, work.ErrNotFound
	}
	found := Detail{
		ID:        uuidFromPgtype(row.ID),
		Kind:      row.Kind,
		Name:      row.Name,
		Blurb:     row.Blurb,
		Tags:      detailTags(row.Tags),
		Creator:   row.Creator,
		IsNSFW:    boolFromPgtype(row.IsNsfw),
		Discovery: work.Discovery(row.Discovery),
		Lifecycle: work.Lifecycle(row.Lifecycle),
		IsOwner:   row.IsOwner,
		Original:  originalUpload(s.reg, row),
		CreatedAt: timeFromPgtype(row.CreatedAt),
		Media:     []work.DetailImage{},
	}
	if row.Identifier != "" {
		found.Identifier = &row.Identifier
	}
	if working || (found.IsOwner && found.Lifecycle == work.LifecycleDraft) {
		var version int64
		if err := tx.QueryRow(ctx, `select working_copy_version from public.assets where id = $1`, id).Scan(&version); err != nil {
			return Detail{}, err
		}
		found.WorkingCopyVersion = &version
	}
	found.Downloads, err = summary.Offered(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.Blocks, err = block.Read(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.Dependencies, err = extensionDependencies(ctx, tx, dependencySubject{
		assetID: id, kind: found.Kind, format: row.OriginalFormat.String, visibility: visibility,
	}, found.Blocks)
	if err != nil {
		return Detail{}, err
	}
	found.LatestUpdate, err = latestUpdate(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	if working && found.Lifecycle == work.LifecyclePublished {
		unpublished, err := s.assets.UnpublishedChanges(ctx, tx, id)
		if err != nil {
			return Detail{}, err
		}
		found.UnpublishedChanges = &unpublished
	}
	if !working && !found.IsOwner {
		if err := private.ApplyPublishedPolicy(ctx, tx, id, found.Blocks); err != nil {
			return Detail{}, err
		}
	}
	if row.WithheldAt.Valid {
		found.Withhold = &Withhold{
			Reason: row.WithheldReason.String,
			At:     row.WithheldAt.Time,
		}
	}

	images, err := queries.AssetPageMedia(ctx, uuidToPgtype(id))
	if err != nil {
		return Detail{}, fmt.Errorf("read asset page media: %w", err)
	}
	flagged := found.IsNSFW != nil && *found.IsNSFW
	blurred := flagged && visibility != work.ContentShown
	draft := found.Lifecycle == work.LifecycleDraft
	for _, image := range images {
		mediaID := uuidFromPgtype(image.ID)
		found.Media = append(found.Media, work.DetailImage{
			ID:        mediaID,
			Role:      work.MediaRole(image.Role),
			IsCover:   image.IsCover,
			DetailURL: s.assets.ImageAddress(mediaID, "detail", blurred, draft || working),
			ThumbURL:  s.assets.ImageAddress(mediaID, "thumb", blurred, draft || working),
			Width:     int(image.Width.Int32),
			Height:    int(image.Height.Int32),
			Bytes:     image.ByteSize,
		})
	}
	if found.IsOwner {
		if err := private.RestorePromptFragments(ctx, tx, id, found.Blocks); err != nil {
			return Detail{}, err
		}
		if draft {
			found.Readiness = work.Readiness(found.Kind, found.Name, found.IsNSFW, found.Blocks)
		} else {
			found.Readiness = work.PublishedShortfall(found.Kind, found.Name, found.IsNSFW, found.Blocks)
		}
		if viewerID != nil {
			sealed, err := private.SealedBlockCount(ctx, tx, *viewerID, id)
			if err != nil {
				return Detail{}, err
			}
			found.SealedBlocks = sealed
		}
	}
	found.AllowedApps, err = private.Apps(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.LinkedInstallOnly = len(found.AllowedApps) > 0 || private.HasPromptFragments(found.Blocks)
	offered := make([]string, len(found.Downloads))
	for i, target := range found.Downloads {
		offered[i] = target.Format
	}
	found.EligibleApps = private.EligibleApps(found.Kind, offered)
	found.InstallCapabilities = format.InstallCapabilities(found.Kind, offered)
	found.AppTargets = format.AppTargets(found.Downloads, s.reg)
	if found.LinkedInstallOnly {
		found.Downloads = []format.Target{}
		found.AppTargets = []format.AppTarget{}
	}
	if draft || working {
		return found, nil
	}
	for _, image := range found.Media {
		if image.IsCover {
			preview := s.assets.ImageAddress(image.ID, "og", flagged, false)
			found.Preview = &preview
			break
		}
	}
	return found, nil
}

func latestUpdate(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (*work.Version, error) {
	var recorded work.Version
	err := tx.QueryRow(ctx, `
		select s.id, s.number, s.recorded_at, s.initial_recorded,
		       s.version_label, s.summary, s.notes
		  from public.assets a
		  join public.asset_snapshots s on s.id = a.published_snapshot_id
		 where a.id = $1
	`, assetID).Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
		&recorded.Initial, &recorded.VersionLabel, &recorded.Summary, &recorded.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the latest recorded version: %w", err)
	}
	return &recorded, nil
}

func originalUpload(reg *format.Registry, row db.AssetPageRow) *work.OriginalUpload {
	if !row.OriginalFormat.Valid {
		return nil
	}
	label := ""
	if declaration, known := reg.Declaration(row.OriginalFormat.String); known {
		label = declaration.Label
	}
	return &work.OriginalUpload{
		Label:     label,
		MediaType: row.OriginalMediaType.String,
		ArrivedAt: timeFromPgtype(row.OriginalArrivedAt),
	}
}

func detailTags(tags []string) []DetailTag {
	out := make([]DetailTag, 0, len(tags))
	for _, tag := range tags {
		value := normalizeBrowseText(tag)
		if value == "" {
			continue
		}
		out = append(out, DetailTag{Label: tag, Value: value})
	}
	return out
}
