package publication

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	PurposeHeader   = "header"
	PurposeDocument = "document"
	PurposeSocial   = "social"
)

const (
	shownVariant   = "detail"
	galleryVariant = "grid"
	socialVariant  = "og"
)

var ErrPostMediaNotFound = errors.New("no such post media")

type PostMedia struct {
	ID      uuid.UUID
	PostID  uuid.UUID
	Purpose string
	Width   int
	Height  int
}

type Header struct {
	MediaID uuid.UUID
	Alt     string
	Caption string
}

func PostMediaURL(mediaID uuid.UUID, purpose string, version uint32) string {
	variant := shownVariant
	if purpose == PurposeSocial {
		variant = socialVariant
	}
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, variant, version)
}

func PostMediaThumbURL(mediaID uuid.UUID, version uint32) string {
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, galleryVariant, version)
}

func (s *Service) SignPrivate(path string) string {
	return s.signer.Sign(path, s.now())
}

func (s *Service) AddPostMedia(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
	purpose string,
	file io.Reader,
) (PostMedia, error) {
	current, err := s.post(ctx, postID)
	if err != nil {
		return PostMedia{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return PostMedia{}, err
	}
	if current.Deletion != nil {
		return PostMedia{}, ErrPostDeleted
	}
	if purpose != PurposeHeader && purpose != PurposeDocument && purpose != PurposeSocial {
		return PostMedia{}, FieldError{
			Field:   "purpose",
			Message: "A picture is uploaded as a header, a document picture or a social image.",
		}
	}
	stored, prepared, err := s.media.Accept(ctx, file)
	if err != nil {
		return PostMedia{}, err
	}
	added := PostMedia{
		ID: uuid.New(), PostID: postID, Purpose: purpose,
		Width: prepared.Width, Height: prepared.Height,
	}
	_, err = s.pool.Exec(ctx, `
		insert into post_media (id, post_id, blob_id, purpose, width, height)
		values ($1, $2, $3, $4, $5, $6)
	`, added.ID, postID, stored.ID, purpose, added.Width, added.Height)
	if err != nil {
		return PostMedia{}, fmt.Errorf("record post media: %w", err)
	}
	return added, nil
}

func (s *Service) PostMediaVariant(
	ctx context.Context,
	mediaID uuid.UUID,
	variant string,
	version uint32,
	expires, signature string,
) (string, string, bool, error) {
	_, ordinary := mediaproc.VariantByName(variant)
	_, composed := mediaproc.SocialPreviewByName(variant)
	if (!ordinary && !composed) || version != mediaproc.DerivativeVersion {
		return "", "", false, ErrPostMediaNotFound
	}
	var blobID uuid.UUID
	var digestBytes []byte
	var published bool
	err := s.pool.QueryRow(ctx, `
		select media.blob_id, blob.sha256,
		       exists (select 1
		                 from post_media_uses use
		                 join post_revisions revision on revision.id = use.revision_id
		                where use.media_id = media.id and revision.captured_for = $2)
		  from post_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, mediaID, RevisionPublication).Scan(&blobID, &digestBytes, &published)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, ErrPostMediaNotFound
	}
	if err != nil {
		return "", "", false, fmt.Errorf("find post media: %w", err)
	}
	if !published {
		path := fmt.Sprintf("/media/%s/%s/%d", mediaID, variant, version)
		if !s.signer.Valid(path, expires, signature, s.now()) {
			return "", "", false, ErrPostMediaNotFound
		}
	}
	if len(digestBytes) != sha256.Size {
		return "", "", false, fmt.Errorf("post media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, variant, version)
	if err != nil {
		return "", "", false, err
	}
	return redirect, s.media.DerivativeType(), !published, nil
}

type placed struct {
	header  *Header
	social  *uuid.UUID
	ordered []uuid.UUID
}

func (s *Service) checkPlacement(
	ctx context.Context,
	postID uuid.UUID,
	body postdoc.Document,
	in PostSave,
) (placed, error) {
	owned, err := s.ownedMedia(ctx, postID)
	if err != nil {
		return placed{}, err
	}
	var chosen placed
	for _, named := range body.MediaIDs() {
		id, err := uuid.Parse(named)
		if err != nil {
			return placed{}, fmt.Errorf("read a placed picture: %w", err)
		}
		if err := belongs(owned, id, PurposeDocument, "document"); err != nil {
			return placed{}, err
		}
		chosen.ordered = append(chosen.ordered, id)
	}
	if in.Header != nil {
		if err := belongs(owned, in.Header.MediaID, PurposeHeader, "header.mediaId"); err != nil {
			return placed{}, err
		}
		alt := strings.TrimSpace(in.Header.Alt)
		if alt == "" || len([]rune(alt)) > headerTextLimit {
			return placed{}, FieldError{
				Field:   "header.alt",
				Message: "Describe the header picture for a reader who cannot see it.",
			}
		}
		caption := strings.TrimSpace(in.Header.Caption)
		if len([]rune(caption)) > headerTextLimit {
			return placed{}, FieldError{
				Field:   "header.caption",
				Message: fmt.Sprintf("Keep the caption to %d characters or fewer.", headerTextLimit),
			}
		}
		chosen.header = &Header{MediaID: in.Header.MediaID, Alt: alt, Caption: caption}
		chosen.ordered = append(chosen.ordered, in.Header.MediaID)
	}
	if in.SocialMediaID != nil {
		if err := belongs(owned, *in.SocialMediaID, PurposeSocial, "socialMediaId"); err != nil {
			return placed{}, err
		}
		chosen.social = in.SocialMediaID
		chosen.ordered = append(chosen.ordered, *in.SocialMediaID)
	}
	return chosen, nil
}

func belongs(owned map[uuid.UUID]string, id uuid.UUID, purpose, field string) error {
	held, mine := owned[id]
	if !mine {
		return FieldError{
			Field:   field,
			Message: "That picture does not belong to this post. Upload it here first.",
		}
	}
	if held != purpose {
		return FieldError{
			Field:   field,
			Message: fmt.Sprintf("That picture was uploaded as a %s picture.", held),
		}
	}
	return nil
}

func (s *Service) ownedMedia(ctx context.Context, postID uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := s.pool.Query(ctx, `
		select id, purpose from post_media where post_id = $1 and blob_id is not null
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("read the pictures a post owns: %w", err)
	}
	defer rows.Close()
	owned := make(map[uuid.UUID]string, 8)
	for rows.Next() {
		var id uuid.UUID
		var purpose string
		if err := rows.Scan(&id, &purpose); err != nil {
			return nil, fmt.Errorf("read a picture a post owns: %w", err)
		}
		owned[id] = purpose
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the pictures a post owns: %w", err)
	}
	return owned, nil
}

func recordUses(
	ctx context.Context,
	tx pgx.Tx,
	postID uuid.UUID,
	revisionID *uuid.UUID,
	ordered []uuid.UUID,
) error {
	if revisionID == nil {
		_, err := tx.Exec(ctx, `
			delete from post_media_uses where post_id = $1 and revision_id is null
		`, postID)
		if err != nil {
			return fmt.Errorf("clear what the working copy refers to: %w", err)
		}
	}
	written := make(map[uuid.UUID]bool, len(ordered))
	for _, id := range ordered {
		if written[id] {
			continue
		}
		written[id] = true
		_, err := tx.Exec(ctx, `
			insert into post_media_uses (media_id, post_id, revision_id) values ($1, $2, $3)
		`, id, postID, revisionID)
		if err != nil {
			return fmt.Errorf("record what an edition refers to: %w", err)
		}
	}
	return nil
}
