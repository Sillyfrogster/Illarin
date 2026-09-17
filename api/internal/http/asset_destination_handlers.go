package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListAssetUpdateDestinations(c *gin.Context) {
	owner, ok := api.SignedIn(c, "reading your update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.List(c.Request.Context(), owner.ID)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	listed := make([]AssetUpdateDestination, 0, len(found))
	for _, one := range found {
		listed = append(listed, toAssetUpdateDestination(one))
	}
	c.JSON(http.StatusOK, AssetUpdateDestinationList{Destinations: listed})
}

func (h *Handlers) GetAssetUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading your update destination")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Get(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAssetUpdateDestination(found))
}

func (h *Handlers) AddAssetUpdateDestination(c *gin.Context) {
	owner, ok := api.Verified(c, "configuring update destinations")
	if !ok {
		return
	}
	var request AddAssetUpdateDestinationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	if request.Address == nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Enter the destination address.", "address")
		return
	}
	added, err := h.updateDestinations.Add(c.Request.Context(), owner.ID, string(request.Kind), request.Name, *request.Address)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	answer := AddedAssetUpdateDestination{Destination: toAssetUpdateDestination(added.Destination)}
	if added.Secret != "" {
		answer.Secret = &added.Secret
	}
	c.JSON(http.StatusCreated, answer)
}

func (h *Handlers) assetDestinationError(c *gin.Context, err error) {
	var field assetdestination.FieldError
	switch {
	case errors.Is(err, assetdestination.ErrNotFound), errors.Is(err, asset.ErrNotFound):
		c.Status(http.StatusNotFound)
	case errors.Is(err, assetdestination.ErrChanged):
		api.Refuse(c, http.StatusConflict, "The destination changed. Check its configuration and try again.")
	case errors.Is(err, asset.ErrAssetFrozen):
		api.Refuse(c, http.StatusConflict, "This asset is frozen while it is withheld.")
	case errors.Is(err, asset.ErrUpdateDestinationIneligible):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Choose only your own verified, active destinations.", "destinationIds")
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, field.Message, field.Field)
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not update your destination. Try again.")
	}
}

func (h *Handlers) VerifyAssetUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "verifying update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Verify(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAssetUpdateDestination(found))
}

func (h *Handlers) UpdateAssetUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "changing update destinations")
	if !ok {
		return
	}
	var request UpdateAssetUpdateDestinationRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	found, err := h.updateDestinations.Update(c.Request.Context(), owner.ID, id, request.Name, request.Address)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAssetUpdateDestination(found))
}

func (h *Handlers) DisableAssetUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "disabling update destinations")
	if !ok {
		return
	}
	found, err := h.updateDestinations.Disable(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAssetUpdateDestination(found))
}

func (h *Handlers) RemoveAssetUpdateDestination(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "removing update destinations")
	if !ok {
		return
	}
	if err := h.updateDestinations.Remove(c.Request.Context(), owner.ID, id); err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RotateAssetUpdateDestinationSecret(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "rotating an update destination's secret")
	if !ok {
		return
	}
	added, err := h.updateDestinations.RotateSecret(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, AddedAssetUpdateDestination{Destination: toAssetUpdateDestination(added.Destination), Secret: &added.Secret})
}

func (h *Handlers) ListAssetUpdateDestinationChoices(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading an asset's update destinations")
	if !ok {
		return
	}
	found, err := h.assets.UpdateDestinations(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	choices := make([]AssetUpdateDestinationChoice, 0, len(found))
	for _, one := range found {
		choices = append(choices, AssetUpdateDestinationChoice{Id: one.ID, Name: one.Name, Kind: AssetUpdateDestinationKind(one.Kind), ByDefault: one.ByDefault})
	}
	c.JSON(http.StatusOK, AssetUpdateDestinationChoices{Destinations: choices})
}

func (h *Handlers) SetAssetUpdateDestinationDefaults(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "choosing an asset's update destinations")
	if !ok {
		return
	}
	var request AssetUpdateDestinationDefaultsRequest
	if !api.ReadBoundedJSON(c, &request, 4096, "The destination selection is too large.") {
		return
	}
	if request.DestinationIds == nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send destination IDs, or an empty list to disable default announcements.", "destinationIds")
		return
	}
	if err := h.assets.SetUpdateDestinations(c.Request.Context(), owner.ID, id, readIDs(&request.DestinationIds)); err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toAssetUpdateDestination(d assetdestination.Destination) AssetUpdateDestination {
	answer := AssetUpdateDestination{
		Id: d.ID, Name: d.Name, Kind: AssetUpdateDestinationKind(d.Kind), Host: d.Host,
		Address: d.Address, State: AssetUpdateDestinationState(d.State), CreatedAt: d.CreatedAt,
		SecretSetAt: d.SecretSetAt, PreviousSecretUntil: d.PreviousSecretUntil,
		VerifiedAt: d.VerifiedAt, DisabledAt: d.DisabledAt,
	}
	if d.GuildID != nil && d.ChannelID != nil {
		answer.Channel = &AssetUpdateChannel{GuildId: *d.GuildID, ChannelId: *d.ChannelID}
	}
	return answer
}

func (h *Handlers) ListAssetUpdateAnnouncements(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading what an asset announced")
	if !ok {
		return
	}
	sent, err := h.updateDestinations.Announcements(c.Request.Context(), owner.ID, id)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	listed := make([]AssetUpdateAnnouncement, 0, len(sent))
	for _, one := range sent {
		listed = append(listed, toAssetUpdateAnnouncement(one))
	}
	c.JSON(http.StatusOK, AssetUpdateAnnouncementList{Announcements: listed})
}

func toAssetUpdateAnnouncement(one assetdestination.Announcement) AssetUpdateAnnouncement {
	shown := AssetUpdateAnnouncement{
		Id: one.ID, EventId: one.EventID, UpdateId: one.UpdateID, UpdateNumber: one.UpdateNumber,
		Destination: one.Destination, Kind: AssetUpdateDestinationKind(one.Kind),
		Removed: one.Removed, State: AssetUpdateAnnouncementState(one.State),
		MessageId: one.MessageID, Run: one.Run, Attempts: one.Attempts,
		OccurredAt: one.OccurredAt, DueAt: one.DueAt, SettledAt: one.SettledAt,
	}
	if one.SettledReason != "" {
		reason := AssetUpdateAnnouncementSettledReason(one.SettledReason)
		shown.SettledReason = &reason
	}
	if one.Last != nil {
		shown.Last = &AssetUpdateAnnouncementAttempt{
			Run: one.Last.Run, Number: one.Last.Number,
			Outcome: AssetUpdateAnnouncementAttemptOutcome(one.Last.Outcome),
			Status:  one.Last.Status, Detail: one.Last.Detail,
			TookMs: int(one.Last.Took.Milliseconds()), AttemptedAt: one.Last.Attempted,
		}
	}
	return shown
}
