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
	routes.Handle(http.MethodDelete, "/v1/profiles/:handle/restricted", d.JSON, h.RestoreProfile)
	routes.Handle(http.MethodGet, "/v1/profiles/:handle/restricted", d.JSON, h.GetRestrictedProfile)
	routes.Handle(http.MethodPut, "/v1/profiles/:handle/restricted", d.JSON, h.RestrictProfile)
	routes.Handle(http.MethodDelete, "/v1/works/:id/takedown", d.JSON, h.LiftTakedown)
	routes.Handle(http.MethodPut, "/v1/works/:id/takedown", d.JSON, h.TakeDownWork)
	registerAliases(routes, h)
}

func (h *Handlers) GetRestrictedProfile(c *gin.Context) {
	handle := c.Param("handle")
	if _, ok := api.Admin(c, "read why a profile is restricted"); !ok {
		return
	}
	found, err := h.staff.RestrictedProfile(c.Request.Context(), handle)
	if err != nil {
		restrictedProfileError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestrictedProfile(found))
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
	restricted, err := h.staff.RestrictProfile(c.Request.Context(), admin, handle, request.Reason)
	if err != nil {
		restrictedProfileError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestrictedProfile(restricted))
}

func (h *Handlers) RestoreProfile(c *gin.Context) {
	handle := c.Param("handle")
	admin, ok := api.Admin(c, "restore a profile")
	if !ok {
		return
	}
	if err := h.staff.RestoreProfile(c.Request.Context(), admin, handle); err != nil {
		restrictedProfileError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) TakeDownWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	admin, ok := api.Admin(c, "take down works")
	if !ok {
		return
	}
	var request TakeDownWorkRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Give a reason for taking down the work.")
		return
	}
	err := h.staff.TakeDown(c.Request.Context(), id, admin.ID, request.Reason)
	switch {
	case errors.Is(err, ErrInvalidTakedownReason):
		api.Refuse(c, http.StatusBadRequest, "Give a reason for taking down the work.")
	case errors.Is(err, ErrWorkNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not take down the work.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) LiftTakedown(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if _, ok := api.Admin(c, "take down works"); !ok {
		return
	}
	err := h.staff.LiftTakedown(c.Request.Context(), id)
	switch {
	case errors.Is(err, ErrWorkNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not lift the takedown.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func restrictedProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidReason):
		api.RefuseField(c, http.StatusBadRequest, "reason",
			fmt.Sprintf("Give a reason of up to %d characters.", restrictedReasonLimit))
	case errors.Is(err, ErrProfileNotFound):
		api.Refuse(c, http.StatusNotFound, "No such profile.")
	case errors.Is(err, ErrNotRestricted):
		api.Refuse(c, http.StatusNotFound, "That profile is not restricted.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not change whether the profile is restricted.")
	}
}

func toAPIRestrictedProfile(found Restricted) RestrictedProfile {
	restricted := RestrictedProfile{Reason: found.Reason, RestrictedAt: found.RestrictedAt}
	if found.RestrictedBy != "" {
		actor := found.RestrictedBy
		restricted.RestrictedBy = &actor
	}
	return restricted
}
