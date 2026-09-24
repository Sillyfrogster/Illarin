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
	routes.Handle(http.MethodPost, "/v1/works", d.Upload, h.CreateWork)
	routes.Handle(http.MethodGet, "/v1/build-choices", d.JSON, h.GetBuildChoices)
	routes.Handle(http.MethodGet, "/v1/works/:id/original-file", d.JSON, h.GetWorkReplacement)
	routes.Handle(http.MethodPost, "/v1/works/:id/original-file", d.Upload, h.AddWorkOriginalFile)
	routes.Handle(http.MethodPost, "/v1/works/:id/original-file/:operationId/accept", d.JSON, h.AcceptWorkOriginalFile)
	routes.Handle(http.MethodDelete, "/v1/works/:id/original-file/:operationId", d.JSON, h.CancelWorkOriginalFile)
	routes.Handle(http.MethodGet, "/v1/works/:id/shelf", d.JSON, h.ListShelf)
	routes.Handle(http.MethodPost, "/v1/works/:id/shelf", d.JSON, h.AddMarkdown)
	routes.Handle(http.MethodDelete, "/v1/works/:id/shelf/imports/:importId", d.JSON, h.LetGoOfShelfImport)
	routes.Handle(http.MethodDelete, "/v1/works/:id/shelf/pieces/:pieceId", d.JSON, h.LetGoOfShelfPiece)
	routes.Handle(http.MethodPost, "/v1/works/:id/shelf/pieces/:pieceId/place", d.JSON, h.PlaceShelfPiece)
	routes.Handle(http.MethodPost, "/v1/works/:id/shelf/pieces/:pieceId/undo", d.JSON, h.UndoShelfPlacement)
	routes.Handle(http.MethodGet, "/v1/uploads/:id", d.JSON, h.GetUpload)
	registerAliases(routes, h)
}
