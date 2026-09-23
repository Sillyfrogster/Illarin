package connect

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Files hands a connected app the main file it was sent
type Files interface {
	ServeSendFile(c *gin.Context, workID uuid.UUID, format string)
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
	routes.Handle(http.MethodPost, "/v1/connect/requests", d.JSON, h.StartConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/poll", d.JSON, h.PollConnectionRequest)
	routes.Handle(http.MethodGet, "/v1/connect/requests/:userCode", d.JSON, h.GetConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/requests/:userCode/approve", d.JSON, h.ApproveConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/requests/:userCode/deny", d.JSON, h.DenyConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/authorizations", d.JSON, h.StartConnectionAuthorization)
	routes.Handle(http.MethodGet, "/v1/connect/authorizations/:requestCode", d.JSON, h.GetConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/authorizations/:requestCode/approve", d.JSON, h.ApproveConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/authorizations/:requestCode/deny", d.JSON, h.DenyConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/token", d.JSON, h.ExchangeConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/refresh", d.JSON, h.RefreshAppCredentials)
	routes.Handle(http.MethodGet, "/v1/connected-apps", d.JSON, h.ListConnectedApps)
	routes.Handle(http.MethodGet, "/v1/connected-apps/me", d.JSON, h.GetConnectedApp)
	routes.Handle(http.MethodPut, "/v1/connected-apps/me", d.JSON, h.UpdateCapabilities)
	routes.Handle(http.MethodDelete, "/v1/connected-apps/:id", d.JSON, h.RevokeConnectedApp)
	routes.Handle(http.MethodPut, "/v1/connected-apps/:id/permissions", d.JSON, h.SetConnectedAppPermissions)
	routes.Handle(http.MethodPost, "/v1/sends/collect", d.Collect, h.CollectSends)
	routes.Handle(http.MethodDelete, "/v1/sends/:id", d.JSON, h.DiscardSend)
	routes.Handle(http.MethodPost, "/v1/library/sync", d.JSON, h.SyncLibrary)
	routes.Handle(http.MethodGet, "/v1/works/:id/connected-apps", d.JSON, h.GetWorkConnectedApps)
	routes.Handle(http.MethodPost, "/v1/works/:id/sends", d.JSON, h.SendWork)
	routes.Handle(http.MethodGet, sendPathStart+":id/export", d.Download, h.DownloadSendFile)
	registerAliases(routes, h)
}
