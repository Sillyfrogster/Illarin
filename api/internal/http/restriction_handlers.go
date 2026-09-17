package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetProfileRestriction(c *gin.Context) {
	handle := c.Param("handle")
	if _, ok := api.Admin(c, "read a restriction reason"); !ok {
		return
	}
	found, err := h.accounts.ProfileRestriction(c.Request.Context(), handle)
	if err != nil {
		h.restrictionError(c, err)
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
	restriction, err := h.accounts.RestrictProfile(c.Request.Context(), admin, handle, request.Reason)
	if err != nil {
		h.restrictionError(c, err)
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
	if err := h.accounts.RestoreProfile(c.Request.Context(), admin, handle); err != nil {
		h.restrictionError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) restrictionError(c *gin.Context, err error) {
	var field account.FieldError
	switch {
	case errors.As(err, &field):
		api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
	case errors.Is(err, account.ErrProfileNotFound):
		api.Refuse(c, http.StatusNotFound, "No such profile.")
	case errors.Is(err, account.ErrNotRestricted):
		api.Refuse(c, http.StatusNotFound, "That profile is not restricted.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not change the restriction.")
	}
}

func toAPIRestriction(found account.Restriction) ProfileRestriction {
	restriction := ProfileRestriction{Reason: found.Reason, RestrictedAt: found.RestrictedAt}
	if found.RestrictedBy != "" {
		actor := found.RestrictedBy
		restriction.RestrictedBy = &actor
	}
	return restriction
}
