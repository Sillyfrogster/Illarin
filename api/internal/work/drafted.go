// Drafted changes are what a published work holds that its readers have not seen yet.
package work

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type versionDigest struct {
	content string
	whole   string
}

func (s *Service) publishedDigest(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (versionDigest, error) {
	if _, err := tx.Exec(ctx, `set local search_path = work_public, public`); err != nil {
		return versionDigest{}, err
	}
	measured, err := s.workDigest(ctx, tx, workID)
	if _, reset := tx.Exec(ctx, `set local search_path = public`); reset != nil {
		return versionDigest{}, reset
	}
	return measured, err
}

func (s *Service) workDigest(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (versionDigest, error) {
	content, err := s.contentFingerprint(ctx, tx, workID)
	if err != nil {
		return versionDigest{}, err
	}
	whole := sha256.New()
	fmt.Fprintf(whole, "content\x00%s\n", content)
	if err := digestDetails(ctx, tx, workID, whole); err != nil {
		return versionDigest{}, err
	}
	blocks, err := block.Read(ctx, tx, workID)
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

func digestDetails(ctx context.Context, tx pgx.Tx, workID uuid.UUID, into hash.Hash) error {
	var name, blurb string
	var tags []string
	var isNSFW *bool
	var cover pgtype.UUID
	err := tx.QueryRow(ctx, `
		select name, blurb, tags, is_nsfw,
		       (select blob_id from work_media where id = works.cover_media_id)
		  from works where id = $1
	`, workID).Scan(&name, &blurb, &tags, &isNSFW, &cover)
	if err != nil {
		return fmt.Errorf("read the catalog fields to compare: %w", err)
	}
	nsfw := "unanswered"
	if isNSFW != nil {
		nsfw = strconv.FormatBool(*isNSFW)
	}
	fmt.Fprintf(into, "catalog\x00%s\x00%s\x00%s\x00%s\x00%s\n",
		name, blurb, strings.Join(tags, "\x00"), nsfw, uuidFromPgtype(cover))
	return nil
}

// UnpublishedChanges says whether a published work has drafted changes it has not published
func (s *Service) UnpublishedChanges(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (bool, error) {
	drafted, err := s.DraftedChanges(ctx, tx, workID)
	return drafted.Any, err
}

// Drafted says whether the working copy differs from the published version at all, and whether its content does
type Drafted struct {
	Any     bool
	Content bool
}

func (s *Service) DraftedChanges(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (Drafted, error) {
	published, err := s.publishedDigest(ctx, tx, workID)
	if err != nil {
		return Drafted{}, err
	}
	reviewed, err := s.workDigest(ctx, tx, workID)
	if err != nil {
		return Drafted{}, err
	}
	return Drafted{
		Any:     reviewed.whole != published.whole,
		Content: reviewed.content != published.content,
	}, nil
}
