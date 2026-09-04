package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EventPublished and EventUpdated are the immutable record of a post reaching
// or changing its public edition.
const (
	EventPublished = "publication.post.published.v1"
	EventUpdated   = "publication.post.updated.v1"
)

// PublicPost is the one edition a signed-out reader receives.
type PublicPost struct {
	ID          uuid.UUID
	RevisionID  uuid.UUID
	Slug        string
	Title       string
	Summary     string
	Category    Category
	Document    json.RawMessage
	Release     *Release
	Header      *Header
	SocialMedia *PostMedia
	Media       []PostMedia
	Byline      Byline
	PublishedAt time.Time
	UpdatedAt   *time.Time
}

// PublishPost captures the working copy and puts that exact edition in public
// view. Nothing leaves Illarin while the transaction is open.
func (s *Service) PublishPost(ctx context.Context, editor Editor, id uuid.UUID) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin publication: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Document, err = readyToPublish(locked); err != nil {
		return Post{}, err
	}
	revisionID, err := captureRevision(ctx, tx, editor, locked)
	if err != nil {
		return Post{}, err
	}
	if err := carryUsesForward(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	firstTime := locked.PublishedAt == nil
	_, err = tx.Exec(ctx, `
		update posts
		   set status = $2, public_revision_id = $3,
		       published_at = coalesce(published_at, now()),
		       updated_public_at = case when published_at is null then null else now() end,
		       updated_at = now()
		 where id = $1
	`, id, StatusPublished, revisionID)
	if err != nil {
		return Post{}, fmt.Errorf("put the post in public view: %w", err)
	}
	if firstTime {
		if err := captureByline(ctx, tx, id, locked.AuthorID, locked.GrantID); err != nil {
			return Post{}, err
		}
		if err := reserveAddress(ctx, tx, id, editor.ID, locked.Slug); err != nil {
			return Post{}, err
		}
	}
	event := EventUpdated
	if firstTime {
		event = EventPublished
	}
	_, err = tx.Exec(ctx, `
		insert into publication_events (id, post_id, revision_id, type) values ($1, $2, $3, $4)
	`, uuid.New(), id, revisionID, event)
	if err != nil {
		return Post{}, fmt.Errorf("record the publication event: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.published", GrantID: current.GrantID,
		PostID: &id, RevisionID: &revisionID,
		Before: locked.Status, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit publication: %w", err)
	}
	return s.post(ctx, id)
}

// PublishedPost answers the public edition behind one address, current or former, and always names the address it lives at now.
func (s *Service) PublishedPost(ctx context.Context, slug string) (PublicPost, error) {
	var found PublicPost
	var headerID, socialID *uuid.UUID
	var headerAlt, headerCaption *string
	err := s.pool.QueryRow(ctx, `
		select post.id, revision.id, post.slug, revision.title, revision.summary,
		       category.id, category.slug, category.label, category.position,
		       category.retired_at is not null,
		       revision.document, revision.header_media_id, revision.header_alt,
		       revision.header_caption, revision.social_media_id,
		       post.published_at, post.updated_public_at
		  from posts post
		  join post_revisions revision on revision.id = post.public_revision_id
		  join publication_categories category on category.id = revision.category_id
		 where post.status = $2
		   and (post.slug = $1
		        or post.id = (select post_id from post_slugs where slug = $1))
	`, slug, StatusPublished).Scan(
		&found.ID, &found.RevisionID, &found.Slug, &found.Title, &found.Summary,
		&found.Category.ID, &found.Category.Slug, &found.Category.Label,
		&found.Category.Position, &found.Category.Retired,
		&found.Document, &headerID, &headerAlt, &headerCaption, &socialID,
		&found.PublishedAt, &found.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicPost{}, ErrPostNotFound
	}
	if err != nil {
		return PublicPost{}, fmt.Errorf("read a published post: %w", err)
	}
	release, err := s.publishedRelease(ctx, found.RevisionID)
	if err != nil {
		return PublicPost{}, err
	}
	found.Release = release
	found.Header = scanHeader(headerID, headerAlt, headerCaption)
	if err := s.attachRevisionMedia(ctx, &found, socialID); err != nil {
		return PublicPost{}, err
	}
	byline, err := readByline(ctx, s.pool, found.ID)
	if err != nil {
		return PublicPost{}, err
	}
	found.Byline = byline
	return found, nil
}

// attachRevisionMedia gives an edition the exact pictures it was captured with.
func (s *Service) attachRevisionMedia(
	ctx context.Context,
	found *PublicPost,
	socialID *uuid.UUID,
) error {
	rows, err := s.pool.Query(ctx, `
		select media.id, media.post_id, media.purpose, media.width, media.height
		  from post_media_uses use
		  join post_media media on media.id = use.media_id
		 where use.revision_id = $1 and media.blob_id is not null
		 order by media.created_at
	`, found.RevisionID)
	if err != nil {
		return fmt.Errorf("read the pictures a published edition carries: %w", err)
	}
	defer rows.Close()
	found.Media = []PostMedia{}
	for rows.Next() {
		var one PostMedia
		if err := rows.Scan(&one.ID, &one.PostID, &one.Purpose, &one.Width, &one.Height); err != nil {
			return fmt.Errorf("read a picture a published edition carries: %w", err)
		}
		found.Media = append(found.Media, one)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read the pictures a published edition carries: %w", err)
	}
	if socialID != nil {
		for _, one := range found.Media {
			if one.ID == *socialID {
				social := one
				found.SocialMedia = &social
			}
		}
	}
	return nil
}

func (s *Service) publishedRelease(ctx context.Context, revisionID uuid.UUID) (*Release, error) {
	var app App
	var version, address *string
	var markID *uuid.UUID
	var width, height *int
	err := s.pool.QueryRow(ctx, `
		select app.id, app.slug, app.name, app.home_url, app.position,
		       app.retired_at is not null, mark.id, mark.width, mark.height,
		       revision.release_version, revision.release_url
		  from post_revisions revision
		  join publication_apps app on app.id = revision.release_app_id
		  left join publication_media mark
		         on mark.id = app.mark_media_id and mark.blob_id is not null
		 where revision.id = $1
	`, revisionID).Scan(
		&app.ID, &app.Slug, &app.Name, &app.Home, &app.Position, &app.Retired,
		&markID, &width, &height, &version, &address,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the release a post announces: %w", err)
	}
	app.Mark = scanMark(markID, width, height)
	release := &Release{App: app}
	if version != nil {
		release.Version = *version
	}
	if address != nil {
		release.Address = *address
	}
	return release, nil
}

// working is the exact edition a publish transaction locked and will capture.
type working struct {
	ID             uuid.UUID
	AuthorID       uuid.UUID
	GrantID        *uuid.UUID
	CategoryID     uuid.UUID
	CategorySlug   string
	Status         string
	Slug           string
	Title          string
	Summary        string
	Document       []byte
	ReleaseAppID   *uuid.UUID
	ReleaseVersion *string
	ReleaseAddress *string
	HeaderMediaID  *uuid.UUID
	HeaderAlt      *string
	HeaderCaption  *string
	SocialMediaID  *uuid.UUID
	PublishedAt    *time.Time
}

func lockPost(ctx context.Context, tx pgx.Tx, id uuid.UUID) (working, error) {
	var locked working
	var slug *string
	err := tx.QueryRow(ctx, `
		select post.id, post.author_id, post.grant_id, post.category_id, category.slug, post.status,
		       post.slug, post.title, post.summary, post.document,
		       post.release_app_id, post.release_version, post.release_url,
		       post.header_media_id, post.header_alt, post.header_caption, post.social_media_id,
		       post.published_at
		  from posts post
		  join publication_categories category on category.id = post.category_id
		 where post.id = $1
		   for no key update of post
	`, id).Scan(
		&locked.ID, &locked.AuthorID, &locked.GrantID, &locked.CategoryID, &locked.CategorySlug,
		&locked.Status,
		&slug, &locked.Title, &locked.Summary, &locked.Document,
		&locked.ReleaseAppID, &locked.ReleaseVersion, &locked.ReleaseAddress,
		&locked.HeaderMediaID, &locked.HeaderAlt, &locked.HeaderCaption, &locked.SocialMediaID,
		&locked.PublishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return working{}, ErrPostNotFound
	}
	if err != nil {
		return working{}, fmt.Errorf("lock the post being published: %w", err)
	}
	if slug != nil {
		locked.Slug = *slug
	}
	return locked, nil
}

// readyToPublish checks what publication requires and answers the body the
// revision keeps, read at the current document version.
func readyToPublish(locked working) ([]byte, error) {
	if _, err := checkTitle(locked.Title); err != nil {
		return nil, err
	}
	if locked.Summary == "" {
		return nil, FieldError{
			Field:   "summary",
			Message: "Write the summary readers see before the article.",
		}
	}
	if locked.Slug == "" {
		return nil, FieldError{Field: "slug", Message: "Give the post an address."}
	}
	body, err := postdoc.Read(locked.Document)
	if err != nil {
		return nil, documentRefusal(err)
	}
	if body.Empty() {
		return nil, FieldError{Field: "document", Message: "Write the post before publishing it."}
	}
	if locked.CategorySlug == ReleaseCategory {
		if locked.ReleaseAppID == nil || locked.ReleaseVersion == nil {
			return nil, FieldError{
				Field:   "release",
				Message: "A release names the project it belongs to and its version.",
			}
		}
	}
	document, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("write the post body: %w", err)
	}
	return document, nil
}

// carryUsesForward gives the revision the pictures the working copy referred to.
func carryUsesForward(ctx context.Context, tx pgx.Tx, postID, revisionID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		insert into post_media_uses (media_id, post_id, revision_id)
		select media_id, post_id, $2 from post_media_uses
		 where post_id = $1 and revision_id is null
	`, postID, revisionID)
	if err != nil {
		return fmt.Errorf("carry the pictures forward into the revision: %w", err)
	}
	return nil
}

func captureRevision(
	ctx context.Context,
	tx pgx.Tx,
	editor Editor,
	locked working,
) (uuid.UUID, error) {
	id := uuid.New()
	_, err := tx.Exec(ctx, `
		insert into post_revisions (id, post_id, number, title, summary, slug, category_id,
		                            document, document_version, release_app_id,
		                            release_version, release_url, header_media_id,
		                            header_alt, header_caption, social_media_id, captured_by)
		values ($1, $2,
		        coalesce((select max(number) from post_revisions where post_id = $2), 0) + 1,
		        $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, id, locked.ID, locked.Title, locked.Summary, locked.Slug, locked.CategoryID,
		locked.Document, postdoc.Version, locked.ReleaseAppID,
		locked.ReleaseVersion, locked.ReleaseAddress, locked.HeaderMediaID,
		locked.HeaderAlt, locked.HeaderCaption, locked.SocialMediaID, editor.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("capture the post revision: %w", err)
	}
	return id, nil
}
