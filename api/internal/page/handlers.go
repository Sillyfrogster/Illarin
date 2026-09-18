package page

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
)

type Handlers struct {
	works         *Service
	accounts      *account.Service
	deliveries    *connect.Sends
	notifications *notify.Service
}

func NewHandlers(
	works *Service,
	accounts *account.Service,
	deliveries *connect.Sends,
	notifications *notify.Service,
) *Handlers {
	return &Handlers{works: works, accounts: accounts, deliveries: deliveries, notifications: notifications}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets", d.JSON, h.ListAssets)
	routes.Handle(http.MethodDelete, "/v1/assets/:id", d.JSON, h.DeleteAsset)
	routes.Handle(http.MethodGet, "/v1/assets/:id", d.JSON, h.GetAsset)
	routes.Handle(http.MethodPost, "/v1/assets/:id/restore", d.JSON, h.RestoreAsset)
	routes.Handle(http.MethodPut, "/v1/assets/:id/identity", d.JSON, h.SetAssetIdentity)
	routes.Handle(http.MethodPost, "/v1/assets/:id/publish", d.JSON, h.PublishAsset)
	routes.Handle(http.MethodPut, "/v1/assets/:id/discovery", d.JSON, h.SetAssetDiscovery)
	routes.Handle(http.MethodGet, "/v1/profiles/:handle/deleted", d.JSON, h.ListDeletedAssets)
	routes.Handle(http.MethodGet, "/v1/assets/:id/preserved", d.JSON, h.ListPreservedNamespaces)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/preserved/:namespace", d.JSON, h.DeletePreservedNamespace)
}
