package http

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) DeletePost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "deleting a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	deleted, err := h.publications.DeletePost(
		c.Request.Context(), editor, uuid.UUID(id), version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(deleted))
}

func (h *Handlers) RecoverPost(c *gin.Context, id types.UUID) {
	editor, ok := h.postEditor(c, "recovering a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	recovered, err := h.publications.RecoverPost(
		c.Request.Context(), editor, uuid.UUID(id), version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(recovered))
}

func toAPIDeletion(found *publication.Deletion) *PostDeletion {
	if found == nil {
		return nil
	}
	return &PostDeletion{At: found.At, Until: found.Until, By: found.By}
}
