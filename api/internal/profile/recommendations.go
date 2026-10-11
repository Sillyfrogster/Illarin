package profile

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) AddRecommendation(c *gin.Context) {
	owner, ok := api.Verified(c, "recommending work on your profile")
	if !ok {
		return
	}
	var request AddRecommendationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the work id as JSON.")
		return
	}
	saved, err := h.accounts.AddRecommendation(c.Request.Context(), owner, request.WorkId)
	if err != nil {
		refuseProfile(c, err)
		return
	}
	h.showProfile(c, saved)
}

func (h *Handlers) RemoveRecommendation(c *gin.Context) {
	owner, ok := api.Verified(c, "changing your recommendations")
	if !ok {
		return
	}
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	saved, err := h.accounts.RemoveRecommendation(c.Request.Context(), owner, id)
	if err != nil {
		refuseProfile(c, err)
		return
	}
	h.showProfile(c, saved)
}
