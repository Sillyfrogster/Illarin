package account

import (
	"context"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/google/uuid"
)

// Writers says whether an account's writer switch is on
type Writers interface {
	IsWriter(ctx context.Context, accountID uuid.UUID) (bool, error)
}

type Handlers struct {
	accounts *Service
	apps     *connect.Apps
	blog     Writers
}

func NewHandlers(accounts *Service, apps *connect.Apps, posts Writers) *Handlers {
	return &Handlers{accounts: accounts, apps: apps, blog: posts}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodPost, "/v1/auth/sign-up", d.JSON, h.SignUp)
	routes.Handle(http.MethodPost, "/v1/auth/sign-in", d.JSON, h.SignIn)
	routes.Handle(http.MethodGet, "/v1/auth/discord", d.JSON, h.BeginDiscord)
	routes.Handle(http.MethodGet, "/v1/auth/discord/callback", d.JSON, h.CompleteDiscord)
	routes.Handle(http.MethodPost, "/v1/auth/sign-out", d.JSON, h.SignOut)
	routes.Handle(http.MethodGet, "/v1/auth/session", d.JSON, h.GetSession)
	routes.Handle(http.MethodPost, "/v1/auth/verify-email", d.JSON, h.VerifyEmail)
	routes.Handle(http.MethodPost, "/v1/auth/password-reset", d.JSON, h.RequestPasswordReset)
	routes.Handle(http.MethodPost, "/v1/auth/password-reset/complete", d.JSON, h.CompletePasswordReset)
	routes.Handle(http.MethodDelete, "/v1/account/discord", d.JSON, h.DetachDiscord)
	routes.Handle(http.MethodPatch, "/v1/account/email", d.JSON, h.ChangeUnverifiedEmail)
	routes.Handle(http.MethodPatch, "/v1/account/handle", d.JSON, h.RenameHandle)
	routes.Handle(http.MethodPut, "/v1/account/password", d.JSON, h.SetPassword)
	routes.Handle(http.MethodPut, "/v1/account/nsfw-preference", d.JSON, h.SetNsfwPreference)
	routes.Handle(http.MethodPut, "/v1/account/app-preference", d.JSON, h.SetAppPreference)
	routes.Handle(http.MethodGet, "/v1/account/preferences", d.JSON, h.GetPreferences)
	registerAliases(routes, h)
}
