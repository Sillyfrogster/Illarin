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
		return h.publications.DeletedPosts
	}
	return h.publications.Posts
}

func (h *Handlers) CreatePost(c *gin.Context) {
	editor, ok := h.postEditor(c, "starting a post")
	if !ok {
		return
	}
	var request CreatePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send the post as JSON.")
		return
	}
	started, err := h.publications.CreatePost(c.Request.Context(), editor, PostEdit{
		GrantID:    optionalID(request.GrantId),
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
	found, err := h.publications.Post(c.Request.Context(), editor, id)
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
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send the drafted changes as JSON.")
		return
	}
	document, err := json.Marshal(request.Document)
	if err != nil {
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send the post body as JSON.")
		return
	}
	saved, err := h.publications.SavePost(c.Request.Context(), editor, id,
		PostSave{
			Version:       request.Version,
			CategoryID:    request.CategoryId,
			Title:         request.Title,
			Summary:       request.Summary,
			Slug:          request.Slug,
			Document:      document,
			Release:       toReleaseEdit(request.Release),
			Header:        toHeaderEdit(request.Header),
			SocialMediaID: optionalID(request.SocialMediaId),
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
	added, err := h.publications.AddPostMedia(
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
	c.JSON(http.StatusCreated, toAPIPostPicture(&added, h.publications.SignPrivate))
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
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Include the current drafted changes version.", "version")
		return
	}
	published, err := h.publications.PublishPost(
		c.Request.Context(), editor, id, request.Version,
		announcementOf(request.DestinationIds, request.RoleDestinationIds, request.Note),
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
	moved, err := h.publications.CorrectAddress(
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
	corrected, err := h.publications.CorrectByline(
		c.Request.Context(), editor, id, request.Handle,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(corrected))
}

func (h *Handlers) ListPostApps(c *gin.Context) {
	found, err := h.publications.ReadableApps(c.Request.Context())
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the apps.")
		return
	}
	c.JSON(http.StatusOK, PublicationAppList{Apps: toAPIApps(found)})
}

func (h *Handlers) ListPostCategories(c *gin.Context) {
	found, err := h.publications.ReadableCategories(c.Request.Context())
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the categories.")
		return
	}
	c.JSON(http.StatusOK, PublicationCategoryList{Categories: toAPICategories(found)})
}

func (h *Handlers) ListPublishedPosts(c *gin.Context) {
	q := api.ReadQuery(c)
	params := ListPublishedPostsParams{
		Page:     api.QueryNumber(q, "page"),
		Category: api.QueryText[string](q, "category"),
		App:      api.QueryText[string](q, "app"),
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
	if params.App != nil {
		asked.App = *params.App
	}
	found, err := h.publications.Archive(c.Request.Context(), asked)
	switch {
	case errors.Is(err, ErrCategoryNotFound):
		api.Refuse(c, http.StatusNotFound, "No such publication category.")
		return
	case errors.Is(err, ErrAppNotFound):
		api.Refuse(c, http.StatusNotFound, "No such publication app.")
		return
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not load blog posts. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIArchive(found))
}

func (h *Handlers) GetPublishedPost(c *gin.Context) {
	slug := c.Param("slug")
	found, err := h.publications.PublishedPost(c.Request.Context(), slug)
	if errors.Is(err, ErrPostNotFound) {
		if h.withdrawnPost(c, slug) {
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
		refuseField(c, http.StatusRequestEntityTooLarge, PublicationErrorCodeInvalid,
			"That picture is larger than the upload limit.", api.FilePart)
	case errors.Is(err, storage.ErrInsufficientSpace):
		refusePublication(c, http.StatusServiceUnavailable, PublicationErrorCodeServerError,
			"Uploads are temporarily unavailable because storage is low.")
	default:
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
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
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound, "No such post.")
	case errors.Is(err, ErrRevisionNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound,
			"This post has no such revision.")
	case errors.Is(err, ErrDestinationRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not send to that destination.")
	case errors.Is(err, ErrRoleRefused):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"This post may not mention that destination's role.")
	case errors.Is(err, ErrNotPostEditor):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"Only this post's contributor or an Illarin admin can do that.")
	case errors.Is(err, ErrSlugLocked):
		refuseField(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"The address of a published post is fixed. An admin can correct it.", "slug")
	case errors.Is(err, ErrNotPostAdmin):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"Only an Illarin admin can correct a published post.")
	case errors.Is(err, ErrPostUnpublished):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"There is nothing to correct until the post is published.")
	case errors.Is(err, ErrPostNotPublic):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Only a published post can be withdrawn.")
	case errors.Is(err, ErrPostNotWithdrawn):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This post is not withdrawn.")
	case errors.Is(err, ErrPostWithdrawn):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This post is withdrawn. Republish it to make it public again.")
	case errors.Is(err, ErrPostDeleted):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This post is deleted. Restore it before editing.")
	case errors.Is(err, ErrPostNotDeleted):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This post has not been deleted.")
	case errors.Is(err, ErrPostInPublicView):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Withdraw the post before deleting it.")
	case errors.Is(err, ErrRecoveryExpired):
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"The recovery deadline has passed. This post cannot be restored.")
	case errors.Is(err, ErrDeletePublished):
		refusePublication(c, http.StatusForbidden, PublicationErrorCodeForbidden,
			"Only an Illarin admin can delete or restore a previously published post.")
	case errors.Is(err, ErrSchedulePublishing):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This revision is being published and can no longer be changed.",
			Code:  PublicationErrorCodeScheduleRunning,
		})
	case errors.As(err, &stale):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error:     "This post was saved in another session. Copy any unsaved text, then reload to edit the latest version.",
			Code:      PublicationErrorCodeStaleVersion,
			Field:     pointer("version"),
			Version:   &stale.Version,
			UpdatedAt: &stale.UpdatedAt,
		})
	default:
		h.publicationError(c, err)
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
		Document:        toAPIDocument(found.Document),
		DocumentVersion: found.DocumentVersion,
		Release:         toAPIRelease(found.Release),
		Header:          toAPIHeader(found.Header),
		Media:           showPostMedia(found.Media, h.publications.SignPrivate),
		FormerAddresses: found.FormerAddresses,
		Version:         found.Version,
		Author:          PostAuthor{Handle: found.Author.Handle},
		PublishedAt:     found.PublishedAt,
		UpdatedPublicAt: found.UpdatedPublicAt,
		CreatedAt:       found.CreatedAt,
		UpdatedAt:       found.UpdatedAt,
	}
	if found.GrantID != nil {
		grantID := *found.GrantID
		shown.GrantId = &grantID
	}
	if found.App != nil {
		app := toAPIApp(*found.App)
		shown.App = &app
	}
	if found.SocialMediaID != nil {
		social := *found.SocialMediaID
		shown.SocialMediaId = &social
	}
	if found.PublicRevision != nil {
		public := *found.PublicRevision
		shown.PublicRevisionId = &public
	}
	shown.Schedule = toAPISchedule(found.Schedule)
	shown.Withdrawal = toAPIWithdrawal(found.Withdrawal)
	shown.Deletion = toAPIDeletion(found.Deletion)
	if found.Byline != nil {
		byline := toAPIByline(*found.Byline)
		shown.Byline = &byline
	}
	return shown
}

func toAPIPublicPost(found PublicPost) PublicPostResponse {
	return PublicPostResponse{
		Id:           found.ID,
		Slug:         found.Slug,
		OriginalSlug: found.OriginalSlug,
		Title:        found.Title,
		Summary:      found.Summary,
		Category:     toAPICategory(found.Category),
		Document:     toAPIDocument(found.Document),
		Release:      toAPIRelease(found.Release),
		Header:       toAPIHeader(found.Header),
		SocialImage:  toAPIPostPicture(found.SocialMedia, nil),
		Media:        showPostMedia(found.Media, nil),
		Byline:       toAPIByline(found.Byline),
		Related:      toAPISummaries(found.Related),
		PublishedAt:  found.PublishedAt,
		UpdatedAt:    found.UpdatedAt,
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
	if found.App != nil {
		app := toAPIApp(*found.App)
		shown.App = &app
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
	if found.App != nil {
		app := toAPIApp(*found.App)
		shown.App = &app
	}
	if found.ReleaseVersion != "" {
		shown.ReleaseVersion = pointer(found.ReleaseVersion)
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
	address := PostMediaURL(found.ID, found.Purpose, media.DerivativeVersion)
	thumb := PostMediaThumbURL(found.ID, media.DerivativeVersion)
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
		shown.Avatar = &profile.ProfileAvatar{
			Url:    account.AvatarURL(found.Avatar.MediaID, found.Avatar.DerivativeVersion),
			Width:  found.Avatar.Width,
			Height: found.Avatar.Height,
		}
	}
	if found.App != nil {
		app := toAPIApp(*found.App)
		shown.App = &app
	}
	return shown
}

func toAPIRelease(found *Release) *PostRelease {
	if found == nil {
		return nil
	}
	shown := PostRelease{App: toAPIApp(found.App), Version: found.Version}
	if found.Address != "" {
		shown.Address = pointer(found.Address)
	}
	return &shown
}

func toAPIDocument(stored json.RawMessage) PostDocument {
	var shown PostDocument
	if err := json.Unmarshal(stored, &shown); err != nil {
		return PostDocument{Version: 0, Content: []map[string]any{}}
	}
	if shown.Content == nil {
		shown.Content = []map[string]any{}
	}
	return shown
}

func toReleaseEdit(request *PostReleaseEdit) *ReleaseEdit {
	if request == nil {
		return nil
	}
	edit := &ReleaseEdit{
		AppID:   request.AppId,
		Version: request.Version,
	}
	if request.Address != nil {
		edit.Address = *request.Address
	}
	return edit
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
	CategoryId uuid.UUID  `json:"categoryId"`
	GrantId    *uuid.UUID `json:"grantId,omitempty"`
	Title      string     `json:"title"`
}

type PostResponse struct {
	App              *PublicationApp     `json:"app,omitempty"`
	Author           PostAuthor          `json:"author"`
	Byline           *PostByline         `json:"byline,omitempty"`
	Category         PublicationCategory `json:"category"`
	CreatedAt        time.Time           `json:"createdAt"`
	Deletion         *PostDeletion       `json:"deletion,omitempty"`
	Document         PostDocument        `json:"document"`
	DocumentVersion  int                 `json:"documentVersion"`
	FormerAddresses  []string            `json:"formerAddresses"`
	GrantId          *uuid.UUID          `json:"grantId,omitempty"`
	Header           *PostHeader         `json:"header,omitempty"`
	Id               uuid.UUID           `json:"id"`
	Media            []PostMediaResponse `json:"media"`
	PublicRevisionId *uuid.UUID          `json:"publicRevisionId,omitempty"`
	PublishedAt      *time.Time          `json:"publishedAt,omitempty"`
	Release          *PostRelease        `json:"release,omitempty"`
	Schedule         *PostSchedule       `json:"schedule,omitempty"`
	Slug             string              `json:"slug"`
	SocialMediaId    *uuid.UUID          `json:"socialMediaId,omitempty"`
	Status           PostStatus          `json:"status"`
	Summary          string              `json:"summary"`
	Title            string              `json:"title"`
	UpdatedAt        time.Time           `json:"updatedAt"`
	UpdatedPublicAt  *time.Time          `json:"updatedPublicAt,omitempty"`
	Version          int                 `json:"version"`
	Withdrawal       *PostWithdrawal     `json:"withdrawal,omitempty"`
}

type PostArchive struct {
	App      *PublicationApp       `json:"app,omitempty"`
	Category *PublicationCategory  `json:"category,omitempty"`
	Page     int                   `json:"page"`
	Pages    int                   `json:"pages"`
	Posts    []PostSummaryResponse `json:"posts"`
	Total    int                   `json:"total"`
}

type PostAuthor struct {
	Handle string `json:"handle"`
}

type PostByline struct {
	App          *PublicationApp        `json:"app,omitempty"`
	Avatar       *profile.ProfileAvatar `json:"avatar,omitempty"`
	ContactEmail string                 `json:"contactEmail"`
	DisplayName  string                 `json:"displayName"`
	Handle       string                 `json:"handle"`
	Historical   bool                   `json:"historical"`
}

type PostConflict struct {
	Code      PublicationErrorCode `json:"code"`
	Error     string               `json:"error"`
	Field     *string              `json:"field,omitempty"`
	UpdatedAt *time.Time           `json:"updatedAt,omitempty"`
	Version   *int                 `json:"version,omitempty"`
}

type PostDocument struct {
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
	PostMediaPurposeDocument PostMediaPurpose = "document"
	PostMediaPurposeHeader   PostMediaPurpose = "header"
	PostMediaPurposeSocial   PostMediaPurpose = "social"
)

type PostRelease struct {
	Address *string        `json:"address,omitempty"`
	App     PublicationApp `json:"app"`
	Version string         `json:"version"`
}

type PostReleaseEdit struct {
	Address *string   `json:"address,omitempty"`
	AppId   uuid.UUID `json:"appId"`
	Version string    `json:"version"`
}

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusWithdrawn PostStatus = "withdrawn"
)

type PostSummaryResponse struct {
	App            *PublicationApp     `json:"app,omitempty"`
	Byline         PostByline          `json:"byline"`
	Category       PublicationCategory `json:"category"`
	Id             uuid.UUID           `json:"id"`
	OriginalSlug   string              `json:"originalSlug"`
	PublishedAt    time.Time           `json:"publishedAt"`
	ReleaseVersion *string             `json:"releaseVersion,omitempty"`
	Slug           string              `json:"slug"`
	Summary        string              `json:"summary"`
	Title          string              `json:"title"`
	UpdatedAt      *time.Time          `json:"updatedAt,omitempty"`
}

type PublicPostResponse struct {
	Byline       PostByline            `json:"byline"`
	Category     PublicationCategory   `json:"category"`
	Document     PostDocument          `json:"document"`
	Header       *PostHeader           `json:"header,omitempty"`
	Id           uuid.UUID             `json:"id"`
	Media        []PostMediaResponse   `json:"media"`
	OriginalSlug string                `json:"originalSlug"`
	PublishedAt  time.Time             `json:"publishedAt"`
	Related      []PostSummaryResponse `json:"related"`
	Release      *PostRelease          `json:"release,omitempty"`
	Slug         string                `json:"slug"`
	SocialImage  *PostMediaResponse    `json:"socialImage,omitempty"`
	Summary      string                `json:"summary"`
	Title        string                `json:"title"`
	UpdatedAt    *time.Time            `json:"updatedAt,omitempty"`
}

type PublicationAppList struct {
	Apps []PublicationApp `json:"apps"`
}

type PublishPostRequest struct {
	DestinationIds     *[]uuid.UUID `json:"destinationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RoleDestinationIds *[]uuid.UUID `json:"roleDestinationIds,omitempty"`
	Version            int          `json:"version"`
}

type SavePostRequest struct {
	CategoryId    uuid.UUID        `json:"categoryId"`
	Document      PostDocument     `json:"document"`
	Header        *PostHeaderEdit  `json:"header,omitempty"`
	Release       *PostReleaseEdit `json:"release,omitempty"`
	Slug          string           `json:"slug"`
	SocialMediaId *uuid.UUID       `json:"socialMediaId,omitempty"`
	Summary       string           `json:"summary"`
	Title         string           `json:"title"`
	Version       int              `json:"version"`
}

type ListPostsParams struct {
	Deleted *bool `json:"deleted,omitempty"`
}

type ListPublishedPostsParams struct {
	Page     *int    `json:"page,omitempty"`
	Category *string `json:"category,omitempty"`
	App      *string `json:"app,omitempty"`
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
