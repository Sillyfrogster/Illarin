package connect

import (
	"encoding/json"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

// registerAliases serves the paths this package had before the renames, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/instances", d.JSON, h.GetWorkConnectedApps)
	routes.Handle(http.MethodPost, "/v1/assets/:id/deliveries", d.JSON, h.SendWork)
	routes.Handle(http.MethodPost, "/v1/link/requests", d.JSON, h.StartConnectionRequest)
	routes.Handle(http.MethodPost, oldPollPath, d.JSON, h.PollConnectionRequest)
	routes.Handle(http.MethodGet, "/v1/link/requests/:userCode", d.JSON, h.GetConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/link/requests/:userCode/approve", d.JSON, h.ApproveConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/link/requests/:userCode/deny", d.JSON, h.DenyConnectionRequest)
	routes.Handle(http.MethodPost, "/v1/link/authorizations", d.JSON, h.StartConnectionAuthorization)
	routes.Handle(http.MethodGet, "/v1/link/authorizations/:requestCode", d.JSON, h.GetConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/authorizations/:requestCode/approve", d.JSON, h.ApproveConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/authorizations/:requestCode/deny", d.JSON, h.DenyConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/token", d.JSON, h.ExchangeConnectionAuthorization)
	routes.Handle(http.MethodPost, "/v1/link/refresh", d.JSON, h.RefreshAppCredentials)
	routes.Handle(http.MethodGet, "/v1/instances", d.JSON, h.ListConnectedApps)
	routes.Handle(http.MethodGet, "/v1/instances/me", d.JSON, h.GetConnectedApp)
	routes.Handle(http.MethodPut, "/v1/instances/me", d.JSON, h.UpdateCapabilities)
	routes.Handle(http.MethodDelete, "/v1/instances/:id", d.JSON, h.RevokeConnectedApp)
	routes.Handle(http.MethodPost, "/v1/deliveries/collect", d.Collect, h.CollectSends)
	routes.Handle(http.MethodDelete, "/v1/deliveries/:id", d.JSON, h.DiscardSend)
	routes.Handle(http.MethodGet, "/v1/works/:id/instances", d.JSON, h.GetWorkConnectedApps)
	routes.Handle(http.MethodPost, "/v1/works/:id/deliveries", d.JSON, h.SendWork)
	routes.Handle(http.MethodGet, "/delivery/:id/export", d.Download, h.DownloadSendFile)
}

const oldPollPath = "/v1/link/poll"

// connectedStatus answers the old poll path with the status it gave before the rename
func connectedStatus(c *gin.Context) string {
	if c.FullPath() == oldPollPath {
		return "linked"
	}
	return "connected"
}

// oldPermissions is each permission value an app could send before the rename
var oldPermissions = map[Permission]string{PermissionReceiveWorks: "asset:receive"}

// UnmarshalJSON takes a permission under the value it had before the rename as well as its own
func (p *Permission) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*p = Permission(value)
	for current, old := range oldPermissions {
		if value == old {
			*p = current
		}
	}
	return nil
}

func scopesOf(permissions []Permission) []string {
	scopes := make([]string, len(permissions))
	for index, permission := range permissions {
		scopes[index] = string(permission)
		if old, renamed := oldPermissions[permission]; renamed {
			scopes[index] = old
		}
	}
	return scopes
}

// The field names the connect protocol answered to before the renames, kept for sixty days
var (
	connectionAliases = map[string]string{
		"appName": "applicationName", "name": "instanceName", "appVersion": "applicationVersion",
		"acceptedFormats": "acceptedTargets", "permissions": "scopes",
	}
	connectedAppAliases = map[string]string{
		"appName": "applicationName", "name": "instanceName", "appVersion": "applicationVersion",
		"acceptedFormats": "acceptedTargets", "connectedAt": "linkedAt",
	}
	pendingAliases = map[string]string{
		"appName": "applicationName", "name": "instanceName", "appVersion": "applicationVersion",
		"acceptedFormats": "acceptedTargets",
	}
	capabilitiesAliases   = map[string]string{"appVersion": "applicationVersion", "acceptedFormats": "acceptedTargets"}
	credentialsAliases    = map[string]string{"connectedApp": "instance"}
	collectedSendsAliases = map[string]string{"sends": "deliveries"}
	sendFileAliases       = map[string]string{"type": "kind"}
	collectedSendAliases  = map[string]string{
		"workId": "assetId", "type": "kind", "versionNumber": "contentGeneration", "files": "artifacts",
	}
	libraryEntryAliases     = map[string]string{"workId": "assetId", "versionNumber": "contentGeneration"}
	libraryReportAliases    = map[string]string{"appVersion": "applicationVersion"}
	queuedSendAliases       = map[string]string{"workId": "assetId", "connectedAppId": "instanceId"}
	sendWorkAliases         = map[string]string{"connectedAppId": "instanceId"}
	withheldNoticeAliases   = map[string]string{"workId": "assetId"}
	workConnectedAppAliases = map[string]string{
		"connectedAppId": "instanceId", "appName": "applicationName", "name": "instanceName",
		"send": "delivery", "installedVersion": "installedGeneration",
	}
	workConnectedAppListAliases = map[string]string{"versionNumber": "contentGeneration"}
)

func (r *StartConnectionRequest) UnmarshalJSON(data []byte) error {
	type plain StartConnectionRequest
	return api.UnmarshalAliased(data, (*plain)(r), connectionAliases)
}

func (r *StartConnectionAuthorization) UnmarshalJSON(data []byte) error {
	type plain StartConnectionAuthorization
	return api.UnmarshalAliased(data, (*plain)(r), connectionAliases)
}

func (r *UpdateCapabilities) UnmarshalJSON(data []byte) error {
	type plain UpdateCapabilities
	return api.UnmarshalAliased(data, (*plain)(r), capabilitiesAliases)
}

func (a ConnectedAppDetail) MarshalJSON() ([]byte, error) {
	type plain ConnectedAppDetail
	return api.MarshalAliased(struct {
		plain
		Scopes []string `json:"scopes"`
	}{plain(a), scopesOf(a.Permissions)}, connectedAppAliases)
}

func (a ManagedConnectedApp) MarshalJSON() ([]byte, error) {
	type plain ManagedConnectedApp
	return api.MarshalAliased(struct {
		plain
		Scopes []string `json:"scopes"`
	}{plain(a), scopesOf(a.Permissions)}, connectedAppAliases)
}

func (p PendingConnection) MarshalJSON() ([]byte, error) {
	type plain PendingConnection
	return api.MarshalAliased(struct {
		plain
		Scopes []string `json:"scopes"`
	}{plain(p), scopesOf(p.Permissions)}, pendingAliases)
}

func (p PendingCodeConnection) MarshalJSON() ([]byte, error) {
	type plain PendingCodeConnection
	return api.MarshalAliased(struct {
		plain
		Scopes []string `json:"scopes"`
	}{plain(p), scopesOf(p.Permissions)}, pendingAliases)
}

func (g AppCredentials) MarshalJSON() ([]byte, error) {
	type plain AppCredentials
	return api.MarshalAliased(plain(g), credentialsAliases)
}

func (p ConnectedPoll) MarshalJSON() ([]byte, error) {
	type plain ConnectedPoll
	return api.MarshalAliased(plain(p), credentialsAliases)
}

func (l CollectedSends) MarshalJSON() ([]byte, error) {
	type plain CollectedSends
	return api.MarshalAliased(plain(l), collectedSendsAliases)
}

func (f SendFile) MarshalJSON() ([]byte, error) {
	type plain SendFile
	return api.MarshalAliased(plain(f), sendFileAliases)
}

func (s CollectedSend) MarshalJSON() ([]byte, error) {
	type plain CollectedSend
	return api.MarshalAliased(plain(s), collectedSendAliases)
}

func (e LibraryEntry) MarshalJSON() ([]byte, error) {
	type plain LibraryEntry
	return api.MarshalAliased(plain(e), libraryEntryAliases)
}

func (e *LibraryEntry) UnmarshalJSON(data []byte) error {
	type plain LibraryEntry
	return api.UnmarshalAliased(data, (*plain)(e), libraryEntryAliases)
}

func (r *LibraryReport) UnmarshalJSON(data []byte) error {
	type plain LibraryReport
	return api.UnmarshalAliased(data, (*plain)(r), libraryReportAliases)
}

func (s QueuedSend) MarshalJSON() ([]byte, error) {
	type plain QueuedSend
	return api.MarshalAliased(plain(s), queuedSendAliases)
}

func (r *SendWorkRequest) UnmarshalJSON(data []byte) error {
	type plain SendWorkRequest
	return api.UnmarshalAliased(data, (*plain)(r), sendWorkAliases)
}

func (n WithheldNotice) MarshalJSON() ([]byte, error) {
	type plain WithheldNotice
	return api.MarshalAliased(plain(n), withheldNoticeAliases)
}

func (a WorkConnectedApp) MarshalJSON() ([]byte, error) {
	type plain WorkConnectedApp
	return api.MarshalAliased(plain(a), workConnectedAppAliases)
}

func (l WorkConnectedAppList) MarshalJSON() ([]byte, error) {
	type plain WorkConnectedAppList
	return api.MarshalAliased(plain(l), workConnectedAppListAliases)
}
