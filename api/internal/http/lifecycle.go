package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

type assetIdentityInput struct {
	Name   string  `json:"name"`
	Blurb  *string `json:"blurb"`
	IsNsfw *bool   `json:"isNsfw"`
}

func (h *Handlers) SetAssetIdentity(c *gin.Context, id types.UUID, params SetAssetIdentityParams) {
	owner, ok := h.verifiedAccount(c, "saving an asset")
	if !ok {
		return
	}
	var request assetIdentityInput
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Send a name, a blurb, and an adult content answer of true, false or null.",
		})
		return
	}
	if request.Blurb == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Send a blurb. Use an empty string to clear it.", "field": "blurb",
		})
		return
	}
	candidate := &asset.Candidate{Version: params.XWorkingCopyVersion}
	err := h.assets.SetIdentity(c.Request.Context(), asset.Identity{
		OwnerID: owner.ID, AssetID: uuid.UUID(id),
		Name: request.Name, Blurb: *request.Blurb, IsNSFW: request.IsNsfw,
	}, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case errors.Is(err, asset.ErrNameTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "The name is too long."})
	case errors.Is(err, asset.ErrBlurbTooLong):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "The blurb must be 400 characters or fewer.", "field": "blurb",
		})
	case errors.Is(err, asset.ErrRatingUnanswerable):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "A published asset needs an adult content answer.",
		})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the details."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) PublishAsset(c *gin.Context, id types.UUID, params PublishAssetParams) {
	owner, ok := h.verifiedAccount(c, "publishing an asset")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: params.XWorkingCopyVersion}
	items, err := h.assets.Publish(c.Request.Context(), owner.ID, uuid.UUID(id), candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrPublishFloor):
		notReady := NotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This draft is not ready to publish yet.",
			Code:      &notReady,
			Readiness: toAPIReadiness(items),
		})
		return
	case errors.Is(err, asset.ErrAlreadyPublished):
		published := AlreadyPublished
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "This asset is already published.", Code: &published,
		})
		return
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such draft."})
		return
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not publish the asset."})
		return
	}

	visibility, ok := h.readerVisibility(c, nil)
	if !ok {
		return
	}
	found, err := h.assets.Detail(c.Request.Context(), uuid.UUID(id), &owner.ID, visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the published asset."})
		return
	}
	page, err := toAPIDetail(found, visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the published asset."})
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
			blockID := types.UUID(*item.BlockID)
			served.BlockId = &blockID
		}
		out = append(out, served)
	}
	return &out
}

func (h *Handlers) PublishAssetUpdate(c *gin.Context, id types.UUID, params PublishAssetUpdateParams) {
	owner, ok := h.verifiedAccount(c, "publishing an asset update")
	if !ok {
		return
	}
	var request AssetUpdateRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Send a summary of what changed, and any notes with it.",
		})
		return
	}
	candidate := &asset.Candidate{Version: params.XWorkingCopyVersion}
	recorded, items, err := h.assets.PublishUpdate(c.Request.Context(), asset.UpdateRequest{
		OwnerID: owner.ID, AssetID: uuid.UUID(id), Summary: request.Summary,
		Notes: valueOrEmpty(request.Notes), VersionLabel: valueOrEmpty(request.VersionLabel),
		Announcement: announcementChoice(request),
	}, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, assetdestination.ErrUnlistedConsentRequired):
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"This asset is unlisted. Confirm that its direct link may be sent, or publish quietly.",
			"announceUnlisted")
	case errors.Is(err, asset.ErrUpdateDestinationIneligible):
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Choose only your own verified, active destinations.", "destinationIds")
	case errors.Is(err, asset.ErrSummaryRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Say what changed in this update."})
	case errors.Is(err, asset.ErrSummaryTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "The summary, notes or version label is too long."})
	case errors.Is(err, asset.ErrPublishFloor):
		notReady := NotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This asset is not ready to publish yet.",
			Code:      &notReady,
			Readiness: toAPIReadiness(items),
		})
	case errors.Is(err, asset.ErrNothingToPublish):
		unchanged := NoChanges
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "Nothing has changed since the last update.", Code: &unchanged,
		})
	case errors.Is(err, asset.ErrAssetIsDraft):
		c.JSON(http.StatusConflict, PublishRefusal{Error: "Publish this draft before updating it."})
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not publish the update."})
	default:
		c.JSON(http.StatusOK, AssetUpdate{
			Id: types.UUID(recorded.ID), Number: recorded.Number,
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
