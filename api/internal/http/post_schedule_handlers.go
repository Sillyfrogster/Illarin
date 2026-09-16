package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) SchedulePost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "scheduling a post")
	if !ok {
		return
	}
	var request SchedulePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send the version and the time it goes live, with an explicit offset.", "at")
		return
	}
	scheduled, err := h.publications.SchedulePost(
		c.Request.Context(), editor, uuid.UUID(id), request.Version, request.At,
		announcementOf(request.DestinationIds, request.RoleDestinationIds, request.Note),
	)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toAPIPost(scheduled))
}

func (h *Handlers) ReplacePostSchedule(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "replacing a scheduled post")
	if !ok {
		return
	}
	var request ReplacePostScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Name the edition and the time it goes live, with an explicit offset.", "at")
		return
	}
	replaced, err := h.publications.ReplaceSchedule(
		c.Request.Context(), editor, uuid.UUID(id), uuid.UUID(request.RevisionId), request.At,
		announcementOf(request.DestinationIds, request.RoleDestinationIds, request.Note),
	)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(replaced))
}

func (h *Handlers) CancelPostSchedule(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "cancelling a scheduled post")
	if !ok {
		return
	}
	cancelled, err := h.publications.CancelSchedule(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(cancelled))
}

func (h *Handlers) scheduleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, publication.ErrScheduleNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound,
			"This post has nothing waiting to publish.")
	case errors.Is(err, publication.ErrAlreadyScheduled):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This post already has a scheduled revision. Change that revision instead.",
			Code:  CodeAlreadyScheduled,
		})
	default:
		h.postError(c, err)
	}
}

func toAPISchedule(found *publication.Schedule) *PostSchedule {
	if found == nil {
		return nil
	}
	shown := &PostSchedule{
		Id:             types.UUID(found.ID),
		RevisionId:     types.UUID(found.RevisionID),
		RevisionNumber: found.RevisionNumber,
		At:             found.At,
		State:          PostScheduleState(found.State),
		CreatedBy:      found.CreatedBy,
		CreatedAt:      found.CreatedAt,
	}
	if found.StoppedBecause != "" {
		shown.StoppedBecause = pointer(found.StoppedBecause)
	}
	return shown
}
