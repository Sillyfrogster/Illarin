package blog

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/profile"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListBlogCategories(c *gin.Context) {
	if _, ok := h.blogAdmin(c, "reading blog categories"); !ok {
		return
	}
	sorted, err := h.blog.Categories(c.Request.Context())
	if err != nil {
		h.blogError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogCategoryList{Categories: toAPICategories(sorted)})
}

func (h *Handlers) UpdateBlogCategory(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	admin, ok := h.blogAdmin(c, "changing a blog category")
	if !ok {
		return
	}
	var request UpdateBlogCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the change as JSON.")
		return
	}
	updated, err := h.blog.UpdateCategory(
		c.Request.Context(), admin.ID, id, CategoryUpdate{
			Label:   request.Label,
			Retired: request.Retired,
		},
	)
	if err != nil {
		h.blogError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPICategory(updated))
}

func (h *Handlers) OrderBlogCategories(c *gin.Context) {
	admin, ok := h.blogAdmin(c, "ordering blog categories")
	if !ok {
		return
	}
	var request OrderBlogCategoriesRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the order as JSON.")
		return
	}
	ordered, err := h.blog.OrderCategories(c.Request.Context(), admin.ID, request.CategoryIds)
	if err != nil {
		h.blogError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogCategoryList{Categories: toAPICategories(ordered)})
}

func (h *Handlers) ListWriters(c *gin.Context) {
	if _, ok := h.blogAdmin(c, "reading the writers"); !ok {
		return
	}
	found, err := h.blog.Writers(c.Request.Context())
	if err != nil {
		h.blogError(c, err)
		return
	}
	listed, err := h.withProfiles(c, found)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, WriterList{Writers: listed})
}

func (h *Handlers) SwitchWriterOn(c *gin.Context) {
	admin, ok := h.blogAdmin(c, "switching a writer on")
	if !ok {
		return
	}
	var request SwitchWriterOnRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the handle as JSON.")
		return
	}
	writer, err := h.blog.SwitchWriterOn(c.Request.Context(), admin.ID, request.Handle)
	if err != nil {
		h.blogError(c, err)
		return
	}
	listed, err := h.withProfiles(c, []Writer{writer})
	if err != nil {
		return
	}
	c.JSON(http.StatusCreated, listed[0])
}

func (h *Handlers) SwitchWriterOff(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	admin, ok := h.blogAdmin(c, "switching a writer off")
	if !ok {
		return
	}
	if err := h.blog.SwitchWriterOff(c.Request.Context(), admin.ID, id); err != nil {
		h.blogError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetBlogWorkspace(c *gin.Context) {
	current, ok := api.Verified(c, "opening Your posts")
	if !ok {
		return
	}
	writer, err := h.blog.IsWriter(c.Request.Context(), current.ID)
	if err != nil {
		h.blogError(c, err)
		return
	}
	open, err := h.blog.OpenCategories(c.Request.Context())
	if err != nil {
		h.blogError(c, err)
		return
	}
	c.JSON(http.StatusOK, BlogWorkspace{
		Handle:     current.Handle,
		Admin:      current.Role == api.RoleAdmin,
		Writer:     writer,
		Categories: toAPICategories(open),
	})
}

func (h *Handlers) blogError(c *gin.Context, err error) {
	var field FieldError
	switch {
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, field.Message, field.Field)
	case errors.Is(err, ErrAccountNotFound):
		refuseBlog(c, http.StatusNotFound, BlogErrorCodeNotFound, "No such account.")
	case errors.Is(err, ErrCategoryNotFound):
		refuseBlog(c, http.StatusNotFound, BlogErrorCodeNotFound, "No such blog category.")
	case errors.Is(err, ErrAccountUnverified):
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"That account has not verified its email yet.", "handle")
	case errors.Is(err, ErrIncompleteOrder):
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"Name every one of them exactly once to set the order.")
	case errors.Is(err, storage.ErrInsufficientSpace):
		refuseBlog(c, http.StatusServiceUnavailable, BlogErrorCodeServerError,
			"Uploads are temporarily unavailable because storage is low.")
	default:
		refuseBlog(c, http.StatusInternalServerError, BlogErrorCodeServerError,
			"Could not change the blog.")
	}
}

func (h *Handlers) blogAdmin(c *gin.Context, action string) (accountIdentity, bool) {
	current, ok := api.Verified(c, action)
	if !ok {
		return accountIdentity{}, false
	}
	held, err := h.blog.HoldsAuthority(c.Request.Context(), current.ID)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not check who administers the blog.")
		return accountIdentity{}, false
	}
	if !held {
		api.Refuse(c, http.StatusForbidden, "Only the account that administers the blog can do that.")
		return accountIdentity{}, false
	}
	return accountIdentity{ID: current.ID, Handle: current.Handle}, true
}

type accountIdentity struct {
	ID     uuid.UUID
	Handle string
}

func toAPICategories(sorted []Category) []BlogCategory {
	listed := make([]BlogCategory, 0, len(sorted))
	for _, one := range sorted {
		listed = append(listed, toAPICategory(one))
	}
	return listed
}

func toAPICategory(found Category) BlogCategory {
	return BlogCategory{
		Id:       found.ID,
		Slug:     found.Slug,
		Label:    found.Label,
		Position: found.Position,
		Retired:  found.Retired,
	}
}

func (h *Handlers) withProfiles(c *gin.Context, found []Writer) ([]WriterResponse, error) {
	listed := make([]WriterResponse, 0, len(found))
	for _, one := range found {
		shown, err := h.accounts.PublicProfile(c.Request.Context(), one.Handle)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read a writer.")
			return nil, err
		}
		listed = append(listed, toAPIWriter(one, shown))
	}
	return listed, nil
}

func toAPIWriter(found Writer, shown account.PublicProfile) WriterResponse {
	writer := WriterResponse{
		AccountId:   found.AccountID,
		Handle:      found.Handle,
		DisplayName: shown.DisplayName,
		Restricted:  shown.Restricted,
		Since:       found.Since,
	}
	if shown.Avatar != nil {
		writer.Avatar = &profile.ProfileAvatar{
			Url:    account.AvatarURL(shown.Avatar.MediaID, shown.Avatar.DerivativeVersion),
			Width:  shown.Avatar.Width,
			Height: shown.Avatar.Height,
		}
	}
	return writer
}

type BlogCategory struct {
	Id       uuid.UUID `json:"id"`
	Label    string    `json:"label"`
	Position int       `json:"position"`
	Retired  bool      `json:"retired"`
	Slug     string    `json:"slug"`
}

type BlogCategoryList struct {
	Categories []BlogCategory `json:"categories"`
}

type BlogWorkspace struct {
	Admin      bool           `json:"admin"`
	Categories []BlogCategory `json:"categories"`
	Handle     string         `json:"handle"`
	Writer     bool           `json:"writer"`
}

type OrderBlogCategoriesRequest struct {
	CategoryIds []uuid.UUID `json:"categoryIds"`
}

type SwitchWriterOnRequest struct {
	Handle string `json:"handle"`
}

type UpdateBlogCategoryRequest struct {
	Label   *string `json:"label,omitempty"`
	Retired *bool   `json:"retired,omitempty"`
}

type WriterList struct {
	Writers []WriterResponse `json:"writers"`
}

type WriterResponse struct {
	AccountId   uuid.UUID              `json:"accountId"`
	Avatar      *profile.ProfileAvatar `json:"avatar,omitempty"`
	DisplayName string                 `json:"displayName"`
	Handle      string                 `json:"handle"`
	Restricted  bool                   `json:"restricted"`
	Since       time.Time              `json:"since"`
}
