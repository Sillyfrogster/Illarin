package blog

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) SchedulePost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "scheduling a post")
	if !ok {
		return
	}
	var request SchedulePostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"Send the version and the time it goes live, with an explicit offset.", "at")
		return
	}
	scheduled, err := h.blog.SchedulePost(
		c.Request.Context(), editor, id, request.Version, request.At,
		announcementOf(request.Discord),
	)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toAPIPost(scheduled))
}

func (h *Handlers) ReplacePostSchedule(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "replacing a scheduled post")
	if !ok {
		return
	}
	var request ReplacePostScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid,
			"Name the edition and the time it goes live, with an explicit offset.", "at")
		return
	}
	replaced, err := h.blog.ReplaceSchedule(
		c.Request.Context(), editor, id, request.RevisionId, request.At,
		announcementOf(request.Discord),
	)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(replaced))
}

func (h *Handlers) CancelPostSchedule(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "cancelling a scheduled post")
	if !ok {
		return
	}
	cancelled, err := h.blog.CancelSchedule(c.Request.Context(), editor, id)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(cancelled))
}

func (h *Handlers) scheduleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrScheduleNotFound):
		refuseBlog(c, http.StatusNotFound, BlogErrorCodeNotFound,
			"This post has nothing waiting to publish.")
	case errors.Is(err, ErrAlreadyScheduled):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This post already has a scheduled revision. Change that revision instead.",
			Code:  BlogErrorCodeAlreadyScheduled,
		})
	default:
		h.postError(c, err)
	}
}

func toAPISchedule(found *Schedule) *PostSchedule {
	if found == nil {
		return nil
	}
	shown := &PostSchedule{
		Id:             found.ID,
		RevisionId:     found.RevisionID,
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

type PostSchedule struct {
	At             time.Time         `json:"at"`
	CreatedAt      time.Time         `json:"createdAt"`
	CreatedBy      string            `json:"createdBy"`
	Id             uuid.UUID         `json:"id"`
	RevisionId     uuid.UUID         `json:"revisionId"`
	RevisionNumber int               `json:"revisionNumber"`
	State          PostScheduleState `json:"state"`
	StoppedBecause *string           `json:"stoppedBecause,omitempty"`
}

type PostScheduleState string

const (
	PostScheduleStateCancelled  PostScheduleState = "cancelled"
	PostScheduleStatePending    PostScheduleState = "pending"
	PostScheduleStatePublished  PostScheduleState = "published"
	PostScheduleStatePublishing PostScheduleState = "publishing"
	PostScheduleStateStopped    PostScheduleState = "stopped"
)

type ReplacePostScheduleRequest struct {
	At         time.Time `json:"at"`
	Discord    *bool     `json:"discord,omitempty"`
	RevisionId uuid.UUID `json:"revisionId"`
}

type SchedulePostRequest struct {
	At      time.Time `json:"at"`
	Discord *bool     `json:"discord,omitempty"`
	Version int       `json:"version"`
}

// announcementOf posts to Discord unless the request turned it off
func announcementOf(discord *bool) Announcement {
	return Announcement{Discord: discord == nil || *discord}
}
