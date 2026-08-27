package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetProfileRestriction(c *gin.Context, handle string) {
	if _, ok := h.adminAccount(c, "read a restriction reason"); !ok {
		return
	}
	found, err := h.accounts.ProfileRestriction(c.Request.Context(), handle)
	if err != nil {
		h.restrictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestriction(found))
}

func (h *Handlers) RestrictProfile(c *gin.Context, handle string) {
	admin, ok := h.adminAccount(c, "restrict a profile")
	if !ok {
		return
	}
	var request RestrictProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the audit reason as JSON."})
		return
	}
	restriction, err := h.accounts.RestrictProfile(c.Request.Context(), admin, handle, request.Reason)
	if err != nil {
		h.restrictionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIRestriction(restriction))
}

func (h *Handlers) RestoreProfile(c *gin.Context, handle string) {
	admin, ok := h.adminAccount(c, "restore a profile")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": field.Message, "field": field.Field})
	case errors.Is(err, account.ErrProfileNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such profile."})
	case errors.Is(err, account.ErrNotRestricted):
		c.JSON(http.StatusNotFound, gin.H{"error": "That profile is not restricted."})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not change the restriction."})
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
