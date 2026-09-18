package upload

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/assets", d.Upload, h.CreateWork)
	routes.Handle(http.MethodGet, "/v1/assets/:id/revisions", d.JSON, h.GetWorkReplacement)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions", d.Upload, h.AddWorkRevision)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions/:operationId/accept", d.JSON, h.AcceptWorkRevision)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/revisions/:operationId", d.JSON, h.CancelWorkRevision)
	routes.Handle(http.MethodGet, "/v1/assets/:id/vault", d.JSON, h.ListVaultPictures)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/vault/:pictureId", d.JSON, h.DiscardVaultPicture)
	routes.Handle(http.MethodPost, "/v1/assets/:id/vault/:pictureId/place", d.JSON, h.PlaceVaultPicture)
}

// The field names an upload answered to before the rename, kept for sixty days
var (
	uploadedWorkAliases = map[string]string{"type": "kind", "visibility": "discovery"}
	createWorkAliases   = map[string]string{"visibility": "discovery"}
	startWorkAliases    = map[string]string{"type": "kind"}
)

func (w Work) MarshalJSON() ([]byte, error) {
	type plain Work
	return api.MarshalAliased(plain(w), uploadedWorkAliases)
}

// aliasIngestKeys repeats an upload's answer under the key it had before the rename
func aliasIngestKeys(response gin.H) gin.H {
	response["asset"] = response["work"]
	return response
}

func (r *CreateWorkRequest) UnmarshalJSON(data []byte) error {
	type plain CreateWorkRequest
	return api.UnmarshalAliased(data, (*plain)(r), createWorkAliases)
}

func (r *StartWorkRequest) UnmarshalJSON(data []byte) error {
	type plain StartWorkRequest
	return api.UnmarshalAliased(data, (*plain)(r), startWorkAliases)
}
