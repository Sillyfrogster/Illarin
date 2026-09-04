package http

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListPostRevisions(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "reading the editions of a post")
	if !ok {
		return
	}
	kept, err := h.publications.Revisions(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostRevisionList{Revisions: toAPIRevisions(kept)})
}

func (h *Handlers) CheckpointPost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "keeping an edition of a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	kept, err := h.publications.Checkpoint(c.Request.Context(), editor, uuid.UUID(id), version)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIRevision(kept))
}

func (h *Handlers) RestorePostRevision(c *gin.Context, id types.UUID, revisionID types.UUID) {
	editor, ok := h.postEditor(c, "restoring an edition of a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	restored, err := h.publications.RestoreRevision(
		c.Request.Context(), editor, uuid.UUID(id), uuid.UUID(revisionID), version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(restored))
}

func (h *Handlers) ReadPostHistory(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "reading the history of a post")
	if !ok {
		return
	}
	done, err := h.publications.PostHistory(c.Request.Context(), editor, uuid.UUID(id))
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostActionList{Actions: toAPIActions(done)})
}

// workingVersion reads the version of the working copy an action names.
func (h *Handlers) workingVersion(c *gin.Context) (int, bool) {
	var request PostVersionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name the version of the working copy you mean.",
		})
		return 0, false
	}
	return request.Version, true
}

func toAPIRevisions(kept []publication.Revision) []PostRevision {
	listed := make([]PostRevision, 0, len(kept))
	for _, one := range kept {
		listed = append(listed, toAPIRevision(one))
	}
	return listed
}

func toAPIRevision(one publication.Revision) PostRevision {
	return PostRevision{
		Id:          types.UUID(one.ID),
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

func toAPIActions(done []publication.Action) []PostAction {
	listed := make([]PostAction, 0, len(done))
	for _, one := range done {
		shown := PostAction{
			Id:         types.UUID(one.ID),
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
