package notify

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
)

type Handlers struct {
	notifications *Service
	deliveries    *connect.Sends
}

func NewHandlers(notifications *Service, deliveries *connect.Sends) *Handlers {
	return &Handlers{notifications: notifications, deliveries: deliveries}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPut, "/v1/assets/:id/watch", d.JSON, h.WatchAsset)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/watch", d.JSON, h.StopWatchingAsset)
	routes.Handle(http.MethodDelete, "/v1/notifications", d.JSON, h.ClearNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications", d.JSON, h.ListNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications/unread", d.JSON, h.CountUnreadNotifications)
	routes.Handle(http.MethodPost, "/v1/notifications/read", d.JSON, h.MarkAllNotificationsRead)
	routes.Handle(http.MethodDelete, "/v1/notifications/:id", d.JSON, h.RemoveNotification)
	routes.Handle(http.MethodPost, "/v1/notifications/:id/read", d.JSON, h.MarkNotificationRead)
}
