package private

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	preserved *Service
}

func NewHandlers(preserved *Service) *Handlers {
	return &Handlers{preserved: preserved}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/preserved-prompts", d.JSON, h.ExportPreservedPrompts)
	registerAliases(routes, h)
}

func (h *Handlers) ExportPreservedPrompts(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "reading preserved prompts")
	if !ok {
		return
	}
	preserved, err := h.preserved.OpenPreservedPrompts(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "This work holds no preserved prompts.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read the preserved prompts.")
	default:
		c.Header("Content-Disposition", `attachment; filename="`+preserved.Filename+`"`)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "private, no-store")
		c.Data(http.StatusOK, preserved.MediaType, preserved.Body)
	}
}
