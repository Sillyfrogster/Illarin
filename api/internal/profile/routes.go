package profile

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
)

type Handlers struct {
	accounts       *account.Service
	pages          *page.Service
	versions       *version.Service
	notifications  *notify.Service
	maxUploadBytes int64
}

func NewHandlers(
	accounts *account.Service,
	pages *page.Service,
	versions *version.Service,
	notifications *notify.Service,
	maxUploadBytes int64,
) *Handlers {
	return &Handlers{
		accounts: accounts, pages: pages, versions: versions, notifications: notifications,
		maxUploadBytes: maxUploadBytes,
	}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/profiles/:handle", d.JSON, h.GetProfile)
	routes.Handle(http.MethodPut, "/v1/profiles/:handle/follow", d.JSON, h.FollowCreator)
	routes.Handle(http.MethodDelete, "/v1/profiles/:handle/follow", d.JSON, h.StopFollowingCreator)
	routes.Handle(http.MethodPut, "/v1/account/profile", d.JSON, h.SavePublicProfile)
	routes.Handle(http.MethodPut, "/v1/account/profile/featured", d.JSON, h.SaveFeatured)
	routes.Handle(http.MethodPut, "/v1/account/profile/avatar", d.Upload, h.SetAvatar)
	routes.Handle(http.MethodDelete, "/v1/account/profile/avatar", d.JSON, h.RemoveAvatar)
	routes.Handle(http.MethodPut, "/v1/account/profile/banner", d.Upload, h.SetBanner)
	routes.Handle(http.MethodDelete, "/v1/account/profile/banner", d.JSON, h.RemoveBanner)
}
