package edit

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/assets/:id/blocks", d.JSON, h.AddWorkBlock)
	routes.Handle(http.MethodPut, "/v1/assets/:id/blocks", d.JSON, h.ArrangeWorkBlocks)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/blocks/:blockId", d.JSON, h.RemoveWorkBlock)
	routes.Handle(http.MethodPut, "/v1/assets/:id/blocks/:blockId", d.JSON, h.SaveWorkBlock)
	routes.Handle(http.MethodPost, "/v1/assets/:id/blocks/:blockId/move-and-remove", d.JSON, h.MoveWorkBlockContent)
}

// The field name a block save took before the rename, kept for sixty days
var saveWorkBlockAliases = map[string]string{"makePromptsPublic": "exposeProtected"}

func (r *SaveWorkBlockRequest) UnmarshalJSON(data []byte) error {
	type plain SaveWorkBlockRequest
	return api.UnmarshalAliased(data, (*plain)(r), saveWorkBlockAliases)
}
