package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ArchivePageSize is how many posts one numbered archive page carries.
const ArchivePageSize = 12

// relatedLimit is how many further posts an article offers a reader.
const relatedLimit = 3

// PostSummary is one published post as an archive lists it. Every field comes
// from the published edition, so a listing writes no excerpt and reads no live
// profile.
type PostSummary struct {
	ID             uuid.UUID
	Slug           string
	Title          string
	Summary        string
	Category       Category
	App            *App
	ReleaseVersion string
	Byline         Byline
	PublishedAt    time.Time
	UpdatedAt      *time.Time
}

// ArchiveQuery is the page a reader asked for and the scope they asked for it in.
type ArchiveQuery struct {
	Page     int
	Category string
	App      string
}

// Archive is one page of the publication and the scope it was read under.
type Archive struct {
	Posts    []PostSummary
	Page     int
	Pages    int
	Total    int
	Category *Category
	App      *App
}

// selectSummaries reads the parts of a published edition an archive entry shows.
// A post's app is the one its release names, and otherwise the one its byline
// was captured under.
const selectSummaries = `
	select post.id, post.slug, revision.title, revision.summary,
	       category.id, category.slug, category.label, category.position,
	       category.retired_at is not null,
	       app.id, app.slug, app.name, app.home_url, app.position,
	       app.retired_at is not null, mark.id, mark.width, mark.height,
	       revision.release_version,
	       post.published_at, post.updated_public_at
	  from posts post
	  join post_revisions revision on revision.id = post.public_revision_id
	  join publication_categories category on category.id = revision.category_id
	  left join post_bylines byline on byline.post_id = post.id
	  left join publication_apps app
	         on app.id = coalesce(revision.release_app_id, byline.app_id)
	  left join publication_media mark
	         on mark.id = app.mark_media_id and mark.blob_id is not null
	 where post.status = 'published'
	`

// narrowArchive holds an archive to one category and one app, and both the
// count and the page read it.
const narrowArchive = `
		   and ($1::uuid is null or revision.category_id = $1)
		   and ($2::uuid is null
		        or coalesce(revision.release_app_id, byline.app_id) = $2)
	`

// ReadableCategories answers the categories that carry published posts, in the
// order the publication shows them.
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

// Archive answers one page of the publication, newest first, narrowed to a
// category or an app where the reader named one.
func (s *Service) Archive(ctx context.Context, asked ArchiveQuery) (Archive, error) {
	found := Archive{Page: asked.Page, Posts: []PostSummary{}}
	var categoryID, appID *uuid.UUID
	if asked.Category != "" {
		category, err := s.categoryBySlug(ctx, asked.Category)
		if err != nil {
			return Archive{}, err
		}
		found.Category, categoryID = &category, &category.ID
	}
	if asked.App != "" {
		app, err := s.appBySlug(ctx, asked.App)
		if err != nil {
			return Archive{}, err
		}
		found.App, appID = &app, &app.ID
	}
	if err := s.pool.QueryRow(ctx, `
		select count(*)
		  from posts post
		  join post_revisions revision on revision.id = post.public_revision_id
		  left join post_bylines byline on byline.post_id = post.id
		 where post.status = 'published'
	`+narrowArchive, categoryID, appID).Scan(&found.Total); err != nil {
		return Archive{}, fmt.Errorf("count the published archive: %w", err)
	}
	found.Pages = (found.Total + ArchivePageSize - 1) / ArchivePageSize
	rows, err := s.pool.Query(ctx, selectSummaries+narrowArchive+`
		 order by greatest(post.published_at, post.updated_public_at) desc,
		          post.id desc
		 limit $3 offset $4
	`, categoryID, appID, ArchivePageSize, (asked.Page-1)*ArchivePageSize)
	if err != nil {
		return Archive{}, fmt.Errorf("read the published archive: %w", err)
	}
	if found.Posts, err = s.collectSummaries(ctx, rows); err != nil {
		return Archive{}, err
	}
	return found, nil
}

// RelatedPosts answers the further reading one article offers, preferring the
// same app, then the same category, and otherwise the newest writing there is.
func (s *Service) RelatedPosts(
	ctx context.Context,
	postID uuid.UUID,
	category uuid.UUID,
	app *uuid.UUID,
) ([]PostSummary, error) {
	rows, err := s.pool.Query(ctx, selectSummaries+`
		   and post.id <> $1
		 order by ($2::uuid is not null
		           and coalesce(revision.release_app_id, byline.app_id)
		               is not distinct from $2::uuid) desc,
		          revision.category_id = $3 desc,
		          greatest(post.published_at, post.updated_public_at) desc,
		          post.id desc
		 limit $4
	`, postID, app, category, relatedLimit)
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

// attachBylines gives every listed post the attribution it went public with.
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
	var appID *uuid.UUID
	var appSlug, appName, appHome *string
	var appPosition *int
	var appRetired *bool
	var markID *uuid.UUID
	var markWidth, markHeight *int
	var version *string
	err := row.Scan(
		&one.ID, &one.Slug, &one.Title, &one.Summary,
		&one.Category.ID, &one.Category.Slug, &one.Category.Label,
		&one.Category.Position, &one.Category.Retired,
		&appID, &appSlug, &appName, &appHome, &appPosition, &appRetired,
		&markID, &markWidth, &markHeight, &version,
		&one.PublishedAt, &one.UpdatedAt,
	)
	if err != nil {
		return PostSummary{}, fmt.Errorf("read a published archive entry: %w", err)
	}
	if appID != nil {
		one.App = &App{
			ID: *appID, Slug: *appSlug, Name: *appName, Home: *appHome,
			Mark:     scanMark(markID, markWidth, markHeight),
			Position: *appPosition, Retired: *appRetired,
		}
	}
	if version != nil {
		one.ReleaseVersion = *version
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
		return Category{}, fmt.Errorf("read a publication category: %w", err)
	}
	return found, nil
}

func (s *Service) appBySlug(ctx context.Context, slug string) (App, error) {
	var found App
	var markID *uuid.UUID
	var width, height *int
	err := s.pool.QueryRow(ctx, selectApps+` where app.slug = $1`, slug).Scan(
		&found.ID, &found.Slug, &found.Name, &found.Home, &found.Position,
		&found.Retired, &markID, &width, &height,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return App{}, ErrAppNotFound
	}
	if err != nil {
		return App{}, fmt.Errorf("read a publication app: %w", err)
	}
	found.Mark = scanMark(markID, width, height)
	return found, nil
}
