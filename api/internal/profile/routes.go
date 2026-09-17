package profile

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

type Handlers struct {
	accounts       *account.Service
	maxUploadBytes int64
}

func NewHandlers(accounts *account.Service, maxUploadBytes int64) *Handlers {
	return &Handlers{accounts: accounts, maxUploadBytes: maxUploadBytes}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/profiles/:handle", d.JSON, h.GetProfile)
	routes.Handle(http.MethodPut, "/v1/account/profile", d.JSON, h.SavePublicProfile)
	routes.Handle(http.MethodPut, "/v1/account/profile/avatar", d.Upload, h.SetProfileAvatar)
	routes.Handle(http.MethodDelete, "/v1/account/profile/avatar", d.JSON, h.RemoveProfileAvatar)
}
