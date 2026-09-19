package notify

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
)

type Handlers struct {
	notifications *Service
	sends         *connect.Sends
}

func NewHandlers(notifications *Service, sends *connect.Sends) *Handlers {
	return &Handlers{notifications: notifications, sends: sends}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPut, "/v1/works/:id/follow", d.JSON, h.FollowWork)
	routes.Handle(http.MethodDelete, "/v1/works/:id/follow", d.JSON, h.StopFollowingWork)
	routes.Handle(http.MethodDelete, "/v1/notifications", d.JSON, h.ClearNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications", d.JSON, h.ListNotifications)
	routes.Handle(http.MethodGet, "/v1/notifications/unread", d.JSON, h.CountUnreadNotifications)
	routes.Handle(http.MethodPost, "/v1/notifications/read", d.JSON, h.MarkAllNotificationsRead)
	routes.Handle(http.MethodDelete, "/v1/notifications/:id", d.JSON, h.RemoveNotification)
	routes.Handle(http.MethodPost, "/v1/notifications/:id/read", d.JSON, h.MarkNotificationRead)
	registerAliases(routes, h)
}
