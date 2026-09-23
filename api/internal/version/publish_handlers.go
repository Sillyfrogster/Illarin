package version

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) PublishWorkVersion(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	draftedChangesVersion, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "publishing a version")
	if !ok {
		return
	}
	var request WorkVersionRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a summary of what changed, and any notes with it.")
		return
	}
	candidate := &work.Candidate{Version: draftedChangesVersion}
	recorded, items, err := h.versions.PublishVersion(c.Request.Context(), PublishRequest{
		OwnerID: owner.ID, WorkID: id, Summary: request.Summary,
		Notes: valueOrEmpty(request.Notes), VersionLabel: valueOrEmpty(request.VersionLabel),
		Announcement: announcementChoice(request),
	}, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, ErrSummaryRequired):
		api.Refuse(c, http.StatusBadRequest, "Say what changed in this version.")
	case errors.Is(err, ErrSummaryTooLong):
		api.Refuse(c, http.StatusBadRequest, "The summary, notes or version label is too long.")
	case errors.Is(err, work.ErrPublishFloor):
		notReady := page.PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, page.PublishRefusal{
			Error:     "This work is not ready to publish yet.",
			Code:      &notReady,
			Readiness: page.ToReadiness(items),
		})
	case errors.Is(err, ErrNothingToPublish):
		unchanged := page.PublishRefusalCodeNoChanges
		c.JSON(http.StatusConflict, page.PublishRefusal{
			Error: "Nothing has changed since the last version.", Code: &unchanged,
		})
	case errors.Is(err, work.ErrWorkIsDraft):
		c.JSON(http.StatusConflict, page.PublishRefusal{Error: "Publish this draft before publishing a version of it."})
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not publish the version.")
	default:
		c.JSON(http.StatusOK, WorkVersion{
			Id: recorded.ID, Number: recorded.Number,
			RecordedAt: recorded.RecordedAt, VersionLabel: recorded.VersionLabel,
			Summary: recorded.Summary, Notes: recorded.Notes,
			ContentChanged: recorded.ContentChanged,
		})
	}
}

func announcementChoice(request WorkVersionRequest) Announcement {
	return Announcement{
		Notify:  request.Notify == nil || *request.Notify,
		Discord: request.Discord == nil || *request.Discord,
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
