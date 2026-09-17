package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/gin-gonic/gin"
)

type assetIdentityInput struct {
	Name   string  `json:"name"`
	Blurb  *string `json:"blurb"`
	IsNsfw *bool   `json:"isNsfw"`
}

func (h *Handlers) SetAssetIdentity(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "saving an asset")
	if !ok {
		return
	}
	var request assetIdentityInput
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a name, a blurb, and an adult content answer of true, false or null.")
		return
	}
	if request.Blurb == nil {
		api.RefuseField(c, http.StatusBadRequest, "blurb", "Send a blurb. Use an empty string to clear it.")
		return
	}
	candidate := &asset.Candidate{Version: version}
	err := h.assets.SetIdentity(c.Request.Context(), asset.Identity{
		OwnerID: owner.ID, AssetID: id,
		Name: request.Name, Blurb: *request.Blurb, IsNSFW: request.IsNsfw,
	}, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such asset.")
	case errors.Is(err, asset.ErrNameTooLong):
		api.Refuse(c, http.StatusBadRequest, "The name is too long.")
	case errors.Is(err, asset.ErrBlurbTooLong):
		api.RefuseField(c, http.StatusBadRequest, "blurb", "The blurb must be 400 characters or fewer.")
	case errors.Is(err, asset.ErrRatingUnanswerable):
		api.Refuse(c, http.StatusBadRequest, "A published asset needs an adult content answer.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the details.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) PublishAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "publishing an asset")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: version}
	items, err := h.assets.Publish(c.Request.Context(), owner.ID, id, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrPublishFloor):
		notReady := PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This draft is not ready to publish yet.",
			Code:      &notReady,
			Readiness: toAPIReadiness(items),
		})
		return
	case errors.Is(err, asset.ErrAlreadyPublished):
		published := PublishRefusalCodeAlreadyPublished
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "This asset is already published.", Code: &published,
		})
		return
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such draft.")
		return
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not publish the asset.")
		return
	}

	visibility, ok := h.readerVisibility(c, nil)
	if !ok {
		return
	}
	found, err := h.assets.Detail(c.Request.Context(), id, &owner.ID, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published asset.")
		return
	}
	page, err := toAPIDetail(found, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published asset.")
		return
	}
	c.JSON(http.StatusOK, page)
}

func toAPIReadiness(items []asset.ReadinessItem) *[]ReadinessItem {
	if items == nil {
		return nil
	}
	out := make([]ReadinessItem, 0, len(items))
	for _, item := range items {
		served := ReadinessItem{
			Id: item.ID, Label: item.Label, Detail: item.Detail, Met: item.Met,
		}
		if item.BlockID != nil {
			blockID := *item.BlockID
			served.BlockId = &blockID
		}
		out = append(out, served)
	}
	return &out
}

func (h *Handlers) PublishAssetUpdate(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	workingCopyVersion, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "publishing an asset update")
	if !ok {
		return
	}
	var request AssetUpdateRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a summary of what changed, and any notes with it.")
		return
	}
	candidate := &asset.Candidate{Version: workingCopyVersion}
	recorded, items, err := h.assets.PublishUpdate(c.Request.Context(), asset.UpdateRequest{
		OwnerID: owner.ID, AssetID: id, Summary: request.Summary,
		Notes: valueOrEmpty(request.Notes), VersionLabel: valueOrEmpty(request.VersionLabel),
		Announcement: announcementChoice(request),
	}, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, assetdestination.ErrUnlistedConsentRequired):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"This asset is unlisted. Confirm that its direct link may be sent, or publish quietly.",
			"announceUnlisted")
	case errors.Is(err, asset.ErrUpdateDestinationIneligible):
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Choose only your own verified, active destinations.", "destinationIds")
	case errors.Is(err, asset.ErrSummaryRequired):
		api.Refuse(c, http.StatusBadRequest, "Say what changed in this update.")
	case errors.Is(err, asset.ErrSummaryTooLong):
		api.Refuse(c, http.StatusBadRequest, "The summary, notes or version label is too long.")
	case errors.Is(err, asset.ErrPublishFloor):
		notReady := PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This asset is not ready to publish yet.",
			Code:      &notReady,
			Readiness: toAPIReadiness(items),
		})
	case errors.Is(err, asset.ErrNothingToPublish):
		unchanged := PublishRefusalCodeNoChanges
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "Nothing has changed since the last update.", Code: &unchanged,
		})
	case errors.Is(err, asset.ErrAssetIsDraft):
		c.JSON(http.StatusConflict, PublishRefusal{Error: "Publish this draft before updating it."})
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such asset.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not publish the update.")
	default:
		c.JSON(http.StatusOK, AssetUpdate{
			Id: recorded.ID, Number: recorded.Number,
			RecordedAt: recorded.RecordedAt, VersionLabel: recorded.VersionLabel,
			Summary: recorded.Summary, Notes: recorded.Notes,
			ContentGeneration: recorded.ContentGeneration,
			ContentChanged:    recorded.ContentChanged,
		})
	}
}

func announcementChoice(request AssetUpdateRequest) asset.UpdateAnnouncement {
	choice := asset.UpdateAnnouncement{Notify: request.Notify == nil || *request.Notify}
	if request.DestinationIds != nil {
		chosen := readIDs(request.DestinationIds)
		choice.DestinationIDs = &chosen
	}
	if request.AnnounceUnlisted != nil {
		choice.AnnounceUnlisted = *request.AnnounceUnlisted
	}
	return choice
}
