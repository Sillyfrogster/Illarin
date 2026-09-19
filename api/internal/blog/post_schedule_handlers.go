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
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send the version and the time it goes live, with an explicit offset.", "at")
		return
	}
	scheduled, err := h.publications.SchedulePost(
		c.Request.Context(), editor, id, request.Version, request.At,
		announcementOf(request.IntegrationIds, request.RoleIntegrationIds, request.Note),
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
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Name the edition and the time it goes live, with an explicit offset.", "at")
		return
	}
	replaced, err := h.publications.ReplaceSchedule(
		c.Request.Context(), editor, id, request.RevisionId, request.At,
		announcementOf(request.IntegrationIds, request.RoleIntegrationIds, request.Note),
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
	cancelled, err := h.publications.CancelSchedule(c.Request.Context(), editor, id)
	if err != nil {
		h.scheduleError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(cancelled))
}

func (h *Handlers) scheduleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrScheduleNotFound):
		refusePublication(c, http.StatusNotFound, PublicationErrorCodeNotFound,
			"This post has nothing waiting to publish.")
	case errors.Is(err, ErrAlreadyScheduled):
		c.AbortWithStatusJSON(http.StatusConflict, PostConflict{
			Error: "This post already has a scheduled revision. Change that revision instead.",
			Code:  PublicationErrorCodeAlreadyScheduled,
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
	At                 time.Time    `json:"at"`
	IntegrationIds     *[]uuid.UUID `json:"integrationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RevisionId         uuid.UUID    `json:"revisionId"`
	RoleIntegrationIds *[]uuid.UUID `json:"roleIntegrationIds,omitempty"`
}

type SchedulePostRequest struct {
	At                 time.Time    `json:"at"`
	IntegrationIds     *[]uuid.UUID `json:"integrationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RoleIntegrationIds *[]uuid.UUID `json:"roleIntegrationIds,omitempty"`
	Version            int          `json:"version"`
}

func announcementOf(
	integrations *[]uuid.UUID,
	roles *[]uuid.UUID,
	note *string,
) Announcement {
	made := Announcement{Ping: readIDs(roles)}
	if integrations != nil {
		chosen := readIDs(integrations)
		made.Integrations = &chosen
	}
	if note != nil {
		made.Note = *note
	}
	return made
}

func readIDs(listed *[]uuid.UUID) []uuid.UUID {
	if listed == nil {
		return nil
	}
	held := make([]uuid.UUID, 0, len(*listed))
	for _, one := range *listed {
		held = append(held, one)
	}
	return held
}
