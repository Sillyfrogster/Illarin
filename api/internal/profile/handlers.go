package profile

import (
	"context"
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	restrictedOwnerMessage = "An admin has restricted your public profile. Contact Illarin to have it reviewed."
	recentVersionsRead     = 24
)

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
	h.showProfile(c, found)
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
	h.showProfile(c, saved)
}

func (h *Handlers) SaveFeatured(c *gin.Context) {
	owner, ok := api.Verified(c, "choosing your featured works")
	if !ok {
		return
	}
	var request SaveFeaturedRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the featured work ids as JSON.")
		return
	}
	saved, err := h.accounts.SaveFeatured(c.Request.Context(), owner, request.WorkIds)
	if err != nil {
		refuseProfile(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) SetAvatar(c *gin.Context)    { h.setPicture(c, account.Avatar) }
func (h *Handlers) RemoveAvatar(c *gin.Context) { h.removePicture(c, account.Avatar) }
func (h *Handlers) SetBanner(c *gin.Context)    { h.setPicture(c, account.Banner) }
func (h *Handlers) RemoveBanner(c *gin.Context) { h.removePicture(c, account.Banner) }
func (h *Handlers) FollowCreator(c *gin.Context) {
	h.setCreatorFollow(c, h.notifications.FollowCreator)
}
func (h *Handlers) StopFollowingCreator(c *gin.Context) {
	h.setCreatorFollow(c, h.notifications.StopFollowingCreator)
}

func (h *Handlers) setPicture(c *gin.Context, picture account.Picture) {
	owner, ok := api.Verified(c, "changing your "+string(picture))
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
	saved, err := h.accounts.SetPicture(c.Request.Context(), owner, picture, limitedFile)
	if errors.Is(err, account.ErrProfileRestricted) {
		refuseProfile(c, err)
		return
	}
	if err != nil {
		refuseImage(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) removePicture(c *gin.Context, picture account.Picture) {
	owner, ok := api.Verified(c, "changing your "+string(picture))
	if !ok {
		return
	}
	saved, err := h.accounts.RemovePicture(c.Request.Context(), owner, picture)
	if errors.Is(err, account.ErrPictureMissing) {
		api.Refuse(c, http.StatusNotFound, "There is no "+string(picture)+" to remove.")
		return
	}
	if err != nil {
		refuseProfile(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) setCreatorFollow(
	c *gin.Context,
	change func(ctx context.Context, account, creator uuid.UUID) (notify.CreatorFollow, error),
) {
	current, ok := api.SignedIn(c, "following a creator")
	if !ok {
		return
	}
	found, err := h.accounts.PublicProfile(c.Request.Context(), c.Param("handle"))
	if errors.Is(err, account.ErrProfileNotFound) {
		api.Refuse(c, http.StatusNotFound, "No such profile.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
		return
	}
	follow, err := change(c.Request.Context(), current.ID, found.ID)
	switch {
	case errors.Is(err, notify.ErrOwnProfile):
		api.Refuse(c, http.StatusForbidden, "You cannot follow yourself.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not change whether you follow this creator. Try again.")
	default:
		c.JSON(http.StatusOK, CreatorFollow{Followers: follow.Followers, Following: follow.Following != nil && *follow.Following})
	}
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

// showProfile answers with the profile and its portfolio as the viewer may see it
func (h *Handlers) showProfile(c *gin.Context, found account.PublicProfile) {
	ctx := c.Request.Context()
	viewer, ok := api.ViewerID(c)
	if !ok {
		return
	}
	preference, ok := page.ReaderNSFWPreference(c, h.accounts, nil)
	if !ok {
		return
	}
	shown := toAPIProfile(found)
	shown.IsOwner = viewer != nil && *viewer == found.ID
	follow, err := h.notifications.CreatorFollowOf(ctx, viewer, found.ID)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
		return
	}
	shown.Followers, shown.Following = follow.Followers, follow.Following
	if !found.Restricted && len(found.FeaturedWorkIDs) > 0 {
		featured, err := h.pages.Featured(ctx, found.ID, preference)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
			return
		}
		shown.Featured = page.ToAPIBrowseWorks(featured)
	}
	recent, err := h.versions.RecentByCreator(ctx, found.ID, preference, recentVersionsRead)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the profile.")
		return
	}
	for _, one := range recent {
		shown.RecentVersions = append(shown.RecentVersions, toAPIRecentVersion(one))
	}
	c.JSON(http.StatusOK, shown)
}

func toAPIProfile(found account.PublicProfile) Profile {
	links := make([]ProfileLink, 0, len(found.Links))
	for _, link := range found.Links {
		links = append(links, ProfileLink{Label: link.Label, Address: link.Address})
	}
	return Profile{
		Id:             found.ID,
		Handle:         found.Handle,
		DisplayName:    found.DisplayName,
		Biography:      found.Biography,
		ContactEmail:   found.ContactEmail,
		Links:          links,
		Avatar:         toAPIPicture(account.Avatar, found.Avatar),
		Banner:         toAPIPicture(account.Banner, found.Banner),
		Tint:           found.Tint,
		Featured:       []page.BrowseWork{},
		RecentVersions: []RecentVersion{},
		Works:          found.Works,
		Restricted:     found.Restricted,
	}
}

func toAPIPicture(picture account.Picture, stored *account.ProfilePicture) *ProfilePicture {
	if stored == nil {
		return nil
	}
	return &ProfilePicture{
		Url:    account.PictureURL(picture, stored.MediaID, stored.ImageSizeVersion),
		Width:  stored.Width,
		Height: stored.Height,
	}
}

func toAPIRecentVersion(one version.RecentVersion) RecentVersion {
	var cover *string
	if one.CoverURL != "" {
		cover = &one.CoverURL
	}
	return RecentVersion{
		WorkId: one.WorkID, WorkName: one.WorkName, WorkType: one.WorkType,
		Number: one.Number, Initial: one.Initial, VersionLabel: one.VersionLabel,
		Summary: one.Summary, RecordedAt: one.RecordedAt, Cover: cover,
	}
}
