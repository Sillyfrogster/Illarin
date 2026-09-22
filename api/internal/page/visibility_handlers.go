package page

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) SetWorkVisibility(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	var request WorkVisibilityRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil || !request.Visibility.Valid() {
		api.Refuse(c, http.StatusBadRequest, "Choose listed or unlisted.")
		return
	}
	err := h.works.SetVisibility(
		c.Request.Context(), owner.ID, id, work.Visibility(request.Visibility),
	)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "A taken-down work cannot be changed.")
	case errors.Is(err, work.ErrWorkIsDraft):
		api.Refuse(c, http.StatusConflict, "Visibility applies once the work is published.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the visibility. Try again.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) PublishWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "publishing a work")
	if !ok {
		return
	}
	var request PublishWorkRequest
	if c.Request.ContentLength != 0 {
		if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
			api.Refuse(c, http.StatusBadRequest, "The publish request could not be read.")
			return
		}
	}
	candidate := &work.Candidate{Version: version}
	announce := request.Discord == nil || *request.Discord
	items, err := h.works.Publish(c.Request.Context(), owner.ID, id, candidate, announce)
	if CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrPublishFloor):
		notReady := PublishRefusalCodeNotReady
		c.JSON(http.StatusConflict, PublishRefusal{
			Error:     "This draft is not ready to publish yet.",
			Code:      &notReady,
			Readiness: ToReadiness(items),
		})
		return
	case errors.Is(err, ErrAlreadyPublished):
		published := PublishRefusalCodeAlreadyPublished
		c.JSON(http.StatusConflict, PublishRefusal{
			Error: "This work is already published.", Code: &published,
		})
		return
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such draft.")
		return
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not publish the work.")
		return
	}

	preference, ok := ReaderNSFWPreference(c, h.accounts, nil)
	if !ok {
		return
	}
	found, err := h.works.Detail(c.Request.Context(), id, &owner.ID, preference)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published work.")
		return
	}
	page, err := ToPage(found, preference)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the published work.")
		return
	}
	c.JSON(http.StatusOK, page)
}

func ToReadiness(items []work.ReadinessItem) *[]ReadinessItem {
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
