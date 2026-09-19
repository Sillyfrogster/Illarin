package blog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	postbody "github.com/Sillyfrogster/Illarin/api/internal/blog/body"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	announcements "github.com/Sillyfrogster/Illarin/api/internal/integration/blog"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	PostPublished = "publication.post.published.v1"
	PostUpdated   = "publication.post.updated.v1"
)

type PublicPost struct {
	ID           uuid.UUID
	RevisionID   uuid.UUID
	Slug         string
	OriginalSlug string
	Title        string
	Summary      string
	Category     Category
	Document     json.RawMessage
	Release      *Release
	Header       *Header
	SocialMedia  *PostMedia
	Media        []PostMedia
	Byline       Byline
	Related      []PostSummary
	PublishedAt  time.Time
	UpdatedAt    *time.Time
}

func (s *Service) PublishPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
	announcement Announcement,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	chosen, note, err := s.Chosen(ctx, current.GrantID, announcement)
	if err != nil {
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
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status == StatusWithdrawn {
		return Post{}, ErrPostWithdrawn
	}
	if locked.Document, err = readyToPublish(locked); err != nil {
		return Post{}, err
	}
	revisionID, err := captureRevision(ctx, tx, editor, locked, RevisionPublication)
	if err != nil {
		return Post{}, err
	}
	if err := carryUsesForward(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	err = s.makePublic(ctx, tx, locked, revisionID, editor.ID, locked.Slug, captured{
		Chosen: chosen, Note: note,
	})
	if err != nil {
		return Post{}, err
	}
	if err := overtakeSchedule(ctx, tx, editor, locked, StatusPublished); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.published",
		GrantID: current.GrantID,
		PostID:  &id, RevisionID: &revisionID,
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

type captured struct {
	Chosen []announcements.Sending
	Note   string
}

func (s *Service) makePublic(
	ctx context.Context,
	tx pgx.Tx,
	locked working,
	revisionID uuid.UUID,
	actor uuid.UUID,
	address string,
	choice captured,
) error {
	firstTime := locked.PublishedAt == nil
	_, err := tx.Exec(ctx, `
		update posts
		   set status = $2, public_revision_id = $3,
		       slug = case when published_at is null then $4 else slug end,
		       published_at = coalesce(published_at, now()),
		       updated_public_at = case when published_at is null then null else now() end,
		       updated_at = now()
		 where id = $1
	`, locked.ID, StatusPublished, revisionID, address)
	if err != nil {
		return fmt.Errorf("put the post in public view: %w", err)
	}
	if firstTime {
		if err := captureByline(ctx, tx, locked.ID, locked.AuthorID, locked.GrantID); err != nil {
			return err
		}
		if err := reserveAddress(ctx, tx, locked.ID, actor, address); err != nil {
			return err
		}
	}
	event := PostUpdated
	if firstTime {
		event = PostPublished
	}
	eventID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into blog_announcements (id, post_id, revision_id, type, note)
		values ($1, $2, $3, $4, $5)
	`, eventID, locked.ID, revisionID, event, choice.Note)
	if err != nil {
		return fmt.Errorf("record the announcement: %w", err)
	}
	return s.QueueAttempts(ctx, tx, locked.ID, eventID, event, choice.Chosen)
}

func (s *Service) PublishedPost(ctx context.Context, slug string) (PublicPost, error) {
	var found PublicPost
	var headerID, socialID *uuid.UUID
	var headerAlt, headerCaption *string
	err := s.pool.QueryRow(ctx, `
		select post.id, revision.id, post.slug, `+firstAddress+`, revision.title, revision.summary,
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
		&found.ID, &found.RevisionID, &found.Slug, &found.OriginalSlug,
		&found.Title, &found.Summary,
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
	found.Related, err = s.RelatedPosts(ctx, found.ID, found.Category.ID, postApp(found))
	if err != nil {
		return PublicPost{}, err
	}
	return found, nil
}

func postApp(found PublicPost) *uuid.UUID {
	if found.Release != nil {
		return &found.Release.App.ID
	}
	if found.Byline.App != nil {
		return &found.Byline.App.ID
	}
	return nil
}

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
	return s.publishedReleaseWith(ctx, s.pool, revisionID)
}

func (s *Service) publishedReleaseWith(ctx context.Context, q db.DBTX, revisionID uuid.UUID) (*Release, error) {
	var app App
	var version, address *string
	err := q.QueryRow(ctx, `
		select app.id, app.slug, app.name, app.home_url, app.position,
		       app.retired_at is not null,
		       revision.release_version, revision.release_url
		  from post_revisions revision
		  join publication_apps app on app.id = revision.release_app_id
		 where revision.id = $1
	`, revisionID).Scan(
		&app.ID, &app.Slug, &app.Name, &app.Home, &app.Position, &app.Retired,
		&version, &address,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the release a post announces: %w", err)
	}
	release := &Release{App: app}
	if version != nil {
		release.Version = *version
	}
	if address != nil {
		release.Address = *address
	}
	return release, nil
}

type working struct {
	ID               uuid.UUID
	AuthorID         uuid.UUID
	GrantID          *uuid.UUID
	CategoryID       uuid.UUID
	CategorySlug     string
	Status           string
	Slug             string
	Title            string
	Summary          string
	Document         []byte
	ReleaseAppID     *uuid.UUID
	ReleaseVersion   *string
	ReleaseAddress   *string
	HeaderMediaID    *uuid.UUID
	HeaderAlt        *string
	HeaderCaption    *string
	SocialMediaID    *uuid.UUID
	Version          int
	PublishedAt      *time.Time
	DeletedAt        *time.Time
	RecoverableUntil time.Time
	UpdatedAt        time.Time
}

func lockPost(ctx context.Context, tx pgx.Tx, id uuid.UUID) (working, error) {
	locked, err := lockRemovedPost(ctx, tx, id)
	if err != nil {
		return working{}, err
	}
	if locked.DeletedAt != nil {
		return working{}, ErrPostDeleted
	}
	return locked, nil
}

func lockRemovedPost(ctx context.Context, tx pgx.Tx, id uuid.UUID) (working, error) {
	var locked working
	var slug *string
	var recoverableUntil *time.Time
	err := tx.QueryRow(ctx, `
		select post.id, post.author_id, post.grant_id, post.category_id, category.slug, post.status,
		       post.slug, post.title, post.summary, post.document,
		       post.release_app_id, post.release_version, post.release_url,
		       post.header_media_id, post.header_alt, post.header_caption, post.social_media_id,
		       post.working_version, post.published_at,
		       post.deleted_at, post.recoverable_until, post.updated_at
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
		&locked.Version, &locked.PublishedAt,
		&locked.DeletedAt, &recoverableUntil, &locked.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return working{}, ErrPostNotFound
	}
	if err != nil {
		return working{}, fmt.Errorf("lock the post being changed: %w", err)
	}
	if slug != nil {
		locked.Slug = *slug
	}
	if recoverableUntil != nil {
		locked.RecoverableUntil = *recoverableUntil
	}
	return locked, nil
}

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
	body, err := postbody.Read(locked.Document)
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

const StatusDeleted = "deleted"

const RecoveryDays = 30

const RecoveryPoll = time.Hour

var (
	ErrPostDeleted      = errors.New("the post has been deleted")
	ErrPostNotDeleted   = errors.New("the post has not been deleted")
	ErrPostInPublicView = errors.New("the post is still in public view")
	ErrDeletePublished  = errors.New("only an admin may delete a post that has published")
	ErrRecoveryExpired  = errors.New("the recovery window has closed")
)

type Deletion struct {
	At    time.Time
	Until time.Time
	By    string
}

func (s *Service) DeletePost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := mayRemove(editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin deletion: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status == StatusPublished {
		return Post{}, ErrPostInPublicView
	}
	if err := overtakeSchedule(ctx, tx, editor, locked, StatusDeleted); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set deleted_at = now(), recoverable_until = now() + make_interval(days => $2),
		       deleted_by = $3, updated_at = now()
		 where id = $1
	`, id, RecoveryDays, editor.ID)
	if err != nil {
		return Post{}, fmt.Errorf("delete the post: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.deleted",
		GrantID: locked.GrantID,
		PostID:  &id, Before: locked.Status, After: StatusDeleted,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit deletion: %w", err)
	}
	return s.post(ctx, id)
}

func (s *Service) RecoverPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := mayRemove(editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin recovery: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockRemovedPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.DeletedAt == nil {
		return Post{}, ErrPostNotDeleted
	}
	if !locked.RecoverableUntil.After(s.now()) {
		return Post{}, ErrRecoveryExpired
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set deleted_at = null, recoverable_until = null, deleted_by = null,
		       updated_at = now()
		 where id = $1
	`, id)
	if err != nil {
		return Post{}, fmt.Errorf("recover the post: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.recovered",
		GrantID: locked.GrantID,
		PostID:  &id, Before: StatusDeleted, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit recovery: %w", err)
	}
	return s.post(ctx, id)
}

func mayRemove(editor Editor, found Post) error {
	if editor.Admin {
		return nil
	}
	if found.PublishedAt != nil {
		return ErrDeletePublished
	}
	return nil
}

func (s *Service) RunRecovery(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(RecoveryPoll)
	defer ticker.Stop()
	for {
		_, err := s.RemoveExpiredPosts(ctx, s.now())
		if err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) RemoveExpiredPosts(ctx context.Context, now time.Time) (int, error) {
	expired, err := s.expiredPosts(ctx, now)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, id := range expired {
		gone, err := s.removePost(ctx, id, now)
		if err != nil {
			return removed, err
		}
		if gone {
			removed++
		}
	}
	return removed, nil
}

func (s *Service) expiredPosts(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		select id from posts
		 where deleted_at is not null and recoverable_until <= $1
		 order by recoverable_until, id
	`, now)
	if err != nil {
		return nil, fmt.Errorf("read the posts past recovery: %w", err)
	}
	defer rows.Close()
	expired := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a post past recovery: %w", err)
		}
		expired = append(expired, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the posts past recovery: %w", err)
	}
	return expired, nil
}

func (s *Service) removePost(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin removal: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockRemovedPost(ctx, tx, id)
	if errors.Is(err, ErrPostNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if locked.DeletedAt == nil || locked.RecoverableUntil.After(now) {
		return false, nil
	}
	if locked.PublishedAt != nil {
		if err := retireAddresses(ctx, tx, locked); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(ctx, `delete from posts where id = $1`, id); err != nil {
		return false, fmt.Errorf("remove the post: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit removal: %w", err)
	}
	return true, nil
}

func retireAddresses(ctx context.Context, tx pgx.Tx, locked working) error {
	var explanation string
	err := tx.QueryRow(ctx, `
		select coalesce((
			select explanation from post_withdrawals
			 where post_id = $1 order by withdrawn_at desc limit 1
		), '')
	`, locked.ID).Scan(&explanation)
	if err != nil {
		return fmt.Errorf("read what a retired address tells readers: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into post_addresses (slug, current_slug, explanation)
		select reservation.slug, $2, $3 from post_slugs reservation
		 where reservation.post_id = $1
		on conflict (slug) do nothing
	`, locked.ID, locked.Slug, explanation)
	if err != nil {
		return fmt.Errorf("keep the addresses a post published under: %w", err)
	}
	return nil
}

func (s *Service) retiredAddress(ctx context.Context, slug string) (Tombstone, error) {
	var found Tombstone
	err := s.pool.QueryRow(ctx, `
		select current_slug, explanation from post_addresses where slug = $1
	`, slug).Scan(&found.Slug, &found.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tombstone{}, ErrPostNotFound
	}
	if err != nil {
		return Tombstone{}, fmt.Errorf("read a retired address: %w", err)
	}
	return found, nil
}

const PostWithdrawn = "publication.post.withdrawn.v1"

const StatusWithdrawn = "withdrawn"

const withdrawalTextLimit = 500

var (
	ErrPostNotPublic    = errors.New("the post is not in public view")
	ErrPostNotWithdrawn = errors.New("the post is not out of public view")
	ErrPostWithdrawn    = errors.New("the post is out of public view")
)

type Withdrawal struct {
	Reason      string
	Explanation string
	By          string
	At          time.Time
}

type Tombstone struct {
	Slug        string
	Explanation string
}

func (s *Service) WithdrawPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
	reason, explanation string,
	announcement Announcement,
) (Post, error) {
	said, err := checkWithdrawal(reason, explanation)
	if err != nil {
		return Post{}, err
	}
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	chosen, note, err := s.Chosen(ctx, current.GrantID, announcement)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin withdrawal: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostNotPublic
	}
	public, err := publicRevision(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if err := overtakeSchedule(ctx, tx, editor, locked, StatusWithdrawn); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into post_withdrawals (id, post_id, revision_id, reason, explanation, withdrawn_by)
		values ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), id, public, said.reason, said.explanation, editor.ID)
	if err != nil {
		return Post{}, fmt.Errorf("keep why the post was withdrawn: %w", err)
	}
	_, err = tx.Exec(ctx, `
		update posts set status = $2, updated_at = now() where id = $1
	`, id, StatusWithdrawn)
	if err != nil {
		return Post{}, fmt.Errorf("take the post out of public view: %w", err)
	}
	eventID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into blog_announcements (id, post_id, revision_id, type, note)
		values ($1, $2, $3, $4, $5)
	`, eventID, id, public, PostWithdrawn, note)
	if err != nil {
		return Post{}, fmt.Errorf("record the unpublishing announcement: %w", err)
	}
	if err := s.QueueAttempts(ctx, tx, id, eventID, PostWithdrawn, chosen); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.withdrawn",
		GrantID: locked.GrantID,
		PostID:  &id, RevisionID: &public,
		Before: StatusPublished, After: StatusWithdrawn,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit withdrawal: %w", err)
	}
	return s.post(ctx, id)
}

func (s *Service) RepublishPost(
	ctx context.Context,
	editor Editor,
	id, revisionID uuid.UUID,
	version int,
	announcement Announcement,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	chosen, note, err := s.Chosen(ctx, current.GrantID, announcement)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin republication: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status != StatusWithdrawn {
		return Post{}, ErrPostNotWithdrawn
	}
	if _, err := lockedRevision(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	withdrawn, err := publicRevision(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set status = $2, public_revision_id = $3,
		       updated_public_at = case when $4 then now() else updated_public_at end,
		       updated_at = now()
		 where id = $1
	`, id, StatusPublished, revisionID, revisionID != withdrawn)
	if err != nil {
		return Post{}, fmt.Errorf("put the post back in public view: %w", err)
	}
	eventID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into blog_announcements (id, post_id, revision_id, type, note)
		values ($1, $2, $3, $4, $5)
	`, eventID, id, revisionID, PostPublished, note)
	if err != nil {
		return Post{}, fmt.Errorf("record the republishing announcement: %w", err)
	}
	if err := s.QueueAttempts(ctx, tx, id, eventID, PostPublished, chosen); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.republished",
		GrantID: locked.GrantID,
		PostID:  &id, RevisionID: &revisionID,
		Before: StatusWithdrawn, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit republication: %w", err)
	}
	return s.post(ctx, id)
}

func (s *Service) WithdrawnPost(ctx context.Context, slug string) (Tombstone, error) {
	var found Tombstone
	err := s.pool.QueryRow(ctx, `
		select post.slug, withdrawal.explanation
		  from posts post
		  join post_withdrawals withdrawal on withdrawal.post_id = post.id
		 where post.status = $2
		   and (post.slug = $1
		        or post.id = (select post_id from post_slugs where slug = $1))
		 order by withdrawal.withdrawn_at desc
		 limit 1
	`, slug, StatusWithdrawn).Scan(&found.Slug, &found.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.retiredAddress(ctx, slug)
	}
	if err != nil {
		return Tombstone{}, fmt.Errorf("read a withdrawn address: %w", err)
	}
	return found, nil
}

type said struct {
	reason      string
	explanation string
}

func checkWithdrawal(reason, explanation string) (said, error) {
	held := said{reason: oneParagraph(reason), explanation: oneParagraph(explanation)}
	if held.reason == "" {
		return said{}, FieldError{
			Field:   "reason",
			Message: "Say why the post is coming down. Only Illarin reads this.",
		}
	}
	if len(held.reason) > withdrawalTextLimit {
		return said{}, FieldError{
			Field:   "reason",
			Message: fmt.Sprintf("Keep the reason under %d characters.", withdrawalTextLimit),
		}
	}
	if len(held.explanation) > withdrawalTextLimit {
		return said{}, FieldError{
			Field:   "explanation",
			Message: fmt.Sprintf("Keep the public explanation under %d characters.", withdrawalTextLimit),
		}
	}
	if held.explanation != "" && held.explanation == held.reason {
		return said{}, FieldError{
			Field:   "explanation",
			Message: "The public explanation is written for readers. Write it separately.",
		}
	}
	return held, nil
}

func oneParagraph(written string) string {
	return strings.Join(strings.Fields(written), " ")
}

func publicRevision(ctx context.Context, tx pgx.Tx, id uuid.UUID) (uuid.UUID, error) {
	var revisionID *uuid.UUID
	err := tx.QueryRow(ctx, `select public_revision_id from posts where id = $1`, id).
		Scan(&revisionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("read the edition the post is showing: %w", err)
	}
	if revisionID == nil {
		return uuid.Nil, ErrPostNotPublic
	}
	return *revisionID, nil
}

func (s *Service) withdrawalsFor(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]Withdrawal, error) {
	rows, err := s.pool.Query(ctx, `
		select distinct on (withdrawal.post_id)
		       withdrawal.post_id, withdrawal.reason, withdrawal.explanation,
		       actor.username, withdrawal.withdrawn_at
		  from post_withdrawals withdrawal
		  left join users actor on actor.id = withdrawal.withdrawn_by
		 where withdrawal.post_id = any($1)
		 order by withdrawal.post_id, withdrawal.withdrawn_at desc
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("read why posts were withdrawn: %w", err)
	}
	defer rows.Close()
	latest := make(map[uuid.UUID]Withdrawal, len(ids))
	for rows.Next() {
		var postID uuid.UUID
		var one Withdrawal
		var actor *string
		if err := rows.Scan(&postID, &one.Reason, &one.Explanation, &actor, &one.At); err != nil {
			return nil, fmt.Errorf("read why a post was withdrawn: %w", err)
		}
		if actor != nil {
			one.By = *actor
		}
		latest[postID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read why posts were withdrawn: %w", err)
	}
	return latest, nil
}
