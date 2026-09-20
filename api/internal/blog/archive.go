package blog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const ArchivePageSize = 12

const relatedLimit = 3

type PostSummary struct {
	ID           uuid.UUID
	Slug         string
	OriginalSlug string
	Title        string
	Summary      string
	Category     Category
	Byline       Byline
	PublishedAt  time.Time
	UpdatedAt    *time.Time
}

type ArchiveQuery struct {
	Page     int
	Category string
}

type Archive struct {
	Posts    []PostSummary
	Page     int
	Pages    int
	Total    int
	Category *Category
}

const selectSummaries = `
	select post.id, post.slug, ` + firstAddress + `, revision.title, revision.summary,
	       category.id, category.slug, category.label, category.position,
	       category.retired_at is not null,
	       post.published_at, post.updated_public_at
	  from posts post
	  join post_revisions revision on revision.id = post.public_revision_id
	  join blog_categories category on category.id = revision.category_id
	 where post.status = 'published'
	`

const narrowArchive = `
		   and ($1::uuid is null or revision.category_id = $1)
	`

func (s *Service) ReadableCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+`
		 where exists (
		       select 1
		         from posts post
		         join post_revisions revision on revision.id = post.public_revision_id
		        where post.status = 'published' and revision.category_id = category.id
		 )
		 order by category.position, category.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("read the categories a reader can browse: %w", err)
	}
	defer rows.Close()
	return collectCategories(rows)
}

func (s *Service) Archive(ctx context.Context, asked ArchiveQuery) (Archive, error) {
	found := Archive{Page: asked.Page, Posts: []PostSummary{}}
	var categoryID *uuid.UUID
	if asked.Category != "" {
		category, err := s.categoryBySlug(ctx, asked.Category)
		if err != nil {
			return Archive{}, err
		}
		found.Category, categoryID = &category, &category.ID
	}
	if err := s.pool.QueryRow(ctx, `
		select count(*)
		  from posts post
		  join post_revisions revision on revision.id = post.public_revision_id
		 where post.status = 'published'
	`+narrowArchive, categoryID).Scan(&found.Total); err != nil {
		return Archive{}, fmt.Errorf("count the published archive: %w", err)
	}
	found.Pages = (found.Total + ArchivePageSize - 1) / ArchivePageSize
	rows, err := s.pool.Query(ctx, selectSummaries+narrowArchive+`
		 order by greatest(post.published_at, post.updated_public_at) desc,
		          post.id desc
		 limit $2 offset $3
	`, categoryID, ArchivePageSize, (asked.Page-1)*ArchivePageSize)
	if err != nil {
		return Archive{}, fmt.Errorf("read the published archive: %w", err)
	}
	if found.Posts, err = s.collectSummaries(ctx, rows); err != nil {
		return Archive{}, err
	}
	return found, nil
}

func (s *Service) RelatedPosts(
	ctx context.Context,
	postID uuid.UUID,
	category uuid.UUID,
) ([]PostSummary, error) {
	rows, err := s.pool.Query(ctx, selectSummaries+`
		   and post.id <> $1
		 order by revision.category_id = $2 desc,
		          greatest(post.published_at, post.updated_public_at) desc,
		          post.id desc
		 limit $3
	`, postID, category, relatedLimit)
	if err != nil {
		return nil, fmt.Errorf("read the further reading an article offers: %w", err)
	}
	return s.collectSummaries(ctx, rows)
}

func (s *Service) collectSummaries(ctx context.Context, rows pgx.Rows) ([]PostSummary, error) {
	defer rows.Close()
	listed := []PostSummary{}
	for rows.Next() {
		one, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		listed = append(listed, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the published archive: %w", err)
	}
	return s.attachBylines(ctx, listed)
}

func (s *Service) attachBylines(ctx context.Context, listed []PostSummary) ([]PostSummary, error) {
	if len(listed) == 0 {
		return listed, nil
	}
	ids := make([]uuid.UUID, 0, len(listed))
	for _, one := range listed {
		ids = append(ids, one.ID)
	}
	held, err := bylinesFor(ctx, s.pool, ids)
	if err != nil {
		return nil, err
	}
	for index := range listed {
		listed[index].Byline = held[listed[index].ID]
	}
	return listed, nil
}

func scanSummary(row rowScanner) (PostSummary, error) {
	var one PostSummary
	err := row.Scan(
		&one.ID, &one.Slug, &one.OriginalSlug, &one.Title, &one.Summary,
		&one.Category.ID, &one.Category.Slug, &one.Category.Label,
		&one.Category.Position, &one.Category.Retired,
		&one.PublishedAt, &one.UpdatedAt,
	)
	if err != nil {
		return PostSummary{}, fmt.Errorf("read a published archive entry: %w", err)
	}
	return one, nil
}

func (s *Service) categoryBySlug(ctx context.Context, slug string) (Category, error) {
	var found Category
	err := s.pool.QueryRow(ctx, selectCategories+` where category.slug = $1`, slug).
		Scan(&found.ID, &found.Slug, &found.Label, &found.Position, &found.Retired)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrCategoryNotFound
	}
	if err != nil {
		return Category{}, fmt.Errorf("read a blog category: %w", err)
	}
	return found, nil
}

type Byline struct {
	AccountID    *uuid.UUID
	Handle       string
	DisplayName  string
	ContactEmail string
	Avatar       *Portrait
}

type Portrait struct {
	MediaID           uuid.UUID
	Width             int
	Height            int
	DerivativeVersion uint32
}

type snapshot struct {
	AccountID    uuid.UUID
	Handle       string
	DisplayName  string
	ContactEmail string
	AvatarID     *uuid.UUID
}

func takeSnapshot(
	ctx context.Context,
	tx pgx.Tx,
	accountID uuid.UUID,
) (snapshot, error) {
	taken := snapshot{AccountID: accountID}
	var restricted bool
	err := tx.QueryRow(ctx, `
		select account.username, restriction.user_id is not null,
		       coalesce(profile.display_name, ''), coalesce(profile.contact_email, ''),
		       avatar.id
		  from users account
		  left join profile_restrictions restriction on restriction.user_id = account.id
		  left join public_profiles profile on profile.user_id = account.id
		  left join profile_media avatar
		         on avatar.id = profile.avatar_media_id and avatar.blob_id is not null
		 where account.id = $1
	`, accountID).Scan(
		&taken.Handle, &restricted, &taken.DisplayName, &taken.ContactEmail, &taken.AvatarID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return snapshot{}, ErrAccountNotFound
	}
	if err != nil {
		return snapshot{}, fmt.Errorf("read the author's public identity: %w", err)
	}
	if restricted {
		taken.DisplayName, taken.ContactEmail, taken.AvatarID = "", "", nil
	}
	return taken, nil
}

func captureByline(
	ctx context.Context,
	tx pgx.Tx,
	postID, authorID uuid.UUID,
) error {
	taken, err := takeSnapshot(ctx, tx, authorID)
	if err != nil {
		return err
	}
	return writeByline(ctx, tx, postID, taken, `on conflict (post_id) do nothing`)
}

func replaceByline(
	ctx context.Context,
	tx pgx.Tx,
	postID, accountID uuid.UUID,
) error {
	taken, err := takeSnapshot(ctx, tx, accountID)
	if err != nil {
		return err
	}
	return writeByline(ctx, tx, postID, taken, `
		on conflict (post_id) do update
		   set account_id = excluded.account_id, handle = excluded.handle,
		       display_name = excluded.display_name, contact_email = excluded.contact_email,
		       avatar_media_id = excluded.avatar_media_id, captured_at = now()`)
}

func writeByline(
	ctx context.Context,
	tx pgx.Tx,
	postID uuid.UUID,
	taken snapshot,
	whenHeld string,
) error {
	_, err := tx.Exec(ctx, `
		insert into post_bylines (post_id, account_id, handle, display_name, contact_email,
		                          avatar_media_id)
		values ($1, $2, $3, $4, $5, $6)
	`+whenHeld,
		postID, taken.AccountID, taken.Handle, taken.DisplayName, taken.ContactEmail,
		taken.AvatarID)
	if err != nil {
		return fmt.Errorf("record the post byline: %w", err)
	}
	return nil
}

const selectBylines = `
	select byline.post_id, byline.account_id, byline.handle, byline.display_name,
	       byline.contact_email, avatar.id, avatar.width, avatar.height
	  from post_bylines byline
	  left join profile_media avatar
	         on avatar.id = byline.avatar_media_id and avatar.blob_id is not null
	`

func readByline(ctx context.Context, pool queryRower, postID uuid.UUID) (Byline, error) {
	_, found, err := scanByline(pool.QueryRow(ctx,
		selectBylines+` where byline.post_id = $1`, postID))
	return found, err
}

func bylinesFor(
	ctx context.Context,
	pool rowQuerier,
	ids []uuid.UUID,
) (map[uuid.UUID]Byline, error) {
	rows, err := pool.Query(ctx, selectBylines+` where byline.post_id = any($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("read the post bylines: %w", err)
	}
	defer rows.Close()
	held := make(map[uuid.UUID]Byline, len(ids))
	for rows.Next() {
		postID, one, err := scanByline(rows)
		if err != nil {
			return nil, err
		}
		held[postID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the post bylines: %w", err)
	}
	return held, nil
}

func scanByline(row rowScanner) (uuid.UUID, Byline, error) {
	var postID uuid.UUID
	var found Byline
	var avatarID *uuid.UUID
	var width, height *int
	err := row.Scan(
		&postID, &found.AccountID, &found.Handle, &found.DisplayName, &found.ContactEmail,
		&avatarID, &width, &height,
	)
	if err != nil {
		return uuid.Nil, Byline{}, fmt.Errorf("read the post byline: %w", err)
	}
	found.Avatar = scanPortrait(avatarID, width, height)
	return postID, found, nil
}

func scanPortrait(mediaID *uuid.UUID, width, height *int) *Portrait {
	if mediaID == nil || width == nil || height == nil {
		return nil
	}
	return &Portrait{
		MediaID:           *mediaID,
		Width:             *width,
		Height:            *height,
		DerivativeVersion: mediaproc.DerivativeVersion,
	}
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type rowScanner interface {
	Scan(into ...any) error
}

var (
	ErrNotPostAdmin     = errors.New("only an Illarin admin may correct a published post")
	ErrPostNotPublished = errors.New("the post has not been published")
)

func (s *Service) CorrectAddress(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	candidate string,
) (Post, error) {
	if !editor.Admin {
		return Post{}, ErrNotPostAdmin
	}
	slug, err := checkSlug(candidate)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin address correction: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostNotPublished
	}
	if slug == locked.Slug {
		return Post{}, FieldError{Field: "slug", Message: "The post already lives at that address."}
	}
	taken, err := addressTaken(ctx, tx, id, slug)
	if err != nil {
		return Post{}, err
	}
	if taken {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err := reserveAddress(ctx, tx, id, editor.ID, slug); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `update posts set slug = $2, updated_at = now() where id = $1`, id, slug)
	if isUniqueViolation(err) {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err != nil {
		return Post{}, fmt.Errorf("move the post to its corrected address: %w", err)
	}
	err = recordActivity(ctx, tx, change{
		Actor: editor.ID, Action: "post.address.corrected", PostID: &id, Before: StatusPublished, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit address correction: %w", err)
	}
	return s.post(ctx, id)
}

func (s *Service) CorrectByline(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	handle string,
) (Post, error) {
	if !editor.Admin {
		return Post{}, ErrNotPostAdmin
	}
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin byline correction: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostNotPublished
	}
	if err := replaceByline(ctx, tx, id, accountID); err != nil {
		return Post{}, err
	}
	err = recordActivity(ctx, tx, change{
		Actor: editor.ID, Action: "post.byline.corrected", PostID: &id, SubjectID: &accountID, Before: StatusPublished, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit byline correction: %w", err)
	}
	return s.post(ctx, id)
}

const postSlugLimit = 80

const firstAddress = `coalesce((
	                 select earliest.slug from post_slugs earliest
	                  where earliest.post_id = post.id
	                  order by earliest.reserved_at, earliest.slug
	                  limit 1
	               ), post.slug)`

var reservedSlugs = map[string]bool{
	"admin":       true,
	"api":         true,
	"app":         true,
	"apps":        true,
	"archive":     true,
	"atom":        true,
	"blog":        true,
	"category":    true,
	"feed":        true,
	"feed-json":   true,
	"feed-xml":    true,
	"feeds":       true,
	"media":       true,
	"page":        true,
	"preview":     true,
	"robots":      true,
	"rss":         true,
	"sitemap":     true,
	"tag":         true,
	"tags":        true,
	"unpublished": true,
}

func normalizeSlug(candidate string) string {
	var out strings.Builder
	previousHyphen := true
	for _, letter := range strings.ToLower(strings.TrimSpace(candidate)) {
		switch {
		case unicode.IsLetter(letter) && letter < unicode.MaxASCII, unicode.IsDigit(letter) && letter < unicode.MaxASCII:
			out.WriteRune(letter)
			previousHyphen = false
		case previousHyphen:
			continue
		default:
			out.WriteRune('-')
			previousHyphen = true
		}
		if out.Len() >= postSlugLimit {
			break
		}
	}
	return strings.Trim(out.String(), "-")
}

func checkSlug(candidate string) (string, error) {
	slug := normalizeSlug(candidate)
	if slug == "" {
		return "", FieldError{Field: "slug", Message: "Give the post an address."}
	}
	if reservedSlugs[slug] {
		return "", FieldError{Field: "slug", Message: "The blog already answers on that address."}
	}
	return slug, nil
}

func addressTaken(
	ctx context.Context,
	reader queryRower,
	postID uuid.UUID,
	slug string,
) (bool, error) {
	var taken bool
	err := reader.QueryRow(ctx, `
		select exists (select 1 from posts where slug = $2 and id <> $1)
		    or exists (select 1 from post_slugs where slug = $2 and post_id is distinct from $1)
	`, postID, slug).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("read who holds a post address: %w", err)
	}
	return taken, nil
}

func reserveAddress(
	ctx context.Context,
	writer execer,
	postID, actorID uuid.UUID,
	slug string,
) error {
	_, err := writer.Exec(ctx, `
		insert into post_slugs (slug, post_id, reserved_by) values ($1, $2, $3)
		on conflict (slug) do nothing
	`, slug, postID, actorID)
	if err != nil {
		return fmt.Errorf("reserve the post address: %w", err)
	}
	return nil
}
