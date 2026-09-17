package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) WithdrawPost(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "withdrawing a post")
	if !ok {
		return
	}
	var request WithdrawPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send the withdrawal as JSON.")
		return
	}
	explanation := ""
	if request.Explanation != nil {
		explanation = *request.Explanation
	}
	withdrawn, err := h.publications.WithdrawPost(
		c.Request.Context(), editor, id, request.Version, request.Reason, explanation,
		announcementOf(request.DestinationIds, nil, request.Note),
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(withdrawn))
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
		refuseField(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Choose a revision to republish.", "revisionId")
		return
	}
	back, err := h.publications.RepublishPost(
		c.Request.Context(), editor, id, request.RevisionId, request.Version,
		announcementOf(request.DestinationIds, nil, request.Note),
	)
	if err != nil {
		h.postError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toAPIPost(back))
}

func (h *Handlers) withdrawnPost(c *gin.Context, slug string) bool {
	found, err := h.publications.WithdrawnPost(c.Request.Context(), slug)
	if errors.Is(err, publication.ErrPostNotFound) {
		return false
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the address.")
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
