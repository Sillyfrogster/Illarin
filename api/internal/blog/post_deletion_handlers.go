package blog

import (
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) DeletePost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "deleting a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	deleted, err := h.publications.DeletePost(
		c.Request.Context(), editor, id, version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(deleted))
}

func (h *Handlers) RecoverPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "recovering a post")
	if !ok {
		return
	}
	version, ok := h.workingVersion(c)
	if !ok {
		return
	}
	recovered, err := h.publications.RecoverPost(
		c.Request.Context(), editor, id, version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(recovered))
}

func toAPIDeletion(found *Deletion) *PostDeletion {
	if found == nil {
		return nil
	}
	return &PostDeletion{At: found.At, Until: found.Until, By: found.By}
}

type PostDeletion struct {
	At    time.Time `json:"at"`
	By    string    `json:"by"`
	Until time.Time `json:"until"`
}
