package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
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
	c.JSON(http.StatusOK, PostList{Posts: toAPIPosts(held)})
}

func (h *Handlers) CreatePost(c *gin.Context) {
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
	c.JSON(http.StatusCreated, toAPIPost(started))
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
	c.JSON(http.StatusOK, toAPIPost(found))
}

func (h *Handlers) SavePost(c *gin.Context, id types.UUID) {
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
			Version:    request.Version,
			CategoryID: uuid.UUID(request.CategoryId),
			Title:      request.Title,
			Summary:    request.Summary,
			Slug:       request.Slug,
			Document:   document,
			Release:    toReleaseEdit(request.Release),
		})
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPost(saved))
}

func (h *Handlers) PublishPost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "publishing a post")
	if !ok {
		return
	}
	published, err := h.publications.PublishPost(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPost(published))
}

func (h *Handlers) GetPublishedPost(c *gin.Context, slug string) {
	found, err := h.publications.PublishedPost(c.Request.Context(), slug)
	if errors.Is(err, publication.ErrPostNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No such post."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the post."})
		return
	}
	c.JSON(http.StatusOK, toAPIPublicPost(found))
}

// postEditor answers the signed-in account and how far its post access reaches.
func (h *Handlers) postEditor(c *gin.Context, action string) (publication.Editor, bool) {
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
		c.JSON(http.StatusNotFound, gin.H{"error": "No such post."})
	case errors.Is(err, publication.ErrNotPostEditor):
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only this post's contributor or an Illarin admin can do that.",
		})
	case errors.Is(err, publication.ErrSlugLocked):
		c.JSON(http.StatusForbidden, gin.H{
			"error": "The address of a published post is fixed. An admin can correct it.",
			"field": "slug",
		})
	case errors.As(err, &stale):
		c.JSON(http.StatusConflict, PostConflict{
			Error:     "Someone saved this post while you were writing. Reload to carry on.",
			Field:     pointer("version"),
			Version:   stale.Version,
			UpdatedAt: &stale.UpdatedAt,
		})
	default:
		h.publicationError(c, err)
	}
}

func toAPIPosts(held []publication.Post) []Post {
	listed := make([]Post, 0, len(held))
	for _, one := range held {
		listed = append(listed, toAPIPost(one))
	}
	return listed
}

func toAPIPost(found publication.Post) Post {
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
	return shown
}

func toAPIPublicPost(found publication.PublicPost) PublicPost {
	return PublicPost{
		Id:          types.UUID(found.ID),
		Slug:        found.Slug,
		Title:       found.Title,
		Summary:     found.Summary,
		Category:    toAPICategory(found.Category),
		Document:    toAPIDocument(found.Document),
		Release:     toAPIRelease(found.Release),
		Byline:      toAPIByline(found.Byline),
		PublishedAt: found.PublishedAt,
		UpdatedAt:   found.UpdatedAt,
	}
}

func toAPIByline(found publication.Byline) PostByline {
	shown := PostByline{
		Handle:       found.Handle,
		DisplayName:  found.DisplayName,
		ContactEmail: found.ContactEmail,
		Positions:    found.Positions,
		Distinctions: found.Distinctions,
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
