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

// The field name a notification answered to before the rename, kept for sixty days
var notificationAliases = map[string]string{"work": "asset"}

func (n Notification) MarshalJSON() ([]byte, error) {
	type plain Notification
	return api.MarshalAliased(plain(n), notificationAliases)
}
