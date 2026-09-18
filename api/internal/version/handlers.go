package version

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

type Handlers struct {
	versions *Service
	accounts *account.Service
}

func NewHandlers(versions *Service, accounts *account.Service) *Handlers {
	return &Handlers{versions: versions, accounts: accounts}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/updates", d.JSON, h.ListWorkUpdates)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates", d.JSON, h.PublishWorkUpdate)
	routes.Handle(http.MethodGet, "/v1/works/:id/updates/comparison", d.JSON, h.CompareWorkVersions)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates/:number/restore", d.JSON, h.RestoreWorkVersion)
	routes.Handle(http.MethodPatch, "/v1/works/:id/updates/:number/notes", d.JSON, h.CorrectWorkVersionNotes)
	routes.Handle(http.MethodPost, "/v1/works/:id/updates/:number/withdraw", d.JSON, h.WithdrawWorkVersion)
	routes.Handle(http.MethodGet, "/v1/works/:id/updates/protection", d.JSON, h.ListProtectionMismatches)
	routes.Handle(http.MethodPut, "/v1/works/:id/updates/:number/protection", d.JSON, h.ResolvePromptCorrespondence)
	registerAliases(routes, h)
}
