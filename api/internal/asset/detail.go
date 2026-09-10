package asset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DetailTag struct {
	Label string
	Value string
}

type DetailImage struct {
	ID        uuid.UUID
	Role      MediaRole
	IsCover   bool
	DetailURL string
	ThumbURL  string
	Width     int
	Height    int
	Bytes     int64
}

type Detail struct {
	WorkingCopyVersion *int64
	UnpublishedChanges *bool
	ID                 uuid.UUID
	Kind               string
	Name               string
	Blurb              string
	Tags               []DetailTag
	Creator            string
	IsNSFW             *bool
	Discovery          Discovery
	Lifecycle          Lifecycle
	IsOwner            bool
	Downloads          []format.Target
	Original           *OriginalUpload
	CreatedAt          time.Time
	Blocks             []block.Block
	Media              []DetailImage
	Preview            *string
	LatestUpdate       *Version
	Readiness          []ReadinessItem
	SealedBlocks       int
	LinkedInstallOnly  bool
	AllowedApps        []string
	EligibleApps       []string
	Withhold           *Withhold
}

type Withhold struct {
	Reason string
	Actor  string
	At     time.Time
}

func (s *Service) Detail(
	ctx context.Context,
	id uuid.UUID,
	viewerID *uuid.UUID,
	visibility ContentVisibility,
) (Detail, error) {
	return s.detail(ctx, id, viewerID, visibility, false)
}

func (s *Service) WorkingCopy(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility ContentVisibility) (Detail, error) {
	if viewerID == nil {
		return Detail{}, ErrNotFound
	}
	return s.detail(ctx, id, viewerID, visibility, true)
}

func (s *Service) detail(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, visibility ContentVisibility, working bool) (Detail, error) {
	tx, err := s.beginReadSnapshot(ctx)
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
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("read asset page: %w", err)
	}
	if working && !row.IsOwner {
		return Detail{}, ErrNotFound
	}
	found := Detail{
		ID:        uuidFromPgtype(row.ID),
		Kind:      row.Kind,
		Name:      row.Name,
		Blurb:     row.Blurb,
		Tags:      detailTags(row.Tags),
		Creator:   row.Creator,
		IsNSFW:    boolFromPgtype(row.IsNsfw),
		Discovery: Discovery(row.Discovery),
		Lifecycle: Lifecycle(row.Lifecycle),
		IsOwner:   row.IsOwner,
		Original:  originalUpload(s.reg, row),
		CreatedAt: timeFromPgtype(row.CreatedAt),
		Media:     []DetailImage{},
	}
	if working || (found.IsOwner && found.Lifecycle == LifecycleDraft) {
		var version int64
		if err := tx.QueryRow(ctx, `select working_copy_version from public.assets where id = $1`, id).Scan(&version); err != nil {
			return Detail{}, err
		}
		found.WorkingCopyVersion = &version
	}
	found.Downloads, err = s.exportProjection(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.Blocks, err = readBlocks(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.LatestUpdate, err = latestUpdate(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	if working && found.Lifecycle == LifecyclePublished {
		unpublished, err := s.unpublishedChanges(ctx, tx, id)
		if err != nil {
			return Detail{}, err
		}
		found.UnpublishedChanges = &unpublished
	}
	if !working && !found.IsOwner {
		if err := protected.ApplyPublishedPolicy(ctx, tx, id, found.Blocks); err != nil {
			return Detail{}, err
		}
	}
	if row.WithheldAt.Valid {
		found.Withhold = &Withhold{
			Reason: row.WithheldReason.String,
			Actor:  row.WithheldBy.String,
			At:     row.WithheldAt.Time,
		}
	}

	images, err := queries.AssetPageMedia(ctx, uuidToPgtype(id))
	if err != nil {
		return Detail{}, fmt.Errorf("read asset page media: %w", err)
	}
	flagged := found.IsNSFW != nil && *found.IsNSFW
	blurred := flagged && visibility != ContentShown
	draft := found.Lifecycle == LifecycleDraft
	for _, image := range images {
		mediaID := uuidFromPgtype(image.ID)
		found.Media = append(found.Media, DetailImage{
			ID:        mediaID,
			Role:      MediaRole(image.Role),
			IsCover:   image.IsCover,
			DetailURL: s.variantURL(mediaID, "detail", blurred, draft || working),
			ThumbURL:  s.variantURL(mediaID, "thumb", blurred, draft || working),
			Width:     int(image.Width.Int32),
			Height:    int(image.Height.Int32),
			Bytes:     image.ByteSize,
		})
	}
	if found.IsOwner {
		if err := protected.RestorePromptFragments(ctx, tx, id, found.Blocks); err != nil {
			return Detail{}, err
		}
		if draft {
			found.Readiness = readiness(found.Kind, found.Name, found.IsNSFW, found.Blocks)
		} else {
			found.Readiness = MigratedShortfall(found.Kind, found.Name, found.IsNSFW, found.Blocks)
		}
		if viewerID != nil {
			sealed, err := sealedBlockCount(ctx, tx, *viewerID, id)
			if err != nil {
				return Detail{}, err
			}
			found.SealedBlocks = sealed
		}
	}
	found.AllowedApps, err = protected.Apps(ctx, tx, id)
	if err != nil {
		return Detail{}, err
	}
	found.LinkedInstallOnly = len(found.AllowedApps) > 0 || protected.HasPromptFragments(found.Blocks)
	offered := make([]string, len(found.Downloads))
	for i, target := range found.Downloads {
		offered[i] = target.Format
	}
	found.EligibleApps = protected.EligibleApps(found.Kind, offered)
	if found.LinkedInstallOnly {
		found.Downloads = []format.Target{}
	}
	if draft || working {
		return found, nil
	}
	for _, image := range found.Media {
		if image.IsCover {
			preview := s.variantURL(image.ID, "og", flagged, false)
			found.Preview = &preview
			break
		}
	}
	return found, nil
}

func latestUpdate(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (*Version, error) {
	var recorded Version
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

func (s *Service) unpublishedChanges(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (bool, error) {
	published, err := s.publishedDigest(ctx, tx, assetID)
	if err != nil {
		return false, err
	}
	reviewed, err := s.assetDigest(ctx, tx, assetID)
	if err != nil {
		return false, err
	}
	return reviewed.whole != published.whole, nil
}

func originalUpload(reg *format.Registry, row db.AssetPageRow) *OriginalUpload {
	if !row.OriginalFormat.Valid {
		return nil
	}
	label := ""
	if declaration, known := reg.Declaration(row.OriginalFormat.String); known {
		label = declaration.Label
	}
	return &OriginalUpload{
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

func (s *Service) variantURL(mediaID uuid.UUID, variant string, blurred, private bool) string {
	if blurred {
		variant += "_blurred"
	}
	path := fmt.Sprintf("/media/%s/%s/%d", mediaID, variant, mediaproc.DerivativeVersion)
	if !private {
		return path
	}
	return s.signer.Sign(path, s.now())
}
