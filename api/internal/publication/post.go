package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReleaseCategory is the one category whose posts carry release metadata.
const ReleaseCategory = "release"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

const (
	titleLimit          = 160
	summaryLimit        = 320
	releaseVersionLimit = 40
	headerTextLimit     = 300
)

var (
	ErrPostNotFound  = errors.New("no such post")
	ErrNotPostEditor = errors.New("the account may not manage that post")
	ErrSlugLocked    = errors.New("the address of a published post is fixed")
)

// Stale says the working copy moved on since the editor last read it.
type Stale struct {
	Version   int
	UpdatedAt time.Time
}

func (Stale) Error() string { return "the working copy has already moved on" }

// Editor is the credential acting on a post and how far it may reach. A session
// leaves Grant empty. A publication token fills it in and can never reach past
// the one grant it was issued under.
type Editor struct {
	ID    uuid.UUID
	Admin bool
	Grant *uuid.UUID
	Token *uuid.UUID
}

// Credential names the kind of credential the editor is acting through.
func (e Editor) Credential() string {
	if e.Token != nil {
		return CredentialToken
	}
	return CredentialSession
}

// writesAs answers the grant a new post belongs to, refusing a credential
// bound to one grant that named another.
func (e Editor) writesAs(named *uuid.UUID) (*uuid.UUID, error) {
	if e.Grant == nil {
		return named, nil
	}
	if named != nil && *named != *e.Grant {
		return nil, ErrNotPostEditor
	}
	return e.Grant, nil
}

// Author names the account a post belongs to.
type Author struct {
	ID     uuid.UUID
	Handle string
}

// Release is the metadata a release post carries and no other category does.
type Release struct {
	App     App
	Version string
	Address string
}

// Post is one entry's working copy together with what its public state adds.
type Post struct {
	ID              uuid.UUID
	Author          Author
	GrantID         *uuid.UUID
	App             *App
	Category        Category
	Status          string
	Slug            string
	Title           string
	Summary         string
	Document        json.RawMessage
	DocumentVersion int
	Release         *Release
	Header          *Header
	SocialMediaID   *uuid.UUID
	PublicRevision  *uuid.UUID
	Schedule        *Schedule
	Withdrawal      *Withdrawal
	Deletion        *Deletion
	Media           []PostMedia
	Byline          *Byline
	FormerAddresses []string
	Version         int
	PublishedAt     *time.Time
	UpdatedPublicAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PostEdit is what starting a post needs.
type PostEdit struct {
	GrantID    *uuid.UUID
	CategoryID uuid.UUID
	Title      string
}

// PostSave is the whole working copy an autosave replaces.
type PostSave struct {
	Version       int
	CategoryID    uuid.UUID
	Title         string
	Summary       string
	Slug          string
	Document      []byte
	Release       *ReleaseEdit
	Header        *HeaderEdit
	SocialMediaID *uuid.UUID
}

// HeaderEdit is the picture an article opens with as a request supplied it.
type HeaderEdit struct {
	MediaID uuid.UUID
	Alt     string
	Caption string
}

// ReleaseEdit is the release metadata a request supplied.
type ReleaseEdit struct {
	AppID   uuid.UUID
	Version string
	Address string
}

// Posts answers every post the editor may manage and has not deleted, newest first.
func (s *Service) Posts(ctx context.Context, editor Editor) ([]Post, error) {
	return s.postsFor(ctx, editor,
		`post.deleted_at is null`, `order by post.created_at desc`)
}

// DeletedPosts answers the posts the editor may still recover, the ones
// closest to their deadline first.
func (s *Service) DeletedPosts(ctx context.Context, editor Editor) ([]Post, error) {
	return s.postsFor(ctx, editor,
		`post.deleted_at is not null`, `order by post.recoverable_until`)
}

// postsFor answers one standing of the posts this editor reaches. A token
// reaches its own grant, an admin reaches everything, and anybody else reaches
// the posts under the approvals they hold.
func (s *Service) postsFor(
	ctx context.Context,
	editor Editor,
	standing, order string,
) ([]Post, error) {
	if editor.Grant != nil {
		return s.postsWhere(ctx, `
			where `+standing+` and post.grant_id = $1 and grant_row.active
		`+order, *editor.Grant)
	}
	if editor.Admin {
		return s.postsWhere(ctx, `where `+standing+` `+order)
	}
	return s.postsWhere(ctx, `
		where `+standing+` and grant_row.user_id = $1 and grant_row.active
	`+order, editor.ID)
}

// Post answers one working copy to the account allowed to manage it.
func (s *Service) Post(ctx context.Context, editor Editor, id uuid.UUID) (Post, error) {
	found, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, found); err != nil {
		return Post{}, err
	}
	return found, nil
}

// CreatePost starts a draft under the identity its author is allowed to use.
func (s *Service) CreatePost(ctx context.Context, editor Editor, in PostEdit) (Post, error) {
	category, err := s.category(ctx, in.CategoryID)
	if err != nil {
		return Post{}, err
	}
	if category.Retired {
		return Post{}, FieldError{Field: "categoryId", Message: "That category has been retired."}
	}
	grantID, err := editor.writesAs(in.GrantID)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayWriteAs(ctx, editor, grantID, category); err != nil {
		return Post{}, err
	}
	title, err := checkTitle(in.Title)
	if err != nil {
		return Post{}, err
	}
	empty, err := json.Marshal(postdoc.Document{})
	if err != nil {
		return Post{}, fmt.Errorf("write an empty post body: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin post: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into posts (id, author_id, grant_id, category_id, title, slug,
		                   document, document_version)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, editor.ID, grantID, category.ID, title,
		freeSlug(ctx, tx, normalizeSlug(title)), empty, postdoc.Version)
	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.created",
		GrantID: grantID, TokenID: editor.Token,
		CategoryID: &category.ID, PostID: &id, After: StatusDraft,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit post: %w", err)
	}
	return s.post(ctx, id)
}

// SavePost replaces the working copy, refusing an editor that began from an
// edition someone has already moved past.
func (s *Service) SavePost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	in PostSave,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if current.Deletion != nil {
		return Post{}, ErrPostDeleted
	}
	if in.Version != current.Version {
		return Post{}, Stale{Version: current.Version, UpdatedAt: current.UpdatedAt}
	}
	edition, err := s.checkWorkingCopy(ctx, editor, current, in)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin working copy save: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		update posts
		   set category_id = $3, title = $4, summary = $5, slug = $6,
		       document = $7, document_version = $8,
		       release_app_id = $9, release_version = $10, release_url = $11,
		       header_media_id = $12, header_alt = $13, header_caption = $14,
		       social_media_id = $15,
		       working_version = working_version + 1, updated_at = now()
		 where id = $1 and working_version = $2
	`, id, in.Version, edition.categoryID, edition.title, edition.summary,
		nullable(edition.slug), edition.document, postdoc.Version,
		edition.releaseAppID, nullable(edition.releaseVersion), nullable(edition.releaseAddress),
		headerMediaID(edition.pictures.header), headerAlt(edition.pictures.header),
		headerCaption(edition.pictures.header), edition.pictures.social)
	if isUniqueViolation(err) {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err != nil {
		return Post{}, fmt.Errorf("save working copy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		moved, readErr := s.post(ctx, id)
		if readErr != nil {
			return Post{}, readErr
		}
		return Post{}, Stale{Version: moved.Version, UpdatedAt: moved.UpdatedAt}
	}
	if err := recordUses(ctx, tx, id, nil, edition.pictures.ordered); err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit working copy save: %w", err)
	}
	return s.post(ctx, id)
}

// PostImport is the Markdown to convert and the working copy it replaces.
type PostImport struct {
	Version  int
	Markdown string
}

// ImportPost converts constrained Markdown into the working copy's document,
// leaving everything else about the post as it stands.
func (s *Service) ImportPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	in PostImport,
) (Post, []postdoc.Note, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, nil, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, nil, err
	}
	if in.Version != current.Version {
		return Post{}, nil, Stale{Version: current.Version, UpdatedAt: current.UpdatedAt}
	}
	document, notes, err := postdoc.FromMarkdown(in.Markdown)
	if err != nil {
		return Post{}, nil, importRefusal(err)
	}
	saved, err := s.SavePost(ctx, editor, id, current.carrying(document))
	if err != nil {
		return Post{}, nil, err
	}
	return saved, notes, nil
}

// importRefusal names the field the caller sent, not a path inside the document
// the conversion was building.
func importRefusal(err error) error {
	var problem postdoc.Problem
	if errors.As(err, &problem) {
		return FieldError{Field: "markdown", Message: problem.Message}
	}
	return err
}

// carrying answers this working copy as a save with a different document in it.
func (p Post) carrying(document []byte) PostSave {
	save := PostSave{
		Version:       p.Version,
		CategoryID:    p.Category.ID,
		Title:         p.Title,
		Summary:       p.Summary,
		Slug:          p.Slug,
		Document:      document,
		SocialMediaID: p.SocialMediaID,
	}
	if p.Release != nil {
		save.Release = &ReleaseEdit{
			AppID:   p.Release.App.ID,
			Version: p.Release.Version,
			Address: p.Release.Address,
		}
	}
	if p.Header != nil {
		save.Header = &HeaderEdit{
			MediaID: p.Header.MediaID,
			Alt:     p.Header.Alt,
			Caption: p.Header.Caption,
		}
	}
	return save
}

// edition is one checked working copy on its way into the database.
type edition struct {
	categoryID     uuid.UUID
	title          string
	summary        string
	slug           string
	document       []byte
	releaseAppID   *uuid.UUID
	releaseVersion string
	releaseAddress string
	pictures       placed
}

func (s *Service) checkWorkingCopy(
	ctx context.Context,
	editor Editor,
	current Post,
	in PostSave,
) (edition, error) {
	category, err := s.category(ctx, in.CategoryID)
	if err != nil {
		return edition{}, err
	}
	if category.ID != current.Category.ID {
		if err := s.mayWriteAs(ctx, editor, current.GrantID, category); err != nil {
			return edition{}, err
		}
	}
	title, err := checkTitle(in.Title)
	if err != nil {
		return edition{}, err
	}
	summary := strings.TrimSpace(in.Summary)
	if len(summary) > summaryLimit {
		return edition{}, FieldError{
			Field:   "summary",
			Message: fmt.Sprintf("Keep the summary under %d characters.", summaryLimit),
		}
	}
	slug := normalizeSlug(in.Slug)
	if current.PublishedAt != nil && slug != current.Slug {
		return edition{}, ErrSlugLocked
	}
	if slug != "" {
		if slug, err = checkSlug(slug); err != nil {
			return edition{}, err
		}
		if err := s.refuseTakenAddress(ctx, current.ID, slug); err != nil {
			return edition{}, err
		}
	}
	body, err := postdoc.Read(in.Document)
	if err != nil {
		return edition{}, documentRefusal(err)
	}
	document, err := json.Marshal(body)
	if err != nil {
		return edition{}, fmt.Errorf("write the post body: %w", err)
	}
	release, err := s.checkRelease(ctx, current, category, in.Release)
	if err != nil {
		return edition{}, err
	}
	pictures, err := s.checkPlacement(ctx, current.ID, body, in)
	if err != nil {
		return edition{}, err
	}
	return edition{
		categoryID: category.ID, title: title, summary: summary, slug: slug,
		document: document, releaseAppID: release.appID,
		releaseVersion: release.version, releaseAddress: release.address,
		pictures: pictures,
	}, nil
}

// refuseTakenAddress keeps a post off an address another post holds or ever held.
func (s *Service) refuseTakenAddress(ctx context.Context, postID uuid.UUID, slug string) error {
	taken, err := addressTaken(ctx, s.pool, postID, slug)
	if err != nil {
		return err
	}
	if taken {
		return FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	return nil
}

// releaseFields is the release metadata after it has been checked.
type releaseFields struct {
	appID   *uuid.UUID
	version string
	address string
}

func (s *Service) checkRelease(
	ctx context.Context,
	current Post,
	category Category,
	in *ReleaseEdit,
) (releaseFields, error) {
	if in == nil {
		if category.Slug == ReleaseCategory {
			return releaseFields{}, FieldError{
				Field:   "release",
				Message: "A release names the project it belongs to and its version.",
			}
		}
		return releaseFields{}, nil
	}
	app, err := s.app(ctx, in.AppID)
	if err != nil {
		return releaseFields{}, err
	}
	if current.App != nil && app.ID != current.App.ID {
		return releaseFields{}, FieldError{
			Field:   "release.appId",
			Message: "Your approval covers " + current.App.Name + " only.",
		}
	}
	version := strings.TrimSpace(in.Version)
	if version == "" || len(version) > releaseVersionLimit {
		return releaseFields{}, FieldError{
			Field:   "release.version",
			Message: fmt.Sprintf("Give the release a version of up to %d characters.", releaseVersionLimit),
		}
	}
	address := strings.TrimSpace(in.Address)
	if address != "" && (!strings.HasPrefix(address, "https://") || len(address) > addressLimit) {
		return releaseFields{}, FieldError{
			Field:   "release.address",
			Message: "A release address is an https address.",
		}
	}
	return releaseFields{appID: &app.ID, version: version, address: address}, nil
}

// mayWriteAs answers whether this editor may publish under this grant and
// category. An admin writes as Illarin unless they name a grant of their own.
func (s *Service) mayWriteAs(
	ctx context.Context,
	editor Editor,
	grantID *uuid.UUID,
	category Category,
) error {
	if grantID == nil {
		if editor.Admin {
			return nil
		}
		return ErrNotPostEditor
	}
	held, err := s.grant(ctx, *grantID)
	if err != nil {
		return err
	}
	if held.Holder.ID != editor.ID || !held.Active {
		return ErrNotPostEditor
	}
	for _, allowed := range held.Categories {
		if allowed.ID == category.ID {
			return nil
		}
	}
	return FieldError{
		Field:   "categoryId",
		Message: "Your approval does not cover " + category.Label + " posts.",
		cause:   ErrCategoryRefused,
	}
}

func (s *Service) mayManage(ctx context.Context, editor Editor, found Post) error {
	if editor.Grant != nil {
		if found.GrantID == nil || *found.GrantID != *editor.Grant {
			return ErrNotPostEditor
		}
		return nil
	}
	if editor.Admin {
		return nil
	}
	if found.GrantID == nil {
		return ErrNotPostEditor
	}
	held, err := s.grant(ctx, *found.GrantID)
	if err != nil {
		return err
	}
	if held.Holder.ID != editor.ID || !held.Active {
		return ErrNotPostEditor
	}
	return nil
}

func checkTitle(candidate string) (string, error) {
	title := strings.TrimSpace(candidate)
	if title == "" || len(title) > titleLimit {
		return "", FieldError{
			Field:   "title",
			Message: fmt.Sprintf("Give the post a title of up to %d characters.", titleLimit),
		}
	}
	return title, nil
}

// documentRefusal turns a document problem into the field refusal a client reads.
func documentRefusal(err error) error {
	var problem postdoc.Problem
	if errors.As(err, &problem) {
		return FieldError{Field: problem.Path, Message: problem.Message}
	}
	return err
}

// freeSlug answers the first unused address near the candidate, so starting two
// posts with the same title does not fail before either has been written.
func freeSlug(ctx context.Context, tx pgx.Tx, candidate string) *string {
	if candidate == "" || reservedSlugs[candidate] {
		return nil
	}
	for attempt := range 20 {
		trying := candidate
		if attempt > 0 {
			trying = fmt.Sprintf("%s-%d", candidate, attempt+1)
		}
		taken, err := addressTaken(ctx, tx, uuid.Nil, trying)
		if err != nil {
			return nil
		}
		if !taken {
			return &trying
		}
	}
	return nil
}

func headerMediaID(header *Header) *uuid.UUID {
	if header == nil {
		return nil
	}
	return &header.MediaID
}

func headerAlt(header *Header) *string {
	if header == nil {
		return nil
	}
	return &header.Alt
}

func headerCaption(header *Header) *string {
	if header == nil || header.Caption == "" {
		return nil
	}
	return &header.Caption
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *Service) post(ctx context.Context, id uuid.UUID) (Post, error) {
	found, err := s.postsWhere(ctx, `where post.id = $1`, id)
	if err != nil {
		return Post{}, err
	}
	if len(found) == 0 {
		return Post{}, ErrPostNotFound
	}
	return found[0], nil
}

func scanDeletion(at, until *time.Time, by *string) *Deletion {
	if at == nil || until == nil {
		return nil
	}
	removed := &Deletion{At: *at, Until: *until}
	if by != nil {
		removed.By = *by
	}
	return removed
}

func scanHeader(mediaID *uuid.UUID, alt, caption *string) *Header {
	if mediaID == nil {
		return nil
	}
	header := &Header{MediaID: *mediaID}
	if alt != nil {
		header.Alt = *alt
	}
	if caption != nil {
		header.Caption = *caption
	}
	return header
}

func (s *Service) postsWhere(ctx context.Context, clause string, args ...any) ([]Post, error) {
	rows, err := s.pool.Query(ctx, selectPosts+clause, args...)
	if err != nil {
		return nil, fmt.Errorf("read posts: %w", err)
	}
	defer rows.Close()
	found := make([]Post, 0, 8)
	for rows.Next() {
		one, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read posts: %w", err)
	}
	if err := s.attachWorkingMedia(ctx, found); err != nil {
		return nil, err
	}
	if err := s.attachAttribution(ctx, found); err != nil {
		return nil, err
	}
	if err := s.attachSchedules(ctx, found); err != nil {
		return nil, err
	}
	if err := s.attachWithdrawals(ctx, found); err != nil {
		return nil, err
	}
	return found, nil
}

// attachWithdrawals tells each post still out of public view why it came down.
func (s *Service) attachWithdrawals(ctx context.Context, posts []Post) error {
	ids := make([]uuid.UUID, 0, len(posts))
	for index := range posts {
		if posts[index].Status == StatusWithdrawn {
			ids = append(ids, posts[index].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	latest, err := s.withdrawalsFor(ctx, ids)
	if err != nil {
		return err
	}
	for index := range posts {
		if found, held := latest[posts[index].ID]; held {
			withdrawal := found
			posts[index].Withdrawal = &withdrawal
		}
	}
	return nil
}

// attachAttribution gives each post the byline it carries and the addresses it has left behind.
func (s *Service) attachAttribution(ctx context.Context, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(posts))
	for index := range posts {
		posts[index].FormerAddresses = []string{}
		ids = append(ids, posts[index].ID)
	}
	bylines, err := bylinesFor(ctx, s.pool, ids)
	if err != nil {
		return err
	}
	former, err := s.formerAddresses(ctx, ids)
	if err != nil {
		return err
	}
	for index := range posts {
		if carried, held := bylines[posts[index].ID]; held {
			byline := carried
			posts[index].Byline = &byline
		}
		if left, held := former[posts[index].ID]; held {
			posts[index].FormerAddresses = left
		}
	}
	return nil
}

// formerAddresses answers the addresses each post published under and has since left, newest first.
func (s *Service) formerAddresses(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID][]string, error) {
	rows, err := s.pool.Query(ctx, `
		select reservation.post_id, reservation.slug
		  from post_slugs reservation
		  join posts post on post.id = reservation.post_id
		 where reservation.post_id = any($1) and reservation.slug is distinct from post.slug
		 order by reservation.reserved_at desc
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("read the addresses posts have left: %w", err)
	}
	defer rows.Close()
	left := make(map[uuid.UUID][]string, len(ids))
	for rows.Next() {
		var postID uuid.UUID
		var slug string
		if err := rows.Scan(&postID, &slug); err != nil {
			return nil, fmt.Errorf("read an address a post has left: %w", err)
		}
		left[postID] = append(left[postID], slug)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the addresses posts have left: %w", err)
	}
	return left, nil
}

// attachWorkingMedia gives each working copy the pictures it refers to.
func (s *Service) attachWorkingMedia(ctx context.Context, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(posts))
	for index := range posts {
		posts[index].Media = []PostMedia{}
		ids = append(ids, posts[index].ID)
	}
	rows, err := s.pool.Query(ctx, `
		select use.post_id, media.id, media.post_id, media.purpose, media.width, media.height
		  from post_media_uses use
		  join post_media media on media.id = use.media_id
		 where use.post_id = any($1) and use.revision_id is null and media.blob_id is not null
		 order by media.created_at
	`, ids)
	if err != nil {
		return fmt.Errorf("read the pictures posts refer to: %w", err)
	}
	defer rows.Close()
	held := make(map[uuid.UUID][]PostMedia, len(posts))
	for rows.Next() {
		var postID uuid.UUID
		var one PostMedia
		err := rows.Scan(&postID, &one.ID, &one.PostID, &one.Purpose, &one.Width, &one.Height)
		if err != nil {
			return fmt.Errorf("read a picture a post refers to: %w", err)
		}
		held[postID] = append(held[postID], one)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read the pictures posts refer to: %w", err)
	}
	for index := range posts {
		if pictures, carried := held[posts[index].ID]; carried {
			posts[index].Media = pictures
		}
	}
	return nil
}

const selectPosts = `
	select post.id, author.id, author.username, post.grant_id,
	       app.id, app.slug, app.name, app.home_url, app.position,
	       app.retired_at is not null, mark.id, mark.width, mark.height,
	       category.id, category.slug, category.label, category.position,
	       category.retired_at is not null,
	       post.status, post.slug, post.title, post.summary,
	       post.document, post.document_version,
	       release_app.id, release_app.slug, release_app.name, release_app.home_url,
	       release_app.position, release_app.retired_at is not null,
	       release_mark.id, release_mark.width, release_mark.height,
	       post.release_version, post.release_url,
	       post.header_media_id, post.header_alt, post.header_caption,
	       post.social_media_id, post.public_revision_id,
	       post.working_version, post.published_at, post.updated_public_at,
	       post.deleted_at, post.recoverable_until, remover.username,
	       post.created_at, post.updated_at
	  from posts post
	  join users author on author.id = post.author_id
	  left join users remover on remover.id = post.deleted_by
	  join publication_categories category on category.id = post.category_id
	  left join publication_grants grant_row on grant_row.id = post.grant_id
	  left join publication_apps app on app.id = grant_row.app_id
	  left join publication_media mark
	         on mark.id = app.mark_media_id and mark.blob_id is not null
	  left join publication_apps release_app on release_app.id = post.release_app_id
	  left join publication_media release_mark
	         on release_mark.id = release_app.mark_media_id and release_mark.blob_id is not null
	`

func scanPost(rows pgx.Rows) (Post, error) {
	var one Post
	var app App
	var appID, markID, releaseID, releaseMarkID *uuid.UUID
	var markWidth, markHeight, releaseMarkWidth, releaseMarkHeight *int
	var appSlug, appName, appHome *string
	var appPosition *int
	var appRetired *bool
	var release App
	var releaseSlug, releaseName, releaseHome, releaseVersion, releaseAddress, slug *string
	var headerID *uuid.UUID
	var headerAltText, headerCaptionText *string
	var releasePosition *int
	var releaseRetired *bool
	var deletedAt, recoverableUntil *time.Time
	var remover *string
	err := rows.Scan(
		&one.ID, &one.Author.ID, &one.Author.Handle, &one.GrantID,
		&appID, &appSlug, &appName, &appHome, &appPosition, &appRetired,
		&markID, &markWidth, &markHeight,
		&one.Category.ID, &one.Category.Slug, &one.Category.Label,
		&one.Category.Position, &one.Category.Retired,
		&one.Status, &slug, &one.Title, &one.Summary,
		&one.Document, &one.DocumentVersion,
		&releaseID, &releaseSlug, &releaseName, &releaseHome,
		&releasePosition, &releaseRetired,
		&releaseMarkID, &releaseMarkWidth, &releaseMarkHeight,
		&releaseVersion, &releaseAddress,
		&headerID, &headerAltText, &headerCaptionText, &one.SocialMediaID,
		&one.PublicRevision,
		&one.Version, &one.PublishedAt, &one.UpdatedPublicAt,
		&deletedAt, &recoverableUntil, &remover,
		&one.CreatedAt, &one.UpdatedAt,
	)
	if err != nil {
		return Post{}, fmt.Errorf("read a post: %w", err)
	}
	if slug != nil {
		one.Slug = *slug
	}
	one.Deletion = scanDeletion(deletedAt, recoverableUntil, remover)
	one.Header = scanHeader(headerID, headerAltText, headerCaptionText)
	if appID != nil {
		app = App{
			ID: *appID, Slug: *appSlug, Name: *appName, Home: *appHome,
			Position: *appPosition, Retired: *appRetired,
			Mark: scanMark(markID, markWidth, markHeight),
		}
		one.App = &app
	}
	if releaseID != nil {
		release = App{
			ID: *releaseID, Slug: *releaseSlug, Name: *releaseName, Home: *releaseHome,
			Position: *releasePosition, Retired: *releaseRetired,
			Mark: scanMark(releaseMarkID, releaseMarkWidth, releaseMarkHeight),
		}
		one.Release = &Release{App: release}
		if releaseVersion != nil {
			one.Release.Version = *releaseVersion
		}
		if releaseAddress != nil {
			one.Release.Address = *releaseAddress
		}
	}
	return one, nil
}
