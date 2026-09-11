package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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
	Announcement UpdateAnnouncement
}

// UpdateAnnouncement is the creator's choice of where one update is announced.
type UpdateAnnouncement struct {
	DestinationIDs   *[]uuid.UUID
	AnnounceUnlisted bool
}

type Update struct {
	ID                uuid.UUID
	AssetID           uuid.UUID
	Number            int
	RecordedAt        time.Time
	VersionLabel      string
	Summary           string
	Notes             string
	ContentGeneration int
	ContentChanged    bool
}

type AnnounceUpdate func(ctx context.Context, tx pgx.Tx, published Update, choice UpdateAnnouncement) error

func (s *Service) OnUpdatePublished(announce AnnounceUpdate) {
	s.announce = announce
}

func (s *Service) PublishUpdate(
	ctx context.Context,
	in UpdateRequest,
	candidate *Candidate,
) (Update, []ReadinessItem, error) {
	in.Summary = strings.TrimSpace(in.Summary)
	in.Notes = strings.TrimSpace(in.Notes)
	in.VersionLabel = strings.TrimSpace(in.VersionLabel)
	if in.Summary == "" {
		return Update{}, nil, ErrSummaryRequired
	}
	if utf8.RuneCountInString(in.Summary) > MaxSummaryRunes ||
		utf8.RuneCountInString(in.Notes) > MaxNotesRunes ||
		utf8.RuneCountInString(in.VersionLabel) > MaxVersionLabelRunes {
		return Update{}, nil, ErrSummaryTooLong
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Update{}, nil, err
	}
	defer tx.Rollback(ctx)

	kind, err := candidate.Lock(ctx, tx, in.OwnerID, in.AssetID)
	if err != nil {
		return Update{}, nil, err
	}
	var name, lifecycle string
	var isNSFW *bool
	err = tx.QueryRow(ctx, `
		select name, is_nsfw, lifecycle from assets where id = $1
	`, in.AssetID).Scan(&name, &isNSFW, &lifecycle)
	if err != nil {
		return Update{}, nil, fmt.Errorf("read the asset to update: %w", err)
	}
	if Lifecycle(lifecycle) != LifecyclePublished {
		return Update{}, nil, ErrAssetIsDraft
	}

	blocks, err := readBlocks(ctx, tx, in.AssetID)
	if err != nil {
		return Update{}, nil, err
	}
	items, err := s.candidateReadiness(ctx, tx, in.AssetID, kind, name, isNSFW, blocks)
	if err != nil {
		return Update{}, nil, err
	}
	if !Ready(items) {
		return Update{}, items, ErrPublishFloor
	}

	published, err := s.publishedDigest(ctx, tx, in.AssetID)
	if err != nil {
		return Update{}, nil, err
	}
	reviewed, err := s.assetDigest(ctx, tx, in.AssetID)
	if err != nil {
		return Update{}, nil, err
	}
	if reviewed.whole == published.whole {
		return Update{}, nil, ErrNothingToPublish
	}

	recorded, err := s.recordUpdate(ctx, tx, in, reviewed.content != published.content)
	if err != nil {
		return Update{}, nil, err
	}
	if s.announce != nil {
		if err := s.announce(ctx, tx, recorded, in.Announcement); err != nil {
			return Update{}, nil, err
		}
	}
	if err := candidate.commit(ctx, tx, in.AssetID); err != nil {
		return Update{}, nil, err
	}
	return recorded, items, nil
}

func (s *Service) recordUpdate(
	ctx context.Context,
	tx pgx.Tx,
	in UpdateRequest,
	contentChanged bool,
) (Update, error) {
	_, err := tx.Exec(ctx, `
		update assets
		   set content_generation = content_generation + case when $2 then 1 else 0 end,
		       updated_at = now()
		 where id = $1
	`, in.AssetID, contentChanged)
	if err != nil {
		return Update{}, fmt.Errorf("move the content generation: %w", err)
	}
	var chosenLabel *string
	if in.VersionLabel != "" {
		chosenLabel = &in.VersionLabel
	}
	var snapshotID pgtype.UUID
	err = tx.QueryRow(ctx, `select record_asset_snapshot($1, false, $2, $3, $4)`,
		in.AssetID, in.Summary, in.Notes, chosenLabel).Scan(&snapshotID)
	if err != nil {
		return Update{}, fmt.Errorf("record the update: %w", err)
	}
	if !snapshotID.Valid {
		return Update{}, ErrNotFound
	}
	recorded := Update{
		ID: uuidFromPgtype(snapshotID), AssetID: in.AssetID, ContentChanged: contentChanged,
	}
	err = tx.QueryRow(ctx, `
		select number, recorded_at, version_label, summary, notes, content_generation
		  from asset_snapshots where id = $1
	`, recorded.ID).Scan(&recorded.Number, &recorded.RecordedAt, &recorded.VersionLabel,
		&recorded.Summary, &recorded.Notes, &recorded.ContentGeneration)
	if err != nil {
		return Update{}, fmt.Errorf("read the recorded update: %w", err)
	}
	return recorded, nil
}

type versionDigest struct {
	content string
	whole   string
}

func (s *Service) publishedDigest(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (versionDigest, error) {
	if _, err := tx.Exec(ctx, `set local search_path = asset_public, public`); err != nil {
		return versionDigest{}, err
	}
	measured, err := s.assetDigest(ctx, tx, assetID)
	if _, reset := tx.Exec(ctx, `set local search_path = public`); reset != nil {
		return versionDigest{}, reset
	}
	return measured, err
}

func (s *Service) assetDigest(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (versionDigest, error) {
	content, err := s.contentFingerprint(ctx, tx, assetID)
	if err != nil {
		return versionDigest{}, err
	}
	whole := sha256.New()
	fmt.Fprintf(whole, "content\x00%s\n", content)
	if err := digestCatalog(ctx, tx, assetID, whole); err != nil {
		return versionDigest{}, err
	}
	blocks, err := readBlocks(ctx, tx, assetID)
	if err != nil {
		return versionDigest{}, err
	}
	for _, holder := range blocks {
		var title string
		if holder.Title != nil {
			title = *holder.Title
		}
		fmt.Fprintf(whole, "page\x00%s\x00%s\x00%t\x00%s\x00%s\n",
			holder.Definition, title, holder.Hidden, holder.Layout, holder.Width)
	}
	return versionDigest{content: content, whole: hex.EncodeToString(whole.Sum(nil))}, nil
}

func digestCatalog(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, into hash.Hash) error {
	var name, blurb string
	var tags []string
	var isNSFW *bool
	var cover pgtype.UUID
	err := tx.QueryRow(ctx, `
		select name, blurb, tags, is_nsfw, cover_media_id from assets where id = $1
	`, assetID).Scan(&name, &blurb, &tags, &isNSFW, &cover)
	if err != nil {
		return fmt.Errorf("read the catalog fields to compare: %w", err)
	}
	adult := "unanswered"
	if isNSFW != nil {
		adult = strconv.FormatBool(*isNSFW)
	}
	fmt.Fprintf(into, "catalog\x00%s\x00%s\x00%s\x00%s\x00%s\n",
		name, blurb, strings.Join(tags, "\x00"), adult, uuidFromPgtype(cover))
	return nil
}
