// Package image serves the pictures a work, a profile or a post owns
package image

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
)

type Handlers struct {
	assets         *asset.Service
	accounts       *account.Service
	publications   *blog.Service
	maxUploadBytes int64
}

func NewHandlers(
	assets *asset.Service,
	accounts *account.Service,
	publications *blog.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		assets:         assets,
		accounts:       accounts,
		publications:   publications,
		maxUploadBytes: maxUploadBytes,
	}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/media", d.JSON, h.ListMedia)
	routes.Handle(http.MethodPost, "/v1/assets/:id/media", d.Upload, h.AddMedia)
	routes.Handle(http.MethodGet, "/media/:media_id/:variant/:derivative_version", d.Download, h.GetMediaVariant)
}
