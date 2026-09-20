package page

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// The field names a work's page answered to before the rename, kept for sixty days
var (
	workDetailAliases = map[string]string{
		"type": "kind", "visibility": "discovery", "follow": "watch",
		"draftedChangesVersion": "workingCopyVersion", "latestVersion": "latestUpdate",
		"appFormats": "appTargets", "hasPrivatePrompts": "linkedInstallOnly", "preservedPrompts": "sealedBlocks",
	}
	workListAliases          = map[string]string{"nsfwPreference": "visibility", "apps": "platforms"}
	browseWorkAliases        = map[string]string{"type": "kind"}
	deletedWorkAliases       = map[string]string{"type": "kind"}
	dependencyAliases        = map[string]string{"works": "assets"}
	visibilityRequestAliases = map[string]string{"visibility": "discovery"}
)

// registerAliases serves the paths a work's page had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets", d.JSON, h.ListWorks)
	routes.Handle(http.MethodDelete, "/v1/assets/:id", d.JSON, h.DeleteWork)
	routes.Handle(http.MethodGet, "/v1/assets/:id", d.JSON, h.GetWork)
	routes.Handle(http.MethodPost, "/v1/assets/:id/restore", d.JSON, h.RestoreWork)
	routes.Handle(http.MethodPut, "/v1/assets/:id/identity", d.JSON, h.SetWorkDetails)
	routes.Handle(http.MethodPost, "/v1/assets/:id/publish", d.JSON, h.PublishWork)
	routes.Handle(http.MethodPut, "/v1/assets/:id/discovery", d.JSON, h.SetWorkVisibility)
	routes.Handle(http.MethodGet, "/v1/assets/:id/preserved", d.JSON, h.ListPreservedData)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/preserved/:namespace", d.JSON, h.DeletePreservedData)
}

func aliasBrowseQuery(q *api.Query) {
	q.Alias("type", "kind")
	q.Alias("app", "platform")
}

func aliasPageQuery(q *api.Query) {
	q.Alias("draftedChanges", "workingCopy")
}

func (w WorkDetail) MarshalJSON() ([]byte, error) {
	type plain WorkDetail
	return api.MarshalAliased(plain(w), workDetailAliases)
}

func (l WorkList) MarshalJSON() ([]byte, error) {
	type plain WorkList
	return api.MarshalAliased(plain(l), workListAliases)
}

func (w BrowseWork) MarshalJSON() ([]byte, error) {
	type plain BrowseWork
	return api.MarshalAliased(plain(w), browseWorkAliases)
}

func (w DeletedWork) MarshalJSON() ([]byte, error) {
	type plain DeletedWork
	return api.MarshalAliased(plain(w), deletedWorkAliases)
}

func (d ExtensionDependency) MarshalJSON() ([]byte, error) {
	type plain ExtensionDependency
	return api.MarshalAliased(plain(d), dependencyAliases)
}

func (r *WorkVisibilityRequest) UnmarshalJSON(data []byte) error {
	type plain WorkVisibilityRequest
	return api.UnmarshalAliased(data, (*plain)(r), visibilityRequestAliases)
}
