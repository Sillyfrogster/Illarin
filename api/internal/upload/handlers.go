package upload

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

type Handlers struct {
	uploads        *Service
	works          *page.Service
	maxUploadBytes int64
}

func NewHandlers(uploads *Service, works *page.Service, maxUploadBytes int64) *Handlers {
	return &Handlers{uploads: uploads, works: works, maxUploadBytes: maxUploadBytes}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/assets", d.Upload, h.CreateWork)
	routes.Handle(http.MethodGet, "/v1/assets/:id/revisions", d.JSON, h.GetWorkReplacement)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions", d.Upload, h.AddWorkRevision)
	routes.Handle(http.MethodPost, "/v1/assets/:id/revisions/:operationId/accept", d.JSON, h.AcceptWorkRevision)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/revisions/:operationId", d.JSON, h.CancelWorkRevision)
	routes.Handle(http.MethodGet, "/v1/assets/:id/vault", d.JSON, h.ListVaultPictures)
	routes.Handle(http.MethodDelete, "/v1/assets/:id/vault/:pictureId", d.JSON, h.DiscardVaultPicture)
	routes.Handle(http.MethodPost, "/v1/assets/:id/vault/:pictureId/place", d.JSON, h.PlaceVaultPicture)
	routes.Handle(http.MethodGet, "/v1/ingests/:id", d.JSON, h.GetIngest)
}
