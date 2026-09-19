package private

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/sealed", d.JSON, h.ExportPreservedPrompts)
	routes.Handle(http.MethodGet, "/v1/works/:id/sealed", d.JSON, h.ExportPreservedPrompts)
}

// The field names the private prompts export answered to before the rename, kept for sixty days
var preservedPromptsExportAliases = map[string]string{"work_id": "asset_id", "work_name": "asset_name"}

func (e preservedPromptsExport) MarshalJSON() ([]byte, error) {
	type plain preservedPromptsExport
	return api.MarshalAliased(plain(e), preservedPromptsExportAliases)
}
