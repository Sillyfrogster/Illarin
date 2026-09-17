package http

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

func registerRoutes(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodDelete, "/v1/notifications", d.JSON, h.ClearNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications", d.JSON, h.ListNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications/unread", d.JSON, h.CountUnreadNotifications)
	routes.Handle(http.MethodPost, "/v1/notifications/read", d.JSON, h.MarkAllNotificationsRead)
	routes.Handle(http.MethodDelete, "/v1/notifications/:id", d.JSON, h.RemoveNotification)
	routes.Handle(http.MethodPost, "/v1/notifications/:id/read", d.JSON, h.MarkNotificationRead)
	routes.Handle(http.MethodDelete, "/v1/profiles/:handle/restriction", d.JSON, h.RestoreProfile)
	routes.Handle(http.MethodGet, "/v1/profiles/:handle/restriction", d.JSON, h.GetProfileRestriction)
	routes.Handle(http.MethodPut, "/v1/profiles/:handle/restriction", d.JSON, h.RestrictProfile)
	routes.Handle(http.MethodGet, "/v1/assets/:id/preserved", d.JSON, h.ListPreservedNamespaces)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/preserved/:namespace", d.JSON, h.DeletePreservedNamespace)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/withhold", d.JSON, h.ClearAssetWithhold)
	routes.Handle(http.MethodPut, "/v1/assets/:id/withhold", d.JSON, h.WithholdAsset)
	routes.Handle(http.MethodGet, "/v1/assets/:id/media", d.JSON, h.ListMedia)
	routes.Handle(http.MethodPost, "/v1/assets/:id/media", d.Upload, h.AddMedia)
	routes.Handle(http.MethodGet, "/media/:media_id/:variant/:derivative_version", d.Download, h.GetMediaVariant)
}
