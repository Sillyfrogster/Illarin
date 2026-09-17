package notify

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

type Handlers struct {
	notifications *Service
}

func NewHandlers(notifications *Service) *Handlers {
	return &Handlers{notifications: notifications}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPut, "/v1/assets/:id/watch", d.JSON, h.WatchAsset)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/watch", d.JSON, h.StopWatchingAsset)
}
