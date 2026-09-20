// Package image serves the pictures a work, a profile or a post owns
package image

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
)

type Handlers struct {
	works          *work.Service
	accounts       *account.Service
	posts          *blog.Service
	maxUploadBytes int64
}

func NewHandlers(
	works *work.Service,
	accounts *account.Service,
	posts *blog.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		works:          works,
		accounts:       accounts,
		posts:          posts,
		maxUploadBytes: maxUploadBytes,
	}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/media", d.JSON, h.ListMedia)
	routes.Handle(http.MethodPost, "/v1/works/:id/media", d.Upload, h.AddMedia)
	routes.Handle(http.MethodGet, "/media/:media_id/:variant/:derivative_version", d.Download, h.GetMediaVariant)
	registerAliases(routes, h)
}
