package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListPublicationApps(c *gin.Context) {
	if _, ok := h.publicationAuthority(c, "reading publication apps"); !ok {
		return
	}
	configured, err := h.publications.Apps(c.Request.Context())
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationAppList{Apps: toAPIApps(configured)})
}

func (h *Handlers) DefinePublicationApp(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "configuring a publication app")
	if !ok {
		return
	}
	var request DefinePublicationAppRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the app as JSON."})
		return
	}
	defined, err := h.publications.DefineApp(c.Request.Context(), authority.ID, publication.AppEdit{
		Slug: request.Slug,
		Name: request.Name,
		Home: request.Home,
	})
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIApp(defined))
}

func (h *Handlers) UpdatePublicationApp(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a publication app")
	if !ok {
		return
	}
	var request UpdatePublicationAppRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the change as JSON."})
		return
	}
	updated, err := h.publications.UpdateApp(
		c.Request.Context(), authority.ID, uuid.UUID(id), publication.AppUpdate{
			Slug:    request.Slug,
			Name:    request.Name,
			Home:    request.Home,
			Retired: request.Retired,
		},
	)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIApp(updated))
}

func (h *Handlers) OrderPublicationApps(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "ordering publication apps")
	if !ok {
		return
	}
	var request OrderPublicationAppsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the order as JSON."})
		return
	}
	ordered, err := h.publications.OrderApps(
		c.Request.Context(), authority.ID, toUUIDs(request.AppIds),
	)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationAppList{Apps: toAPIApps(ordered)})
}

func (h *Handlers) SetPublicationAppMark(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a publication app mark")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refuseProfile(c, refusal{reason: "send the image as form data", cause: err})
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refuseProfile(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	marked, err := h.publications.SetAppMark(
		c.Request.Context(), authority.ID, uuid.UUID(id), limitedFile,
	)
	if errors.Is(err, publication.ErrAppNotFound) {
		h.publicationError(c, err)
		return
	}
	if err != nil {
		h.refuseProfile(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIApp(marked))
}

func (h *Handlers) ListPublicationCategories(c *gin.Context) {
	if _, ok := h.publicationAuthority(c, "reading publication categories"); !ok {
		return
	}
	sorted, err := h.publications.Categories(c.Request.Context())
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationCategoryList{Categories: toAPICategories(sorted)})
}

func (h *Handlers) UpdatePublicationCategory(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a publication category")
	if !ok {
		return
	}
	var request UpdatePublicationCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the change as JSON."})
		return
	}
	updated, err := h.publications.UpdateCategory(
		c.Request.Context(), authority.ID, uuid.UUID(id), publication.CategoryUpdate{
			Label:   request.Label,
			Retired: request.Retired,
		},
	)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPICategory(updated))
}

func (h *Handlers) OrderPublicationCategories(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "ordering publication categories")
	if !ok {
		return
	}
	var request OrderPublicationCategoriesRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the order as JSON."})
		return
	}
	ordered, err := h.publications.OrderCategories(
		c.Request.Context(), authority.ID, toUUIDs(request.CategoryIds),
	)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationCategoryList{Categories: toAPICategories(ordered)})
}

func (h *Handlers) ListPublicationGrants(c *gin.Context) {
	if _, ok := h.publicationAuthority(c, "reading publication grants"); !ok {
		return
	}
	made, err := h.publications.Grants(c.Request.Context())
	if err != nil {
		h.publicationError(c, err)
		return
	}
	listed, err := h.withHolders(c, made)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, PublicationGrantList{Grants: listed})
}

func (h *Handlers) CreatePublicationGrant(c *gin.Context) {
	authority, ok := h.publicationAuthority(c, "approving a contributor")
	if !ok {
		return
	}
	var request CreatePublicationGrantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the approval as JSON."})
		return
	}
	made, err := h.publications.CreateGrant(c.Request.Context(), authority.ID, publication.GrantEdit{
		Handle:            request.Handle,
		AppID:             uuid.UUID(request.AppId),
		CategoryIDs:       toUUIDs(request.CategoryIds),
		DefaultCategoryID: uuid.UUID(request.DefaultCategoryId),
	})
	if err != nil {
		h.publicationError(c, err)
		return
	}
	listed, err := h.withHolders(c, []publication.Grant{made})
	if err != nil {
		return
	}
	c.JSON(http.StatusCreated, listed[0])
}

func (h *Handlers) UpdatePublicationGrant(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "changing a contributor's approval")
	if !ok {
		return
	}
	var request UpdatePublicationGrantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the change as JSON."})
		return
	}
	change := publication.GrantUpdate{DefaultCategoryID: (*uuid.UUID)(request.DefaultCategoryId)}
	if request.CategoryIds != nil {
		change.CategoryIDs = toUUIDs(*request.CategoryIds)
	}
	updated, err := h.publications.UpdateGrant(
		c.Request.Context(), authority.ID, uuid.UUID(id), change,
	)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	listed, err := h.withHolders(c, []publication.Grant{updated})
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, listed[0])
}

func (h *Handlers) RevokePublicationGrant(c *gin.Context, id types.UUID) {
	authority, ok := h.publicationAuthority(c, "revoking a contributor's approval")
	if !ok {
		return
	}
	if err := h.publications.RevokeGrant(c.Request.Context(), authority.ID, uuid.UUID(id)); err != nil {
		h.publicationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetPublicationWorkspace(c *gin.Context) {
	current, ok := h.verifiedAccount(c, "opening the publication workspace")
	if !ok {
		return
	}
	held, err := h.publications.Workspace(c.Request.Context(), current.ID)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	listed, err := h.withHolders(c, held)
	if err != nil {
		return
	}
	admin := current.Role == account.RoleAdmin
	open, err := h.publications.WritableCategories(c.Request.Context(), held, admin)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	named, err := h.publications.NameableApps(c.Request.Context(), held, admin)
	if err != nil {
		h.publicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationWorkspace{
		Handle:     current.Handle,
		Admin:      admin,
		Grants:     listed,
		Categories: toAPICategories(open),
		Apps:       toAPIApps(named),
	})
}

func (h *Handlers) publicationError(c *gin.Context, err error) {
	var field publication.FieldError
	switch {
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, categoryOr(err, CodeInvalid),
			field.Message, field.Field)
	case errors.Is(err, publication.ErrAccountNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such account.")
	case errors.Is(err, publication.ErrAppNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such publication app.")
	case errors.Is(err, publication.ErrCategoryNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such publication category.")
	case errors.Is(err, publication.ErrGrantNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such approval.")
	case errors.Is(err, publication.ErrGrantRevoked):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"That approval has been revoked.")
	case errors.Is(err, publication.ErrAccountUnverified):
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"That account has not verified its email yet.", "handle")
	case errors.Is(err, publication.ErrSlugTaken):
		refuseField(c, http.StatusConflict, CodeInvalid,
			"Another app already uses that slug.", "slug")
	case errors.Is(err, publication.ErrAlreadyGranted):
		refusePublication(c, http.StatusConflict, CodeInvalid,
			"That account already publishes for that app.")
	case errors.Is(err, publication.ErrIncompleteOrder):
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"Name every one of them exactly once to set the order.")
	case errors.Is(err, storage.ErrInsufficientSpace):
		refusePublication(c, http.StatusServiceUnavailable, CodeServerError,
			"Uploads are temporarily unavailable because storage is low.")
	default:
		refusePublication(c, http.StatusInternalServerError, CodeServerError,
			"Could not change the publication.")
	}
}

func categoryOr(err error, otherwise PublicationErrorCode) PublicationErrorCode {
	if errors.Is(err, publication.ErrCategoryRefused) {
		return CodeCategoryRefused
	}
	return otherwise
}

func (h *Handlers) publicationAuthority(c *gin.Context, action string) (accountIdentity, bool) {
	current, ok := h.verifiedAccount(c, action)
	if !ok {
		return accountIdentity{}, false
	}
	held, err := h.publications.HoldsAuthority(c.Request.Context(), current.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check publication authority."})
		return accountIdentity{}, false
	}
	if !held {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only the account designated to manage blog access can do that.",
		})
		return accountIdentity{}, false
	}
	return accountIdentity{ID: current.ID, Handle: current.Handle}, true
}

type accountIdentity struct {
	ID     uuid.UUID
	Handle string
}

func toUUIDs(given []types.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(given))
	for _, id := range given {
		ids = append(ids, uuid.UUID(id))
	}
	return ids
}

func toAPIMark(mark *publication.Mark) *PublicationAppMark {
	if mark == nil {
		return nil
	}
	return &PublicationAppMark{
		Url:    publication.MarkURL(mark.MediaID, mark.DerivativeVersion),
		Width:  mark.Width,
		Height: mark.Height,
	}
}

func toAPIApps(configured []publication.App) []PublicationApp {
	listed := make([]PublicationApp, 0, len(configured))
	for _, one := range configured {
		listed = append(listed, toAPIApp(one))
	}
	return listed
}

func toAPIApp(found publication.App) PublicationApp {
	return PublicationApp{
		Id:           types.UUID(found.ID),
		Slug:         found.Slug,
		Name:         found.Name,
		Home:         found.Home,
		Mark:         toAPIMark(found.Mark),
		Position:     found.Position,
		Retired:      found.Retired,
		Destinations: toAPIChoiceRows(found.Destinations),
	}
}

func toAPICategories(sorted []publication.Category) []PublicationCategory {
	listed := make([]PublicationCategory, 0, len(sorted))
	for _, one := range sorted {
		listed = append(listed, toAPICategory(one))
	}
	return listed
}

func toAPICategory(found publication.Category) PublicationCategory {
	return PublicationCategory{
		Id:       types.UUID(found.ID),
		Slug:     found.Slug,
		Label:    found.Label,
		Position: found.Position,
		Retired:  found.Retired,
	}
}

func (h *Handlers) withHolders(
	c *gin.Context,
	made []publication.Grant,
) ([]PublicationGrant, error) {
	listed := make([]PublicationGrant, 0, len(made))
	for _, one := range made {
		found, err := h.accounts.PublicProfile(c.Request.Context(), one.Holder.Handle)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read a contributor."})
			return nil, err
		}
		listed = append(listed, toAPIGrant(one, found))
	}
	return listed, nil
}

func toAPIGrant(found publication.Grant, holder account.PublicProfile) PublicationGrant {
	shown := PublicationGrantHolder{
		Handle:      found.Holder.Handle,
		DisplayName: holder.DisplayName,
		Restricted:  holder.Restricted,
	}
	if holder.Avatar != nil {
		shown.Avatar = &ProfileAvatar{
			Url:    account.AvatarURL(holder.Avatar.MediaID, holder.Avatar.DerivativeVersion),
			Width:  holder.Avatar.Width,
			Height: holder.Avatar.Height,
		}
	}
	return PublicationGrant{
		Id:                    types.UUID(found.ID),
		Holder:                shown,
		App:                   toAPIApp(found.App),
		Categories:            toAPICategories(found.Categories),
		DefaultCategory:       toAPICategory(found.DefaultCategory),
		Destinations:          toAPIChoiceRows(found.Destinations),
		DestinationsInherited: found.DestinationsInherited,
		GrantedAt:             found.GrantedAt,
		RevokedAt:             found.RevokedAt,
		Active:                found.Active,
	}
}
