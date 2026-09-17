package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrUpdateDestinationIneligible = errors.New("choose only your own verified, active update destinations")

type UpdateAnnouncement struct {
	DestinationIDs   *[]uuid.UUID
	AnnounceUnlisted bool
	Notify           bool
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

// UpdateListener is called inside the publish transaction with every update, so what it records commits with the update.
type UpdateListener func(ctx context.Context, tx pgx.Tx, published Update, choice UpdateAnnouncement) error

func (s *Service) OnUpdatePublished(listeners ...UpdateListener) {
	s.updateListeners = append(s.updateListeners, listeners...)
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
	blocks, err := block.Read(ctx, tx, assetID)
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
		for _, element := range holder.Elements {
			if element.Content == nil || element.Content.Empty() {
				continue
			}
			options, err := json.Marshal(element.Options)
			if err != nil {
				return versionDigest{}, err
			}
			fmt.Fprintf(whole, "display\x00%s\x00%s\x00%s\x00%s\n", element.Type, element.Role, element.Slot, options)
		}
	}
	return versionDigest{content: content, whole: hex.EncodeToString(whole.Sum(nil))}, nil
}

func digestCatalog(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, into hash.Hash) error {
	var name, blurb string
	var tags []string
	var isNSFW *bool
	var cover pgtype.UUID
	err := tx.QueryRow(ctx, `
		select name, blurb, tags, is_nsfw,
		       (select blob_id from asset_media where id = assets.cover_media_id)
		  from assets where id = $1
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

// UnpublishedChanges says whether a published work has drafted changes it has not published
func (s *Service) UnpublishedChanges(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (bool, error) {
	drafted, err := s.DraftedChanges(ctx, tx, assetID)
	return drafted.Any, err
}

// Drafted says whether the working copy differs from the published version at all, and whether its content does
type Drafted struct {
	Any     bool
	Content bool
}

func (s *Service) DraftedChanges(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (Drafted, error) {
	published, err := s.publishedDigest(ctx, tx, assetID)
	if err != nil {
		return Drafted{}, err
	}
	reviewed, err := s.assetDigest(ctx, tx, assetID)
	if err != nil {
		return Drafted{}, err
	}
	return Drafted{
		Any:     reviewed.whole != published.whole,
		Content: reviewed.content != published.content,
	}, nil
}

func (s *Service) UpdateListeners() []UpdateListener {
	return s.updateListeners
}
