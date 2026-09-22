package blog

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	postbody "github.com/Sillyfrogster/Illarin/api/internal/blog/body"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const ReleaseCategory = "release"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

const (
	titleLimit      = 160
	summaryLimit    = 320
	headerTextLimit = 300
)

var (
	ErrPostNotFound  = errors.New("no such post")
	ErrNotPostEditor = errors.New("the account may not manage that post")
	ErrSlugLocked    = errors.New("the address of a published post is fixed")
)

type Stale struct {
	Version   int
	UpdatedAt time.Time
}

func (Stale) Error() string { return "the drafted changes have already moved on" }

type Editor struct {
	ID    uuid.UUID
	Admin bool
}

type Author struct {
	ID     uuid.UUID
	Handle string
}

type Post struct {
	ID              uuid.UUID
	Author          Author
	Category        Category
	Status          string
	Slug            string
	Title           string
	Summary         string
	Body            json.RawMessage
	BodyVersion     int
	Header          *Header
	LinkCardMediaID *uuid.UUID
	PublicRevision  *uuid.UUID
	Schedule        *Schedule
	Unpublishing    *Unpublishing
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

type PostEdit struct {
	CategoryID uuid.UUID
	Title      string
}

type PostSave struct {
	Version         int
	CategoryID      uuid.UUID
	Title           string
	Summary         string
	Slug            string
	Body            []byte
	Header          *HeaderEdit
	LinkCardMediaID *uuid.UUID
}

type HeaderEdit struct {
	MediaID uuid.UUID
	Alt     string
	Caption string
}

func (s *Service) Posts(ctx context.Context, editor Editor) ([]Post, error) {
	return s.postsFor(ctx, editor,
		`post.deleted_at is null`, `order by post.created_at desc`)
}

func (s *Service) DeletedPosts(ctx context.Context, editor Editor) ([]Post, error) {
	return s.postsFor(ctx, editor,
		`post.deleted_at is not null`, `order by post.recoverable_until`)
}

func (s *Service) postsFor(
	ctx context.Context,
	editor Editor,
	standingClause, orderClause string,
) ([]Post, error) {
	if editor.Admin {
		return s.postsWhere(ctx, `where `+standingClause+` `+orderClause)
	}
	writer, err := s.IsWriter(ctx, editor.ID)
	if err != nil {
		return nil, err
	}
	if !writer {
		return nil, ErrNotPostEditor
	}
	return s.postsWhere(ctx, `
		where `+standingClause+` and post.author_id = $1
	`+orderClause, editor.ID)
}

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

func (s *Service) CreatePost(ctx context.Context, editor Editor, in PostEdit) (Post, error) {
	category, err := s.category(ctx, in.CategoryID)
	if err != nil {
		return Post{}, err
	}
	if category.Retired {
		return Post{}, FieldError{Field: "categoryId", Message: "That category has been retired."}
	}
	if err := s.mayWrite(ctx, editor); err != nil {
		return Post{}, err
	}
	title, err := checkTitle(in.Title)
	if err != nil {
		return Post{}, err
	}
	empty, err := json.Marshal(postbody.Document{})
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
		insert into posts (id, author_id, category_id, title, slug, body, body_version)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, id, editor.ID, category.ID, title,
		freeSlug(ctx, tx, normalizeSlug(title)), empty, postbody.Version)
	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	err = recordActivity(ctx, tx, change{
		Actor: editor.ID, Action: "post.created",
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
	edition, err := s.checkDraftedChanges(ctx, editor, current, in)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin saving the drafted changes: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		update posts
		   set category_id = $3, title = $4, summary = $5, slug = $6,
		       body = $7, body_version = $8,
		       header_media_id = $9, header_alt = $10, header_caption = $11,
		       link_card_media_id = $12,
		       working_version = working_version + 1, updated_at = now()
		 where id = $1 and working_version = $2
	`, id, in.Version, edition.categoryID, edition.title, edition.summary,
		nullable(edition.slug), edition.body, postbody.Version,
		headerMediaID(edition.pictures.header), headerAlt(edition.pictures.header),
		headerCaption(edition.pictures.header), edition.pictures.social)
	if isUniqueViolation(err) {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err != nil {
		return Post{}, fmt.Errorf("save the drafted changes: %w", err)
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
		return Post{}, fmt.Errorf("commit the drafted changes: %w", err)
	}
	return s.post(ctx, id)
}

type PostImport struct {
	Version  int
	Markdown string
}

func (s *Service) ImportPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	in PostImport,
) (Post, []postbody.Note, error) {
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
	document, notes, err := postbody.FromMarkdown(in.Markdown)
	if err != nil {
		return Post{}, nil, importRefusal(err)
	}
	saved, err := s.SavePost(ctx, editor, id, current.carrying(document))
	if err != nil {
		return Post{}, nil, err
	}
	return saved, notes, nil
}

func importRefusal(err error) error {
	var problem postbody.Problem
	if errors.As(err, &problem) {
		return FieldError{Field: "markdown", Message: problem.Message}
	}
	return err
}

func (p Post) carrying(body []byte) PostSave {
	save := PostSave{
		Version:         p.Version,
		CategoryID:      p.Category.ID,
		Title:           p.Title,
		Summary:         p.Summary,
		Slug:            p.Slug,
		Body:            body,
		LinkCardMediaID: p.LinkCardMediaID,
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

type edition struct {
	categoryID uuid.UUID
	title      string
	summary    string
	slug       string
	body       []byte
	pictures   placed
}

func (s *Service) checkDraftedChanges(
	ctx context.Context,
	editor Editor,
	current Post,
	in PostSave,
) (edition, error) {
	category, err := s.category(ctx, in.CategoryID)
	if err != nil {
		return edition{}, err
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
	read, err := postbody.Read(in.Body)
	if err != nil {
		return edition{}, bodyRefusal(err)
	}
	body, err := json.Marshal(read)
	if err != nil {
		return edition{}, fmt.Errorf("write the post body: %w", err)
	}
	pictures, err := s.checkPlacement(ctx, current.ID, read, in)
	if err != nil {
		return edition{}, err
	}
	return edition{
		categoryID: category.ID, title: title, summary: summary, slug: slug,
		body: body, pictures: pictures,
	}, nil
}

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

func (s *Service) mayWrite(ctx context.Context, editor Editor) error {
	if editor.Admin {
		return nil
	}
	writer, err := s.IsWriter(ctx, editor.ID)
	if err != nil {
		return err
	}
	if !writer {
		return ErrNotPostEditor
	}
	return nil
}

func (s *Service) mayManage(ctx context.Context, editor Editor, found Post) error {
	if !editor.Admin && found.Author.ID != editor.ID {
		return ErrNotPostEditor
	}
	return s.mayWrite(ctx, editor)
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

func bodyRefusal(err error) error {
	var problem postbody.Problem
	if errors.As(err, &problem) {
		return FieldError{Field: problem.Path, Message: problem.Message}
	}
	return err
}

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
	if err := s.attachUnpublishings(ctx, found); err != nil {
		return nil, err
	}
	return found, nil
}

func (s *Service) attachUnpublishings(ctx context.Context, posts []Post) error {
	ids := make([]uuid.UUID, 0, len(posts))
	for index := range posts {
		if posts[index].Status == StatusUnpublished {
			ids = append(ids, posts[index].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	latest, err := s.unpublishingsFor(ctx, ids)
	if err != nil {
		return err
	}
	for index := range posts {
		if found, held := latest[posts[index].ID]; held {
			unpublishing := found
			posts[index].Unpublishing = &unpublishing
		}
	}
	return nil
}

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
	select post.id, author.id, author.username,
	       category.id, category.slug, category.label, category.position,
	       category.retired_at is not null,
	       post.status, post.slug, post.title, post.summary,
	       post.body, post.body_version,
	       post.header_media_id, post.header_alt, post.header_caption,
	       post.link_card_media_id, post.public_revision_id,
	       post.working_version, post.published_at, post.updated_public_at,
	       post.deleted_at, post.recoverable_until, remover.username,
	       post.created_at, post.updated_at
	  from posts post
	  join users author on author.id = post.author_id
	  left join users remover on remover.id = post.deleted_by
	  join blog_categories category on category.id = post.category_id
	`

func scanPost(rows pgx.Rows) (Post, error) {
	var one Post
	var slug *string
	var headerID *uuid.UUID
	var headerAltText, headerCaptionText *string
	var deletedAt, recoverableUntil *time.Time
	var remover *string
	err := rows.Scan(
		&one.ID, &one.Author.ID, &one.Author.Handle,
		&one.Category.ID, &one.Category.Slug, &one.Category.Label,
		&one.Category.Position, &one.Category.Retired,
		&one.Status, &slug, &one.Title, &one.Summary,
		&one.Body, &one.BodyVersion,
		&headerID, &headerAltText, &headerCaptionText, &one.LinkCardMediaID,
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
	return one, nil
}

const (
	PurposeHeader   = "header"
	PurposeBody     = "body"
	PurposeLinkCard = "link_card"
)

const (
	shownSize    = "detail"
	gallerySize  = "grid"
	linkCardSize = "og"
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
	size := shownSize
	if purpose == PurposeLinkCard {
		size = linkCardSize
	}
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, size, version)
}

func PostMediaThumbURL(mediaID uuid.UUID, version uint32) string {
	return fmt.Sprintf("/media/%s/%s/%d", mediaID, gallerySize, version)
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
	if purpose != PurposeHeader && purpose != PurposeBody && purpose != PurposeLinkCard {
		return PostMedia{}, FieldError{
			Field:   "purpose",
			Message: "A picture is uploaded as a header, a body picture or a link card.",
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

func (s *Service) PostImageSize(
	ctx context.Context,
	mediaID uuid.UUID,
	size string,
	version uint32,
	expires, signature string,
) (string, string, bool, error) {
	_, ordinary := mediaproc.ImageSizeByName(size)
	_, composed := mediaproc.LinkCardByName(size)
	if (!ordinary && !composed) || version != mediaproc.ImageSizeVersion {
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
		                where use.media_id = media.id and revision.captured_for = $2
		                  and exists (select 1 from posts where id = media.post_id and deleted_at is null))
		  from post_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, mediaID, RevisionPublish).Scan(&blobID, &digestBytes, &published)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, ErrPostMediaNotFound
	}
	if err != nil {
		return "", "", false, fmt.Errorf("find post media: %w", err)
	}
	if !published {
		path := fmt.Sprintf("/media/%s/%s/%d", mediaID, size, version)
		if !s.signer.Valid(path, expires, signature, s.now()) {
			return "", "", false, ErrPostMediaNotFound
		}
	}
	if len(digestBytes) != sha256.Size {
		return "", "", false, fmt.Errorf("post media blob has a %d-byte digest", len(digestBytes))
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	redirect, err := s.media.Serve(ctx, blobID, digest, size, version)
	if err != nil {
		return "", "", false, err
	}
	return redirect, s.media.ImageSizeMediaType(), !published, nil
}

type placed struct {
	header  *Header
	social  *uuid.UUID
	ordered []uuid.UUID
}

func (s *Service) checkPlacement(
	ctx context.Context,
	postID uuid.UUID,
	body postbody.Document,
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
		if err := belongs(owned, id, PurposeBody, "body"); err != nil {
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
	if in.LinkCardMediaID != nil {
		if err := belongs(owned, *in.LinkCardMediaID, PurposeLinkCard, "linkCardMediaId"); err != nil {
			return placed{}, err
		}
		chosen.social = in.LinkCardMediaID
		chosen.ordered = append(chosen.ordered, *in.LinkCardMediaID)
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
			return fmt.Errorf("clear what the drafted changes refer to: %w", err)
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
