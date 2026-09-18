package version

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	candidate := &work.Candidate{Version: workingCopyVersion}
	recorded, items, err := h.versions.PublishUpdate(c.Request.Context(), UpdateRequest{
		OwnerID: owner.ID, AssetID: id, Summary: request.Summary,
		Notes: valueOrEmpty(request.Notes), VersionLabel: valueOrEmpty(request.VersionLabel),
		Announcement: announcementChoice(request),
	}, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, ErrUnlistedConsentRequired):
		refuseInvalid(c, "announceUnlisted",
			"This asset is unlisted. Confirm that its direct link may be sent, or publish quietly.")
	case errors.Is(err, ErrUpdateDestinationIneligible):
		refuseInvalid(c, "destinationIds", "Choose only your own verified, active destinations.")
	case errors.Is(err, ErrSummaryRequired):
		api.Refuse(c, http.StatusBadRequest, "Say what changed in this update.")
	case errors.Is(err, ErrSummaryTooLong):
		api.Refuse(c, http.StatusBadRequest, "The summary, notes or version label is too long.")
	case errors.Is(err, work.ErrPublishFloor):
		notReady := page.PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, page.PublishRefusal{
			Error:     "This asset is not ready to publish yet.",
			Code:      &notReady,
			Readiness: page.ToReadiness(items),
		})
	case errors.Is(err, ErrNothingToPublish):
		unchanged := page.PublishRefusalCodeNoChanges
		c.JSON(http.StatusConflict, page.PublishRefusal{
			Error: "Nothing has changed since the last update.", Code: &unchanged,
		})
	case errors.Is(err, work.ErrAssetIsDraft):
		c.JSON(http.StatusConflict, page.PublishRefusal{Error: "Publish this draft before updating it."})
	case errors.Is(err, work.ErrNotFound):
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

func announcementChoice(request AssetUpdateRequest) UpdateAnnouncement {
	choice := UpdateAnnouncement{Notify: request.Notify == nil || *request.Notify}
	if request.DestinationIds != nil {
		chosen := append([]uuid.UUID(nil), *request.DestinationIds...)
		choice.DestinationIDs = &chosen
	}
	if request.AnnounceUnlisted != nil {
		choice.AnnounceUnlisted = *request.AnnounceUnlisted
	}
	return choice
}

// refuseInvalid answers with the error body the blog's publish refusals use, naming the field at fault
func refuseInvalid(c *gin.Context, field, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": message, "code": "invalid", "field": field})
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
