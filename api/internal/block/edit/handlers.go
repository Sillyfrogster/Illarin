package edit

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

type Handlers struct {
	blocks *Service
}

func NewHandlers(blocks *Service) *Handlers {
	return &Handlers{blocks: blocks}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/assets/:id/blocks", d.JSON, h.AddAssetBlock)
	routes.Handle(http.MethodPut, "/v1/assets/:id/blocks", d.JSON, h.ArrangeAssetBlocks)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/blocks/:blockId", d.JSON, h.RemoveAssetBlock)
	routes.Handle(http.MethodPut, "/v1/assets/:id/blocks/:blockId", d.JSON, h.SaveAssetBlock)
	routes.Handle(http.MethodPost, "/v1/assets/:id/blocks/:blockId/move-and-remove", d.JSON, h.MoveAssetBlockContent)
}
