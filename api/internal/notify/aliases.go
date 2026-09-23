package notify

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPut, "/v1/assets/:id/watch", d.JSON, h.FollowWork)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/watch", d.JSON, h.StopFollowingWork)
}

// The field names a notification answered to before the renames, kept for sixty days
var (
	notificationAliases = map[string]string{"work": "asset", "sendTo": "sendTargets"}
	sendAppAliases      = map[string]string{
		"connectedAppId": "instanceId", "name": "instanceName", "appName": "applicationName",
	}
)

func (n Notification) MarshalJSON() ([]byte, error) {
	type plain Notification
	return api.MarshalAliased(plain(n), notificationAliases)
}

func (a NotificationSendApp) MarshalJSON() ([]byte, error) {
	type plain NotificationSendApp
	return api.MarshalAliased(plain(a), sendAppAliases)
}
