package page

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

type workDetailsInput struct {
	Name   string  `json:"name"`
	Blurb  *string `json:"blurb"`
	IsNsfw *bool   `json:"isNsfw"`
}

func (h *Handlers) SetWorkDetails(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "saving a work")
	if !ok {
		return
	}
	var request workDetailsInput
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a name, a blurb, and an adult content answer of true, false or null.")
		return
	}
	if request.Blurb == nil {
		api.RefuseField(c, http.StatusBadRequest, "blurb", "Send a blurb. Use an empty string to clear it.")
		return
	}
	candidate := &work.Candidate{Version: version}
	err := h.works.SetDetails(c.Request.Context(), Details{
		OwnerID: owner.ID, WorkID: id,
		Name: request.Name, Blurb: *request.Blurb, IsNSFW: request.IsNsfw,
	}, candidate)
	if CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such work.")
	case errors.Is(err, ErrNameTooLong):
		api.Refuse(c, http.StatusBadRequest, "The name is too long.")
	case errors.Is(err, ErrBlurbTooLong):
		api.RefuseField(c, http.StatusBadRequest, "blurb", "The blurb must be 400 characters or fewer.")
	case errors.Is(err, ErrRatingUnanswerable):
		api.Refuse(c, http.StatusBadRequest, "A published work needs an adult content answer.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not save the details.")
	default:
		c.Status(http.StatusNoContent)
	}
}
