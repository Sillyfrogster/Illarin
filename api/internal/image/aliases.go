package image

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/media", d.JSON, h.ListMedia)
	routes.Handle(http.MethodPost, "/v1/assets/:id/media", d.Upload, h.AddMedia)
}

// The field name a picture answered to before the rename, kept for sixty days
var mediaAliases = map[string]string{"workId": "assetId"}

func (m Media) MarshalJSON() ([]byte, error) {
	type plain Media
	return api.MarshalAliased(plain(m), mediaAliases)
}
