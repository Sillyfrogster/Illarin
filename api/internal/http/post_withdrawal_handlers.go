package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) WithdrawPost(c *gin.Context, id types.UUID, _ WithdrawPostParams) {
	editor, ok := h.postEditor(c, "withdrawing a post")
	if !ok {
		return
	}
	var request WithdrawPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the withdrawal as JSON."})
		return
	}
	explanation := ""
	if request.Explanation != nil {
		explanation = *request.Explanation
	}
	withdrawn, err := h.publications.WithdrawPost(
		c.Request.Context(), editor, uuid.UUID(id), request.Version, request.Reason, explanation,
		announcementOf(request.DestinationIds, request.Note),
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(withdrawn))
}

func (h *Handlers) RepublishPost(c *gin.Context, id types.UUID, _ RepublishPostParams) {
	editor, ok := h.postEditor(c, "republishing a post")
	if !ok {
		return
	}
	var request RepublishPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name the edition readers get back."})
		return
	}
	back, err := h.publications.RepublishPost(
		c.Request.Context(), editor, uuid.UUID(id), uuid.UUID(request.RevisionId), request.Version,
		announcementOf(request.DestinationIds, request.Note),
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(back))
}

// withdrawnPost answers a withdrawn address with the tombstone behind it, and
// says whether the address had one at all.
func (h *Handlers) withdrawnPost(c *gin.Context, slug string) bool {
	found, err := h.publications.WithdrawnPost(c.Request.Context(), slug)
	if errors.Is(err, publication.ErrPostNotFound) {
		return false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the address."})
		return true
	}
	c.JSON(http.StatusGone, WithdrawnPost{Slug: found.Slug, Explanation: found.Explanation})
	return true
}

func toAPIWithdrawal(found *publication.Withdrawal) *PostWithdrawal {
	if found == nil {
		return nil
	}
	return &PostWithdrawal{
		Reason:      found.Reason,
		Explanation: found.Explanation,
		By:          found.By,
		At:          found.At,
	}
}
