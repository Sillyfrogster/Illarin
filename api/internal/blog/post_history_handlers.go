package blog

import (
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListPostRevisions(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "reading the editions of a post")
	if !ok {
		return
	}
	kept, err := h.publications.Revisions(c.Request.Context(), editor, id)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostRevisionList{Revisions: toAPIRevisions(kept)})
}

func (h *Handlers) CheckpointPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "keeping an edition of a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	kept, err := h.publications.Checkpoint(c.Request.Context(), editor, id, version)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIRevision(kept))
}

func (h *Handlers) RestorePostRevision(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	revisionID, ok := api.PathID(c, "revisionId")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "restoring an edition of a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	restored, err := h.publications.RestoreRevision(
		c.Request.Context(), editor, id, revisionID, version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(restored))
}

func (h *Handlers) ReadPostHistory(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "reading the history of a post")
	if !ok {
		return
	}
	done, err := h.publications.PostHistory(c.Request.Context(), editor, id)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostActionList{Actions: toAPIActions(done)})
}

func (h *Handlers) workingVersion(c *gin.Context) (int, bool) {
	var request PostVersionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Include the current drafted changes version.", "version")
		return 0, false
	}
	return request.Version, true
}

func toAPIRevisions(kept []Revision) []PostRevision {
	listed := make([]PostRevision, 0, len(kept))
	for _, one := range kept {
		listed = append(listed, toAPIRevision(one))
	}
	return listed
}

func toAPIRevision(one Revision) PostRevision {
	return PostRevision{
		Id:          one.ID,
		Number:      one.Number,
		Title:       one.Title,
		Summary:     one.Summary,
		Slug:        one.Slug,
		Category:    toAPICategory(one.Category),
		CapturedFor: PostRevisionReason(one.CapturedFor),
		CapturedBy:  one.CapturedBy,
		CapturedAt:  one.CapturedAt,
		Public:      one.Public,
	}
}

func toAPIActions(done []Action) []PostAction {
	listed := make([]PostAction, 0, len(done))
	for _, one := range done {
		shown := PostAction{
			Id:         one.ID,
			Actor:      one.Actor,
			Credential: one.Credential,
			Action:     one.Action,
			Revision:   one.Revision,
			At:         one.At,
		}
		if one.Before != "" {
			shown.Before = pointer(one.Before)
		}
		if one.After != "" {
			shown.After = pointer(one.After)
		}
		listed = append(listed, shown)
	}
	return listed
}

type PostAction struct {
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	After      *string   `json:"after,omitempty"`
	At         time.Time `json:"at"`
	Before     *string   `json:"before,omitempty"`
	Credential string    `json:"credential"`
	Id         uuid.UUID `json:"id"`
	Revision   *int      `json:"revision,omitempty"`
}

type PostActionList struct {
	Actions []PostAction `json:"actions"`
}

type PostRevision struct {
	CapturedAt  time.Time           `json:"capturedAt"`
	CapturedBy  string              `json:"capturedBy"`
	CapturedFor PostRevisionReason  `json:"capturedFor"`
	Category    PublicationCategory `json:"category"`
	Id          uuid.UUID           `json:"id"`
	Number      int                 `json:"number"`
	Public      bool                `json:"public"`
	Slug        string              `json:"slug"`
	Summary     string              `json:"summary"`
	Title       string              `json:"title"`
}

type PostRevisionList struct {
	Revisions []PostRevision `json:"revisions"`
}

type PostRevisionReason string

const (
	PostRevisionReasonCheckpoint  PostRevisionReason = "checkpoint"
	PostRevisionReasonPublication PostRevisionReason = "publication"
	PostRevisionReasonSchedule    PostRevisionReason = "schedule"
)

type PostVersionRequest struct {
	Version int `json:"version"`
}
