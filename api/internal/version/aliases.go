package version

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the renames, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/updates", d.JSON, h.ListWorkVersions)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates", d.JSON, h.PublishWorkVersion)
	routes.Handle(http.MethodGet, "/v1/works/:id/updates/comparison", d.JSON, h.CompareWorkVersions)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates/:number/restore", d.JSON, h.RestoreWorkVersion)
	routes.Handle(http.MethodPatch, "/v1/works/:id/updates/:number/notes", d.JSON, h.CorrectWorkVersionNotes)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates/:number/withdraw", d.JSON, h.WithdrawWorkVersion)
	routes.Handle(http.MethodGet, "/v1/works/:id/updates/protection", d.JSON, h.ListPrivatePromptMismatches)
	routes.Handle(http.MethodPut, "/v1/works/:id/updates/:number/protection", d.JSON, h.ResolvePromptCorrespondence)
	routes.Handle(http.MethodGet, "/v1/works/:id/versions/protection", d.JSON, h.ListPrivatePromptMismatches)
	routes.Handle(http.MethodPut, "/v1/works/:id/versions/:number/protection", d.JSON, h.ResolvePromptCorrespondence)
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates", d.JSON, h.ListWorkVersions)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates", d.JSON, h.PublishWorkVersion)
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/comparison", d.JSON, h.CompareWorkVersions)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates/:number/restore", d.JSON, h.RestoreWorkVersion)
	routes.Handle(http.MethodPatch, "/v1/assets/:id/updates/:number/notes", d.JSON, h.CorrectWorkVersionNotes)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates/:number/withdraw", d.JSON, h.WithdrawWorkVersion)
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/protection", d.JSON, h.ListPrivatePromptMismatches)
	routes.Handle(http.MethodPut, "/v1/assets/:id/updates/:number/protection", d.JSON, h.ResolvePromptCorrespondence)
}

// The field name a change in a comparison answered to before the rename, kept for sixty days
var versionChangeAliases = map[string]string{"type": "kind"}

func (c VersionChange) MarshalJSON() ([]byte, error) {
	type plain VersionChange
	return api.MarshalAliased(plain(c), versionChangeAliases)
}
