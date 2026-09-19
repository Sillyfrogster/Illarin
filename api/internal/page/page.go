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
	DraftedChangesVersion *int64
	UnpublishedChanges    *bool
	ID                    uuid.UUID
	Type                  string
	Name                  string
	Blurb                 string
	Tags                  []DetailTag
	Creator               string
	Identifier            *string
	Dependencies          []Dependency
	IsNSFW                *bool
	Visibility            work.Visibility
	Lifecycle             work.Lifecycle
	IsOwner               bool
	Downloads             []format.Offered
	AppFormats            []format.AppFormat
	Original              *work.OriginalUpload
	CreatedAt             time.Time
	Blocks                []block.Block
	Media                 []work.DetailImage
	Preview               *string
	LatestVersion         *work.Version
	Readiness             []work.ReadinessItem
	SealedBlocks          int
	LinkedInstallOnly     bool
	AllowedApps           []string
	EligibleApps          []string
	InstallCapabilities   []string
	Withhold              *Withhold
}

func (s *Service) Detail(
	ctx context.Context,
	id uuid.UUID,
	viewerID *uuid.UUID,
	preference work.NSFWPreference,
) (Detail, error) {
	return s.detail(ctx, id, viewerID, preference, false)
}

// DraftedChanges reads the page its owner edits, drafted changes included
func (s *Service) DraftedChanges(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, preference work.NSFWPreference) (Detail, error) {
	if viewerID == nil {
		return Detail{}, work.ErrNotFound
	}
	return s.detail(ctx, id, viewerID, preference, true)
}

func (s *Service) detail(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, preference work.NSFWPreference, working bool) (Detail, error) {
	tx, err := s.works.BeginReadSnapshot(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin work page snapshot: %w", err)
	}
	defer tx.Rollback(ctx)
	if working {
		if _, err := tx.Exec(ctx, `set local search_path = public`); err != nil {
			return Detail{}, err
		}
	}

	queries := db.New(tx)
	row, err := queries.WorkPage(ctx, db.WorkPageParams{
		ID: uuidToPgtype(id), ViewerID: uuidToNullable(viewerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, work.ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read work page: %w", err)
	}
	if working && !row.IsOwner {
		return Detail{}, work.ErrNotFound
	}
	found := Detail{
		ID:         uuidFromPgtype(row.ID),
		Type:       row.Type,
		Name:       row.Name,
		Blurb:      row.Blurb,
		Tags:       detailTags(row.Tags),
		Creator:    row.Creator,
		IsNSFW:     boolFromPgtype(row.IsNsfw),
		Visibility: work.Visibility(row.Visibility),
		Lifecycle:  work.Lifecycle(row.Lifecycle),
		IsOwner:    row.IsOwner,
		Original:   originalUpload(s.reg, row),
		CreatedAt:  timeFromPgtype(row.CreatedAt),
		Media:      []work.DetailImage{},
	}
	if row.Identifier != "" {
		found.Identifier = &row.Identifier
	}
	if working || (found.IsOwner && found.Lifecycle == work.LifecycleDraft) {
		var version int64
		if err := tx.QueryRow(ctx, `select drafted_changes_version from public.works where id = $1`, id).Scan(&version); err != nil {
			return Detail{}, err
		}
		found.DraftedChangesVersion = &version
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
		workID: id, workType: found.Type, format: row.OriginalFormat.String, preference: preference,
	}, found.Blocks)
	if err != nil {
		return Detail{}, err
	}
	found.LatestVersion, err = latestVersion(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	if working && found.Lifecycle == work.LifecyclePublished {
		unpublished, err := s.works.UnpublishedChanges(ctx, tx, id)
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

	images, err := queries.WorkPageMedia(ctx, uuidToPgtype(id))
	if err != nil {
		return Detail{}, fmt.Errorf("read work page media: %w", err)
	}
	flagged := found.IsNSFW != nil && *found.IsNSFW
	blurred := flagged && preference != work.NSFWShown
	draft := found.Lifecycle == work.LifecycleDraft
	for _, image := range images {
		mediaID := uuidFromPgtype(image.ID)
		found.Media = append(found.Media, work.DetailImage{
			ID:        mediaID,
			Role:      work.MediaRole(image.Role),
			IsCover:   image.IsCover,
			DetailURL: s.works.ImageAddress(mediaID, "detail", blurred, draft || working),
			ThumbURL:  s.works.ImageAddress(mediaID, "thumb", blurred, draft || working),
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
			found.Readiness = work.Readiness(found.Type, found.Name, found.IsNSFW, found.Blocks)
		} else {
			found.Readiness = work.PublishedShortfall(found.Type, found.Name, found.IsNSFW, found.Blocks)
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
	offered := format.OfferedIDs(found.Downloads)
	found.EligibleApps = private.EligibleApps(s.reg, offered)
	found.InstallCapabilities = format.InstallCapabilities(found.Type, offered)
	found.AppFormats = format.AppFormats(found.Downloads, s.reg)
	if found.LinkedInstallOnly {
		found.Downloads = []format.Offered{}
		found.AppFormats = []format.AppFormat{}
	}
	if draft || working {
		return found, nil
	}
	for _, image := range found.Media {
		if image.IsCover {
			preview := s.works.ImageAddress(image.ID, "og", flagged, false)
			found.Preview = &preview
			break
		}
	}
	return found, nil
}

func latestVersion(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (*work.Version, error) {
	var recorded work.Version
	err := tx.QueryRow(ctx, `
		select s.id, s.number, s.recorded_at, s.initial_recorded,
		       s.version_label, s.summary, s.notes
		  from public.works a
		  join public.work_versions s on s.id = a.published_version_id
		 where a.id = $1
	`, workID).Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
		&recorded.Initial, &recorded.VersionLabel, &recorded.Summary, &recorded.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the latest recorded version: %w", err)
	}
	return &recorded, nil
}

func originalUpload(reg *format.Registry, row db.WorkPageRow) *work.OriginalUpload {
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
