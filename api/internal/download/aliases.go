package download

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/:number/downloads", d.JSON, h.GetRecordedVersionDownloads)
}

// The field name a version's downloads answered to before the rename, kept for sixty days
var recordedDownloadsAliases = map[string]string{"type": "kind"}

func (d RecordedVersionDownloads) MarshalJSON() ([]byte, error) {
	type plain RecordedVersionDownloads
	return api.MarshalAliased(plain(d), recordedDownloadsAliases)
}
