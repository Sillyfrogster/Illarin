package upload

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

type Handlers struct {
	uploads        *Service
	works          *work.Service
	maxUploadBytes int64
}

func NewHandlers(uploads *Service, works *work.Service, maxUploadBytes int64) *Handlers {
	return &Handlers{uploads: uploads, works: works, maxUploadBytes: maxUploadBytes}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/assets", d.Upload, h.CreateAsset)
	routes.Handle(http.MethodGet, "/v1/assets/:id/revisions", d.JSON, h.GetAssetReplacement)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions", d.Upload, h.AddAssetRevision)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions/:operationId/accept", d.JSON, h.AcceptAssetRevision)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/revisions/:operationId", d.JSON, h.CancelAssetRevision)
	routes.Handle(http.MethodGet, "/v1/assets/:id/vault", d.JSON, h.ListVaultPictures)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/vault/:pictureId", d.JSON, h.DiscardVaultPicture)
	routes.Handle(http.MethodPost, "/v1/assets/:id/vault/:pictureId/place", d.JSON, h.PlaceVaultPicture)
	routes.Handle(http.MethodGet, "/v1/ingests/:id", d.JSON, h.GetIngest)
}
