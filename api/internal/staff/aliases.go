package staff

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodDelete, "/v1/assets/:id/withhold", d.JSON, h.ClearWorkWithhold)
	routes.Handle(http.MethodPut, "/v1/assets/:id/withhold", d.JSON, h.WithholdWork)
}
