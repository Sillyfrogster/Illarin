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
	routes.Handle(http.MethodGet, "/v1/works/:id/versions", d.JSON, h.ListWorkVersions)
	routes.Handle(http.MethodPost, "/v1/works/:id/versions", d.JSON, h.PublishWorkVersion)
	routes.Handle(http.MethodGet, "/v1/works/:id/versions/comparison", d.JSON, h.CompareWorkVersions)
	routes.Handle(http.MethodPost, "/v1/works/:id/versions/:number/restore", d.JSON, h.RestoreWorkVersion)
	routes.Handle(http.MethodPatch, "/v1/works/:id/versions/:number/notes", d.JSON, h.CorrectWorkVersionNotes)
	routes.Handle(http.MethodPost, "/v1/works/:id/versions/:number/withdraw", d.JSON, h.WithdrawWorkVersion)
	routes.Handle(http.MethodGet, "/v1/works/:id/versions/private-prompts", d.JSON, h.ListPrivatePromptMismatches)
	routes.Handle(http.MethodPut, "/v1/works/:id/versions/:number/private-prompts", d.JSON, h.ResolvePromptCorrespondence)
	registerAliases(routes, h)
}
