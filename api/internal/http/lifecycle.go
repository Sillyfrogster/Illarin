package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/assetdestination"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) PublishAssetUpdate(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	workingCopyVersion, ok := api.WorkingCopyVersion(c)
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
	if work.CandidateResult(c, candidate, err) {
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
		notReady := work.PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, work.PublishRefusal{
			Error:     "This asset is not ready to publish yet.",
			Code:      &notReady,
			Readiness: work.ToReadiness(items),
		})
	case errors.Is(err, asset.ErrNothingToPublish):
		unchanged := work.PublishRefusalCodeNoChanges
		c.JSON(http.StatusConflict, work.PublishRefusal{
			Error: "Nothing has changed since the last update.", Code: &unchanged,
		})
	case errors.Is(err, asset.ErrAssetIsDraft):
		c.JSON(http.StatusConflict, work.PublishRefusal{Error: "Publish this draft before updating it."})
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
