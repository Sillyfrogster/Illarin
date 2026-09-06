package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListPosts(c *gin.Context) {
	editor, ok := h.postEditor(c, "reading posts")
	if !ok {
		return
	}
	held, err := h.publications.Posts(c.Request.Context(), editor)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostList{Posts: h.toAPIPosts(held)})
}

func (h *Handlers) CreatePost(c *gin.Context, _ CreatePostParams) {
	editor, ok := h.postEditor(c, "starting a post")
	if !ok {
		return
	}
	var request CreatePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the post as JSON."})
		return
	}
	started, err := h.publications.CreatePost(c.Request.Context(), editor, publication.PostEdit{
		GrantID:    optionalID(request.GrantId),
		CategoryID: uuid.UUID(request.CategoryId),
		Title:      request.Title,
	})
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toAPIPost(started))
}

func (h *Handlers) GetPost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "reading a post")
	if !ok {
		return
	}
	found, err := h.publications.Post(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(found))
}

func (h *Handlers) SavePost(c *gin.Context, id types.UUID, _ SavePostParams) {
	editor, ok := h.postEditor(c, "saving a post")
	if !ok {
		return
	}
	var request SavePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the working copy as JSON."})
		return
	}
	document, err := json.Marshal(request.Document)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the post body as JSON."})
		return
	}
	saved, err := h.publications.SavePost(c.Request.Context(), editor, uuid.UUID(id),
		publication.PostSave{
			Version:       request.Version,
			CategoryID:    uuid.UUID(request.CategoryId),
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

func (h *Handlers) AddPostMedia(c *gin.Context, id types.UUID, _ AddPostMediaParams) {
	editor, ok := h.postEditor(c, "adding a picture to a post")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refusePostMedia(c, refusal{
			reason: "send the picture as form data, with a metadata part and a file part",
			cause:  err,
		})
		return
	}
	metadata, err := readPostMediaMetadata(parts)
	if err != nil {
		h.refusePostMedia(c, err)
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refusePostMedia(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	added, err := h.publications.AddPostMedia(
		c.Request.Context(), editor, uuid.UUID(id), string(metadata.Purpose), limitedFile,
	)
	var refused publication.FieldError
	switch {
	case errors.Is(err, publication.ErrPostNotFound), errors.Is(err, publication.ErrNotPostEditor):
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

func (h *Handlers) PublishPost(c *gin.Context, id types.UUID, _ PublishPostParams) {
	editor, ok := h.postEditor(c, "publishing a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	published, err := h.publications.PublishPost(
		c.Request.Context(), editor, uuid.UUID(id), version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(published))
}

func (h *Handlers) CorrectPostAddress(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "correcting a post address")
	if !ok {
		return
	}
	var request CorrectPostAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the address as JSON."})
		return
	}
	moved, err := h.publications.CorrectAddress(
		c.Request.Context(), editor, uuid.UUID(id), request.Slug,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(moved))
}

func (h *Handlers) CorrectPostByline(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "correcting a post byline")
	if !ok {
		return
	}
	var request CorrectPostBylineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the handle as JSON."})
		return
	}
	corrected, err := h.publications.CorrectByline(
		c.Request.Context(), editor, uuid.UUID(id), request.Handle,
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the apps."})
		return
	}
	c.JSON(http.StatusOK, PublicationAppList{Apps: toAPIApps(found)})
}

func (h *Handlers) ListPostCategories(c *gin.Context) {
	found, err := h.publications.ReadableCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the categories."})
		return
	}
	c.JSON(http.StatusOK, PublicationCategoryList{Categories: toAPICategories(found)})
}

func (h *Handlers) ListPublishedPosts(c *gin.Context, params ListPublishedPostsParams) {
	asked := publication.ArchiveQuery{Page: 1}
	if params.Page != nil {
		asked.Page = *params.Page
	}
	if asked.Page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Archive pages count from one."})
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
	case errors.Is(err, publication.ErrCategoryNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such publication category."})
		return
	case errors.Is(err, publication.ErrAppNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such publication app."})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the archive."})
		return
	}
	c.JSON(http.StatusOK, toAPIArchive(found))
}

func (h *Handlers) GetPublishedPost(c *gin.Context, slug string) {
	found, err := h.publications.PublishedPost(c.Request.Context(), slug)
	if errors.Is(err, publication.ErrPostNotFound) {
		if h.withdrawnPost(c, slug) {
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "No such post."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the post."})
		return
	}
	c.JSON(http.StatusOK, toAPIPublicPost(found))
}

// refusePostMedia answers an upload the byte path could not take.
func (h *Handlers) refusePostMedia(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "That picture is larger than the upload limit.",
		})
	case errors.Is(err, storage.ErrInsufficientSpace):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Uploads are temporarily unavailable because storage is low.",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "That picture could not be read. Use a PNG, JPEG, WebP or GIF.",
		})
	}
}

// postEditor answers the credential acting on a post and how far its access
// reaches. A publication token reaches its own grant; a session reaches
// everything the account behind it may manage.
func (h *Handlers) postEditor(c *gin.Context, action string) (publication.Editor, bool) {
	if bearing, ok := publicationBearing(c); ok {
		return bearing.Editor(), true
	}
	current, ok := h.verifiedAccount(c, action)
	if !ok {
		return publication.Editor{}, false
	}
	return publication.Editor{
		ID:    current.ID,
		Admin: current.Role == account.RoleAdmin,
	}, true
}

func (h *Handlers) postError(c *gin.Context, err error) {
	var stale publication.Stale
	switch {
	case errors.Is(err, publication.ErrPostNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such post.")
	case errors.Is(err, publication.ErrRevisionNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound,
			"This post has no such edition.")
	case errors.Is(err, publication.ErrNotPostEditor):
		refusePublication(c, http.StatusForbidden, CodeForbidden,
			"Only this post's contributor or an Illarin admin can do that.")
	case errors.Is(err, publication.ErrSlugLocked):
		refuseField(c, http.StatusForbidden, CodeForbidden,
			"The address of a published post is fixed. An admin can correct it.", "slug")
	case errors.Is(err, publication.ErrNotPostAdmin):
		refusePublication(c, http.StatusForbidden, CodeForbidden,
			"Only an Illarin admin can correct a published post.")
	case errors.Is(err, publication.ErrPostUnpublished):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"There is nothing to correct until the post is published.")
	case errors.Is(err, publication.ErrPostNotPublic):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"Only a post readers can see right now can be withdrawn.")
	case errors.Is(err, publication.ErrPostNotWithdrawn):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"This post is not out of public view.")
	case errors.Is(err, publication.ErrPostWithdrawn):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"This post is out of public view. Put it back with republish.")
	case errors.Is(err, publication.ErrSchedulePublishing):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This edition is going live now and can no longer be changed.",
			Code:  CodeScheduleRunning,
		})
	case errors.As(err, &stale):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error:     "Someone saved this post while you were writing. Reload to carry on.",
			Code:      CodeStaleVersion,
			Field:     pointer("version"),
			Version:   &stale.Version,
			UpdatedAt: &stale.UpdatedAt,
		})
	default:
		h.publicationError(c, err)
	}
}

func (h *Handlers) toAPIPosts(held []publication.Post) []Post {
	listed := make([]Post, 0, len(held))
	for _, one := range held {
		listed = append(listed, h.toAPIPost(one))
	}
	return listed
}

func (h *Handlers) toAPIPost(found publication.Post) Post {
	shown := Post{
		Id:              types.UUID(found.ID),
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
		grantID := types.UUID(*found.GrantID)
		shown.GrantId = &grantID
	}
	if found.App != nil {
		app := toAPIApp(*found.App)
		shown.App = &app
	}
	if found.SocialMediaID != nil {
		social := types.UUID(*found.SocialMediaID)
		shown.SocialMediaId = &social
	}
	if found.PublicRevision != nil {
		public := types.UUID(*found.PublicRevision)
		shown.PublicRevisionId = &public
	}
	shown.Schedule = toAPISchedule(found.Schedule)
	shown.Withdrawal = toAPIWithdrawal(found.Withdrawal)
	if found.Byline != nil {
		byline := toAPIByline(*found.Byline)
		shown.Byline = &byline
	}
	return shown
}

func toAPIPublicPost(found publication.PublicPost) PublicPost {
	return PublicPost{
		Id:           types.UUID(found.ID),
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

func toAPIArchive(found publication.Archive) PostArchive {
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

func toAPISummaries(listed []publication.PostSummary) []PostSummary {
	shown := make([]PostSummary, 0, len(listed))
	for _, one := range listed {
		shown = append(shown, toAPISummary(one))
	}
	return shown
}

func toAPISummary(found publication.PostSummary) PostSummary {
	shown := PostSummary{
		Id:           types.UUID(found.ID),
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

// showPostMedia addresses a set of pictures, signing a working copy's own.
func showPostMedia(held []publication.PostMedia, sign func(string) string) []PostMedia {
	shown := make([]PostMedia, 0, len(held))
	for _, one := range held {
		shown = append(shown, *toAPIPostPicture(&one, sign))
	}
	return shown
}

func toAPIPostPicture(found *publication.PostMedia, sign func(string) string) *PostMedia {
	if found == nil {
		return nil
	}
	address := publication.PostMediaURL(found.ID, found.Purpose, media.DerivativeVersion)
	thumb := publication.PostMediaThumbURL(found.ID, media.DerivativeVersion)
	if sign != nil {
		address = sign(address)
		thumb = sign(thumb)
	}
	return &PostMedia{
		Id:       types.UUID(found.ID),
		PostId:   types.UUID(found.PostID),
		Purpose:  PostMediaPurpose(found.Purpose),
		Url:      address,
		ThumbUrl: thumb,
		Width:    found.Width,
		Height:   found.Height,
	}
}

func toAPIHeader(found *publication.Header) *PostHeader {
	if found == nil {
		return nil
	}
	shown := &PostHeader{MediaId: types.UUID(found.MediaID), Alt: found.Alt}
	if found.Caption != "" {
		shown.Caption = pointer(found.Caption)
	}
	return shown
}

func toAPIByline(found publication.Byline) PostByline {
	shown := PostByline{
		Handle:       found.Handle,
		DisplayName:  found.DisplayName,
		ContactEmail: found.ContactEmail,
		Positions:    found.Positions,
		Distinctions: found.Distinctions,
		Historical:   found.AccountID == nil,
	}
	if found.Avatar != nil {
		shown.Avatar = &ProfileAvatar{
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

func toAPIRelease(found *publication.Release) *PostRelease {
	if found == nil {
		return nil
	}
	shown := PostRelease{App: toAPIApp(found.App), Version: found.Version}
	if found.Address != "" {
		shown.Address = pointer(found.Address)
	}
	return &shown
}

// toAPIDocument hands on the body Go already validated and stored.
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

func toReleaseEdit(request *PostReleaseEdit) *publication.ReleaseEdit {
	if request == nil {
		return nil
	}
	edit := &publication.ReleaseEdit{
		AppID:   uuid.UUID(request.AppId),
		Version: request.Version,
	}
	if request.Address != nil {
		edit.Address = *request.Address
	}
	return edit
}

func toHeaderEdit(request *PostHeaderEdit) *publication.HeaderEdit {
	if request == nil {
		return nil
	}
	edit := &publication.HeaderEdit{MediaID: uuid.UUID(request.MediaId), Alt: request.Alt}
	if request.Caption != nil {
		edit.Caption = *request.Caption
	}
	return edit
}

func optionalID(value *types.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	id := uuid.UUID(*value)
	return &id
}

func pointer[value any](of value) *value {
	return &of
}
