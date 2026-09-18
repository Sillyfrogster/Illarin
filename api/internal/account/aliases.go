package account

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the path the NSFW preference had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	routes.Handle(http.MethodPut, "/v1/account/nsfw-visibility", routes.Deadlines.JSON, h.SetNsfwPreference)
}

// The field name the NSFW preference answered to before the rename, kept for sixty days
var nsfwPreferenceAliases = map[string]string{"preference": "visibility"}

func (r *NsfwPreferenceRequest) UnmarshalJSON(data []byte) error {
	type plain NsfwPreferenceRequest
	return api.UnmarshalAliased(data, (*plain)(r), nsfwPreferenceAliases)
}
