package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime/types"
)

const restrictedOwnerMessage = "An admin has restricted your public profile. Contact Illarin to have it reviewed."

func (h *Handlers) SavePublicProfile(c *gin.Context) {
	owner, ok := h.verifiedAccount(c, "editing your public profile")
	if !ok {
		return
	}
	var request SaveProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the profile fields as JSON."})
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
		h.profileError(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) SetProfileAvatar(c *gin.Context) {
	owner, ok := h.verifiedAccount(c, "changing your avatar")
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
	saved, err := h.accounts.SetAvatar(c.Request.Context(), owner, limitedFile)
	if errors.Is(err, account.ErrProfileRestricted) {
		h.profileError(c, err)
		return
	}
	if err != nil {
		h.refuseProfile(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) RemoveProfileAvatar(c *gin.Context) {
	owner, ok := h.verifiedAccount(c, "changing your avatar")
	if !ok {
		return
	}
	saved, err := h.accounts.RemoveAvatar(c.Request.Context(), owner)
	if errors.Is(err, account.ErrAvatarMissing) {
		c.JSON(http.StatusNotFound, gin.H{"error": "There is no avatar to remove."})
		return
	}
	if err != nil {
		h.profileError(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) profileError(c *gin.Context, err error) {
	var field account.FieldError
	switch {
	case errors.As(err, &field):
		c.JSON(http.StatusBadRequest, gin.H{"error": field.Message, "field": field.Field})
	case errors.Is(err, account.ErrProfileRestricted):
		c.JSON(http.StatusForbidden, gin.H{"error": restrictedOwnerMessage})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the profile."})
	}
}

func (h *Handlers) refuseProfile(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "That image is larger than the upload limit.",
		})
	case errors.Is(err, storage.ErrInsufficientSpace):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Uploads are temporarily unavailable because storage is low.",
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "That image could not be read. Use a PNG, JPEG, WebP or GIF.",
		})
	}
}

func (h *Handlers) showProfile(c *gin.Context, found account.PublicProfile) {
	if found.Restricted {
		c.JSON(http.StatusOK, toAPIProfile(found, publication.Showcase{}))
		return
	}
	shown, err := h.publications.Showcase(c.Request.Context(), found.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the profile."})
		return
	}
	c.JSON(http.StatusOK, toAPIProfile(found, shown))
}

func toAPIProfile(found account.PublicProfile, shown publication.Showcase) Profile {
	links := make([]ProfileLink, 0, len(found.Links))
	for _, link := range found.Links {
		links = append(links, ProfileLink{Label: link.Label, Address: link.Address})
	}
	profile := Profile{
		Id:           types.UUID(found.ID),
		Handle:       found.Handle,
		DisplayName:  found.DisplayName,
		Biography:    found.Biography,
		ContactEmail: found.ContactEmail,
		Links:        links,
		Positions:    toAPIProfileDistinctions(shown.Positions),
		Titles:       toAPIProfileDistinctions(shown.Titles),
		Badges:       toAPIProfileDistinctions(shown.Badges),
		Restricted:   found.Restricted,
	}
	if found.Avatar != nil {
		profile.Avatar = &ProfileAvatar{
			Url:    account.AvatarURL(found.Avatar.MediaID, found.Avatar.DerivativeVersion),
			Width:  found.Avatar.Width,
			Height: found.Avatar.Height,
		}
	}
	return profile
}
