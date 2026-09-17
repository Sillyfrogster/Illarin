package private

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	sealed *Service
}

func NewHandlers(sealed *Service) *Handlers {
	return &Handlers{sealed: sealed}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/sealed", d.JSON, h.ExportSealedContent)
}

func (h *Handlers) ExportSealedContent(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "reading sealed content")
	if !ok {
		return
	}
	sealed, err := h.sealed.OpenSealedContent(c.Request.Context(), owner.ID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "This asset holds no sealed content.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read the sealed content.")
	default:
		c.Header("Content-Disposition", `attachment; filename="`+sealed.Filename+`"`)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "private, no-store")
		c.Data(http.StatusOK, sealed.MediaType, sealed.Body)
	}
}
