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
	registerAppRoute(routes, "/v1/connect/requests", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.StartConnectionRequest})
	registerAppRoute(routes, "/v1/connect/poll", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.PollConnectionRequest})
	routes.Handle(http.MethodGet, "/v1/connect/requests/:userCode", d.JSON, h.GetConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/requests/:userCode/approve", d.JSON, h.ApproveConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/connect/requests/:userCode/deny", d.JSON, h.DenyConnectionRequest)
	registerAppRoute(routes, "/v1/connect/authorizations", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.StartConnectionAuthorization})
	routes.Handle(http.MethodGet, "/v1/connect/authorizations/:requestCode", d.JSON, h.GetConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/authorizations/:requestCode/approve", d.JSON, h.ApproveConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/connect/authorizations/:requestCode/deny", d.JSON, h.DenyConnectionAuthorization)
	registerAppRoute(routes, "/v1/connect/token", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.ExchangeConnectionAuthorization})
	registerAppRoute(routes, "/v1/connect/refresh", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.RefreshAppCredentials})
	routes.Handle(http.MethodGet, "/v1/connected-apps", d.JSON, h.ListConnectedApps)
	registerAppRoute(routes, "/v1/connected-apps/me", d.JSON, map[string]gin.HandlerFunc{
		http.MethodGet: h.GetConnectedApp,
		http.MethodPut: h.UpdateCapabilities,
	})
	routes.Handle(http.MethodDelete, "/v1/connected-apps/:id", d.JSON, h.RevokeConnectedApp)
	routes.Handle(http.MethodPut, "/v1/connected-apps/:id/permissions", d.JSON, h.SetConnectedAppPermissions)
	registerAppRoute(routes, "/v1/sends/collect", d.Collect, map[string]gin.HandlerFunc{http.MethodPost: h.CollectSends})
	routes.Handle(http.MethodDelete, "/v1/sends/:id", d.JSON, h.DiscardSend)
	registerAppRoute(routes, "/v1/library/sync", d.JSON, map[string]gin.HandlerFunc{http.MethodPost: h.SyncLibrary})
	routes.Handle(http.MethodGet, "/v1/works/:id/connected-apps", d.JSON, h.GetWorkConnectedApps)
	routes.Handle(http.MethodPost, "/v1/works/:id/sends", d.JSON, h.SendWork)
	registerAppRoute(routes, sendPathStart+":id/export", d.Download, map[string]gin.HandlerFunc{http.MethodGet: h.DownloadSendFile})
	registerAliases(routes, h)
}
