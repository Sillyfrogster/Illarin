package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListAssetUpdateDestinations(c *gin.Context) {
	owner, ok := h.signedInAccount(c, "reading your update destinations")
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

func (h *Handlers) GetAssetUpdateDestination(c *gin.Context, id types.UUID) {
	owner, ok := h.signedInAccount(c, "reading your update destination")
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
	owner, ok := h.verifiedAccount(c, "configuring update destinations")
	if !ok {
		return
	}
	var request AddAssetUpdateDestinationRequest
	if !readBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	if request.Address == nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid, "Enter the destination address.", "address")
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
		c.JSON(http.StatusConflict, gin.H{"error": "The destination changed. Check its configuration and try again."})
	case errors.Is(err, asset.ErrAssetFrozen):
		c.JSON(http.StatusConflict, gin.H{"error": "This asset is frozen while it is withheld."})
	case errors.Is(err, asset.ErrUpdateDestinationIneligible):
		refuseField(c, http.StatusBadRequest, CodeInvalid, "Choose only your own verified, active destinations.", "destinationIds")
	case errors.As(err, &field):
		refuseField(c, http.StatusBadRequest, CodeInvalid, field.Message, field.Field)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update your destination. Try again."})
	}
}

func (h *Handlers) VerifyAssetUpdateDestination(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "verifying update destinations")
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

func (h *Handlers) UpdateAssetUpdateDestination(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "changing update destinations")
	if !ok {
		return
	}
	var request UpdateAssetUpdateDestinationRequest
	if !readBoundedJSON(c, &request, 4096, "The destination configuration is too large.") {
		return
	}
	found, err := h.updateDestinations.Update(c.Request.Context(), owner.ID, id, request.Name, request.Address)
	if err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAssetUpdateDestination(found))
}

func (h *Handlers) DisableAssetUpdateDestination(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "disabling update destinations")
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

func (h *Handlers) RemoveAssetUpdateDestination(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "removing update destinations")
	if !ok {
		return
	}
	if err := h.updateDestinations.Remove(c.Request.Context(), owner.ID, id); err != nil {
		h.assetDestinationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) RotateAssetUpdateDestinationSecret(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "rotating an update destination's secret")
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

func (h *Handlers) ListAssetUpdateDestinationChoices(c *gin.Context, id types.UUID) {
	owner, ok := h.signedInAccount(c, "reading an asset's update destinations")
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

func (h *Handlers) SetAssetUpdateDestinationDefaults(c *gin.Context, id types.UUID) {
	owner, ok := h.verifiedAccount(c, "choosing an asset's update destinations")
	if !ok {
		return
	}
	var request AssetUpdateDestinationDefaultsRequest
	if !readBoundedJSON(c, &request, 4096, "The destination selection is too large.") {
		return
	}
	if request.DestinationIds == nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid, "Send a destination list, or an empty list for quiet defaults.", "destinationIds")
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
