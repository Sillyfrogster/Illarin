package staff

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	staff *Service
}

func NewHandlers(staff *Service) *Handlers {
	return &Handlers{staff: staff}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodDelete, "/v1/profiles/:handle/restriction", d.JSON, h.RestoreProfile)
	routes.Handle(http.MethodGet, "/v1/profiles/:handle/restriction", d.JSON, h.GetProfileRestriction)
	routes.Handle(http.MethodPut, "/v1/profiles/:handle/restriction", d.JSON, h.RestrictProfile)
	routes.Handle(http.MethodDelete, "/v1/works/:id/withhold", d.JSON, h.ClearWorkWithhold)
	routes.Handle(http.MethodPut, "/v1/works/:id/withhold", d.JSON, h.WithholdWork)
	registerAliases(routes, h)
}

func (h *Handlers) GetProfileRestriction(c *gin.Context) {
	handle := c.Param("handle")
	if _, ok := api.Admin(c, "read a restriction reason"); !ok {
		return
	}
	found, err := h.staff.ProfileRestriction(c.Request.Context(), handle)
	if err != nil {
		restrictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestriction(found))
}

func (h *Handlers) RestrictProfile(c *gin.Context) {
	handle := c.Param("handle")
	admin, ok := api.Admin(c, "restrict a profile")
	if !ok {
		return
	}
	var request RestrictProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the audit reason as JSON.")
		return
	}
	restriction, err := h.staff.RestrictProfile(c.Request.Context(), admin, handle, request.Reason)
	if err != nil {
		restrictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestriction(restriction))
}

func (h *Handlers) RestoreProfile(c *gin.Context) {
	handle := c.Param("handle")
	admin, ok := api.Admin(c, "restore a profile")
	if !ok {
		return
	}
	if err := h.staff.RestoreProfile(c.Request.Context(), admin, handle); err != nil {
		restrictionError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) WithholdWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	admin, ok := api.Admin(c, "manage withholds")
	if !ok {
		return
	}
	var request WithholdWorkRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Give a reason for withholding the work.")
		return
	}
	err := h.staff.Withhold(c.Request.Context(), id, admin.ID, request.Reason)
	switch {
	case errors.Is(err, ErrInvalidWithholdReason):
		api.Refuse(c, http.StatusBadRequest, "Give a reason for withholding the work.")
	case errors.Is(err, ErrWorkNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not withhold the work.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ClearWorkWithhold(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if _, ok := api.Admin(c, "manage withholds"); !ok {
		return
	}
	err := h.staff.ClearWithhold(c.Request.Context(), id)
	switch {
	case errors.Is(err, ErrWorkNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not clear the withhold.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func restrictionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidReason):
		api.RefuseField(c, http.StatusBadRequest, "reason",
			fmt.Sprintf("Give a reason of up to %d characters.", restrictionReasonLimit))
	case errors.Is(err, ErrProfileNotFound):
		api.Refuse(c, http.StatusNotFound, "No such profile.")
	case errors.Is(err, ErrNotRestricted):
		api.Refuse(c, http.StatusNotFound, "That profile is not restricted.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not change the restriction.")
	}
}

func toAPIRestriction(found Restriction) ProfileRestriction {
	restriction := ProfileRestriction{Reason: found.Reason, RestrictedAt: found.RestrictedAt}
	if found.RestrictedBy != "" {
		actor := found.RestrictedBy
		restriction.RestrictedBy = &actor
	}
	return restriction
}
