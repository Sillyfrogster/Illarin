package connect

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Files hands a connected app the file it was sent
type Files interface {
	LinkedInstanceFile(c *gin.Context, assetID uuid.UUID, target string)
}

type Handlers struct {
	apps  *Apps
	sends *Sends
	files Files
}

func NewHandlers(apps *Apps, sends *Sends, files Files) *Handlers {
	return &Handlers{apps: apps, sends: sends, files: files}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/link/requests", d.JSON, h.StartLinkRequest)
	routes.Handle(http.MethodPost, "/v1/link/poll", d.JSON, h.PollLinkRequest)
	routes.Handle(http.MethodGet, "/v1/link/requests/:userCode", d.JSON, h.GetLinkRequest)
	routes.Handle(http.MethodPost, "/v1/link/requests/:userCode/approve", d.JSON, h.ApproveLinkRequest)
	routes.Handle(http.MethodPost, "/v1/link/requests/:userCode/deny", d.JSON, h.DenyLinkRequest)
	routes.Handle(http.MethodPost, "/v1/link/authorizations", d.JSON, h.StartLinkAuthorization)
	routes.Handle(http.MethodGet, "/v1/link/authorizations/:requestCode", d.JSON, h.GetLinkAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/authorizations/:requestCode/approve", d.JSON, h.ApproveLinkAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/authorizations/:requestCode/deny", d.JSON, h.DenyLinkAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/token", d.JSON, h.ExchangeLinkAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/refresh", d.JSON, h.RefreshInstanceToken)
	routes.Handle(http.MethodGet, "/v1/instances", d.JSON, h.ListInstances)
	routes.Handle(http.MethodGet, "/v1/instances/me", d.JSON, h.GetInstance)
	routes.Handle(http.MethodPut, "/v1/instances/me", d.JSON, h.UpdateInstance)
	routes.Handle(http.MethodDelete, "/v1/instances/:id", d.JSON, h.RevokeInstance)
	routes.Handle(http.MethodPost, "/v1/deliveries/collect", d.Deliver, h.CollectDeliveries)
	routes.Handle(http.MethodDelete, "/v1/deliveries/:id", d.JSON, h.DiscardDelivery)
	routes.Handle(http.MethodPost, "/v1/library/sync", d.JSON, h.SyncLibrary)
	routes.Handle(http.MethodGet, "/v1/assets/:id/instances", d.JSON, h.GetAssetInstances)
	routes.Handle(http.MethodPost, "/v1/assets/:id/deliveries", d.JSON, h.SendAssetToInstance)
	routes.Handle(http.MethodGet, "/delivery/:id/export", d.Download, h.DownloadDeliveryExport)
}
