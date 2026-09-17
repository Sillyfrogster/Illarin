package profile

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
)

const restrictedOwnerMessage = "An admin has restricted your public profile. Contact Illarin to have it reviewed."

func (h *Handlers) GetProfile(c *gin.Context) {
	handle := c.Param("handle")
	found, err := h.accounts.PublicProfile(c.Request.Context(), handle)
	if errors.Is(err, account.ErrProfileNotFound) {
		api.Refuse(c, http.StatusNotFound, "No such profile.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
		return
	}
	showProfile(c, found)
}

func (h *Handlers) SavePublicProfile(c *gin.Context) {
	owner, ok := api.Verified(c, "editing your public profile")
	if !ok {
		return
	}
	var request SaveProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the profile fields as JSON.")
		return
	}
	links := make([]account.ProfileLink, 0, len(request.Links))
	for _, link := range request.Links {
		links = append(links, account.ProfileLink{Label: link.Label, Address: link.Address})
	}
	saved, err := h.accounts.SaveProfile(c.Request.Context(), owner, account.ProfileEdit{
		DisplayName:  request.DisplayName,
		Biography:    request.Biography,
		ContactEmail: request.ContactEmail,
		Links:        links,
	})
	if err != nil {
		refuseProfile(c, err)
		return
	}
	showProfile(c, saved)
}

func (h *Handlers) SetProfileAvatar(c *gin.Context) {
	owner, ok := api.Verified(c, "changing your avatar")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		refuseImage(c, err)
		return
	}
	file, err := parts.NextPart()
	if err != nil || file.FormName() != "file" {
		refuseImage(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	saved, err := h.accounts.SetAvatar(c.Request.Context(), owner, limitedFile)
	if errors.Is(err, account.ErrProfileRestricted) {
		refuseProfile(c, err)
		return
	}
	if err != nil {
		refuseImage(c, err)
		return
	}
	showProfile(c, saved)
}

func (h *Handlers) RemoveProfileAvatar(c *gin.Context) {
	owner, ok := api.Verified(c, "changing your avatar")
	if !ok {
		return
	}
	saved, err := h.accounts.RemoveAvatar(c.Request.Context(), owner)
	if errors.Is(err, account.ErrAvatarMissing) {
		api.Refuse(c, http.StatusNotFound, "There is no avatar to remove.")
		return
	}
	if err != nil {
		refuseProfile(c, err)
		return
	}
	showProfile(c, saved)
}

func refuseProfile(c *gin.Context, err error) {
	var field account.FieldError
	switch {
	case errors.As(err, &field):
		api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
	case errors.Is(err, account.ErrProfileRestricted):
		api.Refuse(c, http.StatusForbidden, restrictedOwnerMessage)
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the profile.")
	}
}

func refuseImage(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		api.Refuse(c, http.StatusRequestEntityTooLarge, "That image is larger than the upload limit.")
	case errors.Is(err, storage.ErrInsufficientSpace):
		api.Refuse(c, http.StatusServiceUnavailable, "Uploads are temporarily unavailable because storage is low.")
	default:
		api.Refuse(c, http.StatusBadRequest, "That image could not be read. Use a PNG, JPEG, WebP or GIF.")
	}
}

func showProfile(c *gin.Context, found account.PublicProfile) {
	c.JSON(http.StatusOK, toAPIProfile(found))
}

func toAPIProfile(found account.PublicProfile) Profile {
	links := make([]ProfileLink, 0, len(found.Links))
	for _, link := range found.Links {
		links = append(links, ProfileLink{Label: link.Label, Address: link.Address})
	}
	shown := Profile{
		Id:           found.ID,
		Handle:       found.Handle,
		DisplayName:  found.DisplayName,
		Biography:    found.Biography,
		ContactEmail: found.ContactEmail,
		Links:        links,
		Restricted:   found.Restricted,
	}
	if found.Avatar != nil {
		shown.Avatar = &ProfileAvatar{
			Url:    account.AvatarURL(found.Avatar.MediaID, found.Avatar.DerivativeVersion),
			Width:  found.Avatar.Width,
			Height: found.Avatar.Height,
		}
	}
	return shown
}
