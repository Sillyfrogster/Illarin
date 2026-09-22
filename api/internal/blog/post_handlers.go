package blog

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/profile"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListPosts(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListPostsParams{
		Deleted: api.QueryFlag(q, "deleted"),
	}
	if q.Refused(c) {
		return
	}
	editor, ok := h.postEditor(c, "reading posts")
	if !ok {
		return
	}
	held, err := h.listing(params)(c.Request.Context(), editor)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostList{Posts: h.toAPIPosts(held)})
}

func (h *Handlers) listing(params ListPostsParams) func(
	context.Context, Editor,
) ([]Post, error) {
	if params.Deleted != nil && *params.Deleted {
		return h.blog.DeletedPosts
	}
	return h.blog.Posts
}

func (h *Handlers) CreatePost(c *gin.Context) {
	editor, ok := h.postEditor(c, "starting a post")
	if !ok {
		return
	}
	var request CreatePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Send the post as JSON.")
		return
	}
	started, err := h.blog.CreatePost(c.Request.Context(), editor, PostEdit{
		CategoryID: request.CategoryId,
		Title:      request.Title,
	})
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toAPIPost(started))
}

func (h *Handlers) GetPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "reading a post")
	if !ok {
		return
	}
	found, err := h.blog.Post(c.Request.Context(), editor, id)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(found))
}

func (h *Handlers) SavePost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "saving a post")
	if !ok {
		return
	}
	var request SavePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Send the drafted changes as JSON.")
		return
	}
	body, err := json.Marshal(request.Body)
	if err != nil {
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Send the post body as JSON.")
		return
	}
	saved, err := h.blog.SavePost(c.Request.Context(), editor, id,
		PostSave{
			Version:         request.Version,
			CategoryID:      request.CategoryId,
			Title:           request.Title,
			Summary:         request.Summary,
			Slug:            request.Slug,
			Body:            body,
			Header:          toHeaderEdit(request.Header),
			LinkCardMediaID: optionalID(request.LinkCardMediaId),
		})
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(saved))
}

func (h *Handlers) AddPostMedia(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "adding a picture to a post")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refusePostMedia(c, api.FormRefusal{
			Reason: "send the picture as form data, with a metadata part and a file part",
			Cause:  err,
		})
		return
	}
	metadata, err := readPostMediaMetadata(parts)
	if err != nil {
		h.refusePostMedia(c, err)
		return
	}
	file, err := api.NextPart(parts, api.FilePart)
	if err != nil {
		h.refusePostMedia(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	added, err := h.blog.AddPostMedia(
		c.Request.Context(), editor, id, string(metadata.Purpose), limitedFile,
	)
	var refused FieldError
	switch {
	case errors.Is(err, ErrPostNotFound), errors.Is(err, ErrNotPostEditor):
		h.postError(c, err)
		return
	case errors.As(err, &refused):
		h.postError(c, err)
		return
	case err != nil:
		h.refusePostMedia(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIPostPicture(&added, h.blog.SignPrivate))
}

func (h *Handlers) PublishPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "publishing a post")
	if !ok {
		return
	}
	var request PublishPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"Include the current drafted changes version.", "version")
		return
	}
	published, err := h.blog.PublishPost(
		c.Request.Context(), editor, id, request.Version,
		announcementOf(request.IntegrationIds, request.RoleIntegrationIds, request.Note),
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(published))
}

func (h *Handlers) CorrectPostAddress(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "correcting a post address")
	if !ok {
		return
	}
	var request CorrectPostAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the address as JSON.")
		return
	}
	moved, err := h.blog.CorrectAddress(
		c.Request.Context(), editor, id, request.Slug,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(moved))
}

func (h *Handlers) CorrectPostByline(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "correcting a post byline")
	if !ok {
		return
	}
	var request CorrectPostBylineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the handle as JSON.")
		return
	}
	corrected, err := h.blog.CorrectByline(
		c.Request.Context(), editor, id, request.Handle,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(corrected))
}

func (h *Handlers) ListPostCategories(c *gin.Context) {
	found, err := h.blog.ReadableCategories(c.Request.Context())
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the categories.")
		return
	}
	c.JSON(http.StatusOK, BlogCategoryList{Categories: toAPICategories(found)})
}

func (h *Handlers) ListPublishedPosts(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListPublishedPostsParams{
		Page:     api.QueryNumber(q, "page"),
		Category: api.QueryText[string](q, "category"),
	}
	if q.Refused(c) {
		return
	}
	asked := ArchiveQuery{Page: 1}
	if params.Page != nil {
		asked.Page = *params.Page
	}
	if asked.Page < 1 {
		api.Refuse(c, http.StatusBadRequest, "Archive pages count from one.")
		return
	}
	if params.Category != nil {
		asked.Category = *params.Category
	}
	found, err := h.blog.Archive(c.Request.Context(), asked)
	switch {
	case errors.Is(err, ErrCategoryNotFound):
		api.Refuse(c, http.StatusNotFound, "No such blog category.")
		return
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not load blog posts. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIArchive(found))
}

func (h *Handlers) GetPublishedPost(c *gin.Context) {
	slug := c.Param("slug")
	found, err := h.blog.PublishedPost(c.Request.Context(), slug)
	if errors.Is(err, ErrPostNotFound) {
		if h.unpublishedPost(c, slug) {
			return
		}
		api.Refuse(c, http.StatusNotFound, "No such post.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the post.")
		return
	}
	c.JSON(http.StatusOK, toAPIPublicPost(found))
}

func (h *Handlers) refusePostMedia(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		refuseField(c, http.StatusRequestEntityTooLarge, BlogErrorCodeInvalid,
			"That picture is larger than the upload limit.", api.FilePart)
	case errors.Is(err, storage.ErrInsufficientSpace):
		refuseBlog(c, http.StatusServiceUnavailable, BlogErrorCodeServerError,
			"Uploads are temporarily unavailable because storage is low.")
	default:
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"That picture could not be read. Use a PNG, JPEG, WebP or GIF.", api.FilePart)
	}
}

func (h *Handlers) postEditor(c *gin.Context, action string) (Editor, bool) {
	current, ok := api.Verified(c, action)
	if !ok {
		return Editor{}, false
	}
	return Editor{
		ID:    current.ID,
		Admin: current.Role == api.RoleAdmin,
	}, true
}

func (h *Handlers) postError(c *gin.Context, err error) {
	var stale Stale
	switch {
	case errors.Is(err, ErrPostNotFound):
		refuseBlog(c, http.StatusNotFound, BlogErrorCodeNotFound, "No such post.")
	case errors.Is(err, ErrRevisionNotFound):
		refuseBlog(c, http.StatusNotFound, BlogErrorCodeNotFound,
			"This post has no such revision.")
	case errors.Is(err, ErrIntegrationRefused):
		refuseBlog(c, http.StatusForbidden, BlogErrorCodeForbidden,
			"This post may not send to that integration.")
	case errors.Is(err, ErrRoleRefused):
		refuseBlog(c, http.StatusForbidden, BlogErrorCodeForbidden,
			"This post may not mention that integration's role.")
	case errors.Is(err, ErrNotPostEditor):
		refuseBlog(c, http.StatusForbidden, BlogErrorCodeForbidden,
			"Only this post's contributor or an Illarin admin can do that.")
	case errors.Is(err, ErrSlugLocked):
		refuseField(c, http.StatusForbidden, BlogErrorCodeForbidden,
			"The address of a published post is fixed. An admin can correct it.", "slug")
	case errors.Is(err, ErrNotPostAdmin):
		refuseBlog(c, http.StatusForbidden, BlogErrorCodeForbidden,
			"Only an Illarin admin can correct a published post.")
	case errors.Is(err, ErrPostNotPublished):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"There is nothing to correct until the post is published.")
	case errors.Is(err, ErrPostNotPublic):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"Only a published post can be unpublished.")
	case errors.Is(err, ErrPostNotUnpublished):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"This post is not unpublished.")
	case errors.Is(err, ErrPostUnpublished):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"This post is unpublished. Republish it to make it public again.")
	case errors.Is(err, ErrPostDeleted):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"This post is deleted. Restore it before editing.")
	case errors.Is(err, ErrPostNotDeleted):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"This post has not been deleted.")
	case errors.Is(err, ErrRecoveryExpired):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"The recovery deadline has passed. This post cannot be restored.")
	case errors.Is(err, ErrSchedulePublishing):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This revision is being published and can no longer be changed.",
			Code:  BlogErrorCodeScheduleRunning,
		})
	case errors.As(err, &stale):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error:     "This post was saved in another session. Copy any unsaved text, then reload to edit the latest version.",
			Code:      BlogErrorCodeStaleVersion,
			Field:     pointer("version"),
			Version:   &stale.Version,
			UpdatedAt: &stale.UpdatedAt,
		})
	default:
		h.blogError(c, err)
	}
}

func (h *Handlers) toAPIPosts(held []Post) []PostResponse {
	listed := make([]PostResponse, 0, len(held))
	for _, one := range held {
		listed = append(listed, h.toAPIPost(one))
	}
	return listed
}

func (h *Handlers) toAPIPost(found Post) PostResponse {
	shown := PostResponse{
		Id:              found.ID,
		Status:          PostStatus(found.Status),
		Title:           found.Title,
		Summary:         found.Summary,
		Slug:            found.Slug,
		Category:        toAPICategory(found.Category),
		Body:            toAPIBody(found.Body),
		BodyVersion:     found.BodyVersion,
		Header:          toAPIHeader(found.Header),
		Media:           showPostMedia(found.Media, h.blog.SignPrivate),
		FormerAddresses: found.FormerAddresses,
		Version:         found.Version,
		Author:          PostAuthor{Handle: found.Author.Handle},
		PublishedAt:     found.PublishedAt,
		UpdatedPublicAt: found.UpdatedPublicAt,
		CreatedAt:       found.CreatedAt,
		UpdatedAt:       found.UpdatedAt,
	}
	if found.LinkCardMediaID != nil {
		linkCard := *found.LinkCardMediaID
		shown.LinkCardMediaId = &linkCard
	}
	if found.PublicRevision != nil {
		public := *found.PublicRevision
		shown.PublicRevisionId = &public
	}
	shown.Schedule = toAPISchedule(found.Schedule)
	shown.Unpublishing = toAPIUnpublishing(found.Unpublishing)
	shown.Deletion = toAPIDeletion(found.Deletion)
	if found.Byline != nil {
		byline := toAPIByline(*found.Byline)
		shown.Byline = &byline
	}
	return shown
}

func toAPIPublicPost(found PublicPost) PublicPostResponse {
	return PublicPostResponse{
		Id:            found.ID,
		Slug:          found.Slug,
		OriginalSlug:  found.OriginalSlug,
		Title:         found.Title,
		Summary:       found.Summary,
		Category:      toAPICategory(found.Category),
		Body:          toAPIBody(found.Body),
		Header:        toAPIHeader(found.Header),
		LinkCardImage: toAPIPostPicture(found.LinkCard, nil),
		Media:         showPostMedia(found.Media, nil),
		Byline:        toAPIByline(found.Byline),
		Related:       toAPISummaries(found.Related),
		PublishedAt:   found.PublishedAt,
		UpdatedAt:     found.UpdatedAt,
	}
}

func toAPIArchive(found Archive) PostArchive {
	shown := PostArchive{
		Posts: toAPISummaries(found.Posts),
		Page:  found.Page,
		Pages: found.Pages,
		Total: found.Total,
	}
	if found.Category != nil {
		category := toAPICategory(*found.Category)
		shown.Category = &category
	}
	return shown
}

func toAPISummaries(listed []PostSummary) []PostSummaryResponse {
	shown := make([]PostSummaryResponse, 0, len(listed))
	for _, one := range listed {
		shown = append(shown, toAPISummary(one))
	}
	return shown
}

func toAPISummary(found PostSummary) PostSummaryResponse {
	shown := PostSummaryResponse{
		Id:           found.ID,
		Slug:         found.Slug,
		OriginalSlug: found.OriginalSlug,
		Title:        found.Title,
		Summary:      found.Summary,
		Category:     toAPICategory(found.Category),
		Byline:       toAPIByline(found.Byline),
		PublishedAt:  found.PublishedAt,
		UpdatedAt:    found.UpdatedAt,
	}
	return shown
}

func showPostMedia(held []PostMedia, sign func(string) string) []PostMediaResponse {
	shown := make([]PostMediaResponse, 0, len(held))
	for _, one := range held {
		shown = append(shown, *toAPIPostPicture(&one, sign))
	}
	return shown
}

func toAPIPostPicture(found *PostMedia, sign func(string) string) *PostMediaResponse {
	if found == nil {
		return nil
	}
	address := PostMediaURL(found.ID, found.Purpose, media.ImageSizeVersion)
	thumb := PostMediaThumbURL(found.ID, media.ImageSizeVersion)
	if sign != nil {
		address = sign(address)
		thumb = sign(thumb)
	}
	return &PostMediaResponse{
		Id:       found.ID,
		PostId:   found.PostID,
		Purpose:  PostMediaPurpose(found.Purpose),
		Url:      address,
		ThumbUrl: thumb,
		Width:    found.Width,
		Height:   found.Height,
	}
}

func toAPIHeader(found *Header) *PostHeader {
	if found == nil {
		return nil
	}
	shown := &PostHeader{MediaId: found.MediaID, Alt: found.Alt}
	if found.Caption != "" {
		shown.Caption = pointer(found.Caption)
	}
	return shown
}

func toAPIByline(found Byline) PostByline {
	shown := PostByline{
		Handle:       found.Handle,
		DisplayName:  found.DisplayName,
		ContactEmail: found.ContactEmail,
		Historical:   found.AccountID == nil,
	}
	if found.Avatar != nil {
		shown.Avatar = &profile.ProfilePicture{
			Url:    account.PictureURL(account.Avatar, found.Avatar.MediaID, found.Avatar.ImageSizeVersion),
			Width:  found.Avatar.Width,
			Height: found.Avatar.Height,
		}
	}
	return shown
}

func toAPIBody(stored json.RawMessage) PostBody {
	var shown PostBody
	if err := json.Unmarshal(stored, &shown); err != nil {
		return PostBody{Version: 0, Content: []map[string]any{}}
	}
	if shown.Content == nil {
		shown.Content = []map[string]any{}
	}
	return shown
}

func toHeaderEdit(request *PostHeaderEdit) *HeaderEdit {
	if request == nil {
		return nil
	}
	edit := &HeaderEdit{MediaID: request.MediaId, Alt: request.Alt}
	if request.Caption != nil {
		edit.Caption = *request.Caption
	}
	return edit
}

func optionalID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	id := uuid.UUID(*value)
	return &id
}

func pointer[value any](of value) *value {
	return &of
}

type CorrectPostAddressRequest struct {
	Slug string `json:"slug"`
}

type CorrectPostBylineRequest struct {
	Handle string `json:"handle"`
}

type CreatePostRequest struct {
	CategoryId uuid.UUID `json:"categoryId"`
	Title      string    `json:"title"`
}

type PostResponse struct {
	Author           PostAuthor          `json:"author"`
	Byline           *PostByline         `json:"byline,omitempty"`
	Category         BlogCategory        `json:"category"`
	CreatedAt        time.Time           `json:"createdAt"`
	Deletion         *PostDeletion       `json:"deletion,omitempty"`
	Body             PostBody            `json:"body"`
	BodyVersion      int                 `json:"bodyVersion"`
	FormerAddresses  []string            `json:"formerAddresses"`
	Header           *PostHeader         `json:"header,omitempty"`
	Id               uuid.UUID           `json:"id"`
	Media            []PostMediaResponse `json:"media"`
	PublicRevisionId *uuid.UUID          `json:"publicRevisionId,omitempty"`
	PublishedAt      *time.Time          `json:"publishedAt,omitempty"`
	Schedule         *PostSchedule       `json:"schedule,omitempty"`
	Slug             string              `json:"slug"`
	LinkCardMediaId  *uuid.UUID          `json:"linkCardMediaId,omitempty"`
	Status           PostStatus          `json:"status"`
	Summary          string              `json:"summary"`
	Title            string              `json:"title"`
	UpdatedAt        time.Time           `json:"updatedAt"`
	UpdatedPublicAt  *time.Time          `json:"updatedPublicAt,omitempty"`
	Version          int                 `json:"version"`
	Unpublishing     *PostUnpublishing   `json:"unpublishing,omitempty"`
}

type PostArchive struct {
	Category *BlogCategory         `json:"category,omitempty"`
	Page     int                   `json:"page"`
	Pages    int                   `json:"pages"`
	Posts    []PostSummaryResponse `json:"posts"`
	Total    int                   `json:"total"`
}

type PostAuthor struct {
	Handle string `json:"handle"`
}

type PostByline struct {
	Avatar       *profile.ProfilePicture `json:"avatar,omitempty"`
	ContactEmail string                  `json:"contactEmail"`
	DisplayName  string                  `json:"displayName"`
	Handle       string                  `json:"handle"`
	Historical   bool                    `json:"historical"`
}

type PostConflict struct {
	Code      BlogErrorCode `json:"code"`
	Error     string        `json:"error"`
	Field     *string       `json:"field,omitempty"`
	UpdatedAt *time.Time    `json:"updatedAt,omitempty"`
	Version   *int          `json:"version,omitempty"`
}

type PostBody struct {
	Content []map[string]interface{} `json:"content"`
	Version int                      `json:"version"`
}

type PostHeader struct {
	Alt     string    `json:"alt"`
	Caption *string   `json:"caption,omitempty"`
	MediaId uuid.UUID `json:"mediaId"`
}

type PostHeaderEdit struct {
	Alt     string    `json:"alt"`
	Caption *string   `json:"caption,omitempty"`
	MediaId uuid.UUID `json:"mediaId"`
}

type PostList struct {
	Posts []PostResponse `json:"posts"`
}

type PostMediaResponse struct {
	Height   int              `json:"height"`
	Id       uuid.UUID        `json:"id"`
	PostId   uuid.UUID        `json:"postId"`
	Purpose  PostMediaPurpose `json:"purpose"`
	ThumbUrl string           `json:"thumbUrl"`
	Url      string           `json:"url"`
	Width    int              `json:"width"`
}

type PostMediaPurpose string

const (
	PostMediaPurposeBody     PostMediaPurpose = "body"
	PostMediaPurposeHeader   PostMediaPurpose = "header"
	PostMediaPurposeLinkCard PostMediaPurpose = "link_card"
)

type PostStatus string

const (
	PostStatusDraft       PostStatus = "draft"
	PostStatusPublished   PostStatus = "published"
	PostStatusUnpublished PostStatus = "unpublished"
)

type PostSummaryResponse struct {
	Byline       PostByline   `json:"byline"`
	Category     BlogCategory `json:"category"`
	Id           uuid.UUID    `json:"id"`
	OriginalSlug string       `json:"originalSlug"`
	PublishedAt  time.Time    `json:"publishedAt"`
	Slug         string       `json:"slug"`
	Summary      string       `json:"summary"`
	Title        string       `json:"title"`
	UpdatedAt    *time.Time   `json:"updatedAt,omitempty"`
}

type PublicPostResponse struct {
	Byline        PostByline            `json:"byline"`
	Category      BlogCategory          `json:"category"`
	Body          PostBody              `json:"body"`
	Header        *PostHeader           `json:"header,omitempty"`
	Id            uuid.UUID             `json:"id"`
	Media         []PostMediaResponse   `json:"media"`
	OriginalSlug  string                `json:"originalSlug"`
	PublishedAt   time.Time             `json:"publishedAt"`
	Related       []PostSummaryResponse `json:"related"`
	Slug          string                `json:"slug"`
	LinkCardImage *PostMediaResponse    `json:"linkCardImage,omitempty"`
	Summary       string                `json:"summary"`
	Title         string                `json:"title"`
	UpdatedAt     *time.Time            `json:"updatedAt,omitempty"`
}

type PublishPostRequest struct {
	IntegrationIds     *[]uuid.UUID `json:"integrationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RoleIntegrationIds *[]uuid.UUID `json:"roleIntegrationIds,omitempty"`
	Version            int          `json:"version"`
}

type SavePostRequest struct {
	CategoryId      uuid.UUID       `json:"categoryId"`
	Body            PostBody        `json:"body"`
	Header          *PostHeaderEdit `json:"header,omitempty"`
	Slug            string          `json:"slug"`
	LinkCardMediaId *uuid.UUID      `json:"linkCardMediaId,omitempty"`
	Summary         string          `json:"summary"`
	Title           string          `json:"title"`
	Version         int             `json:"version"`
}

type ListPostsParams struct {
	Deleted *bool `json:"deleted,omitempty"`
}

type ListPublishedPostsParams struct {
	Page     *int    `json:"page,omitempty"`
	Category *string `json:"category,omitempty"`
}

func readPostMediaMetadata(parts *multipart.Reader) (AddPostMediaRequest, error) {
	part, err := api.NextPart(parts, api.MetadataPart)
	if err != nil {
		return AddPostMediaRequest{}, err
	}
	var metadata AddPostMediaRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return AddPostMediaRequest{}, api.FormRefusal{
			Reason: "the " + api.MetadataPart + " part is not valid JSON",
			Cause:  err,
		}
	}
	return metadata, nil
}

type AddPostMediaRequest struct {
	Purpose PostMediaPurpose `json:"purpose"`
}
