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
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates", d.JSON, h.ListAssetUpdates)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates", d.JSON, h.PublishAssetUpdate)
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/comparison", d.JSON, h.CompareAssetVersions)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates/:number/restore", d.JSON, h.RestoreAssetVersion)
	routes.Handle(http.MethodPatch, "/v1/assets/:id/updates/:number/notes", d.JSON, h.CorrectAssetVersionNotes)
	routes.Handle(http.MethodPost, "/v1/assets/:id/updates/:number/withdraw", d.JSON, h.WithdrawAssetVersion)
	routes.Handle(http.MethodGet, "/v1/assets/:id/updates/protection", d.JSON, h.ListProtectionMismatches)
	routes.Handle(http.MethodPut, "/v1/assets/:id/updates/:number/protection", d.JSON, h.ResolvePromptCorrespondence)
}
