package blog

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) UnpublishPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "unpublishing a post")
	if !ok {
		return
	}
	var request UnpublishPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseBlog(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Send the unpublishing as JSON.")
		return
	}
	explanation := ""
	if request.Explanation != nil {
		explanation = *request.Explanation
	}
	unpublished, err := h.blog.UnpublishPost(
		c.Request.Context(), editor, id, request.Version, request.Reason, explanation,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(unpublished))
}

func (h *Handlers) RepublishPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "republishing a post")
	if !ok {
		return
	}
	var request RepublishPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refuseField(c, http.StatusBadRequest, BlogErrorCodeInvalid, "Choose a revision to republish.", "revisionId")
		return
	}
	back, err := h.blog.RepublishPost(
		c.Request.Context(), editor, id, request.RevisionId, request.Version,
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(back))
}

func (h *Handlers) unpublishedPost(c *gin.Context, slug string) bool {
	found, err := h.blog.UnpublishedPost(c.Request.Context(), slug)
	if errors.Is(err, ErrPostNotFound) {
		return false
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the address.")
		return true
	}
	c.JSON(http.StatusGone, UnpublishedPost{Slug: found.Slug, Explanation: found.Explanation})
	return true
}

func toAPIUnpublishing(found *Unpublishing) *PostUnpublishing {
	if found == nil {
		return nil
	}
	return &PostUnpublishing{
		Reason:      found.Reason,
		Explanation: found.Explanation,
		By:          found.By,
		At:          found.At,
	}
}

type PostUnpublishing struct {
	At          time.Time `json:"at"`
	By          string    `json:"by"`
	Explanation string    `json:"explanation"`
	Reason      string    `json:"reason"`
}

type RepublishPostRequest struct {
	RevisionId uuid.UUID `json:"revisionId"`
	Version    int       `json:"version"`
}

type UnpublishPostRequest struct {
	Explanation *string `json:"explanation,omitempty"`
	Reason      string  `json:"reason"`
	Version     int     `json:"version"`
}

type UnpublishedPost struct {
	Explanation string `json:"explanation"`
	Slug        string `json:"slug"`
}
