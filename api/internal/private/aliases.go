package private

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/sealed", d.JSON, h.ExportSealedContent)
}

// The field names the private prompts export answered to before the rename, kept for sixty days
var sealedExportAliases = map[string]string{"work_id": "asset_id", "work_name": "asset_name"}

func (e sealedExport) MarshalJSON() ([]byte, error) {
	type plain sealedExport
	return api.MarshalAliased(plain(e), sealedExportAliases)
}
