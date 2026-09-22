// Package full registers every route the server has, for tests outside the package that holds them
package full

import (
	"net/http"

	protocol "github.com/Sillyfrogster/Illarin/api"
	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/image"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/profile"
	"github.com/Sillyfrogster/Illarin/api/internal/staff"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/gin-gonic/gin"
)

// Harness builds test routers that serve every route
var Harness = apitest.Harness{Register: Register}

func Register(r *gin.Engine, s apitest.Services, d api.Deadlines) error {
	if err := d.Check(); err != nil {
		return err
	}
	routes := api.NewRoutes(r.Group(
		"",
		api.NoStoreCredentialResponses(),
		api.GuardBrowserMutations(s.Apps.BrowserOrigin()),
		api.Sessions(s.Accounts.Current),
	), d)
	routes.Handle(http.MethodGet, "/healthz", d.JSON, ok)
	routes.Handle(http.MethodGet, "/readyz", d.JSON, ok)
	routes.Handle(http.MethodGet, "/protocol", d.JSON, protocolDocument)

	downloads := download.NewHandlers(s.Downloads, s.Accounts)
	posts := blog.NewHandlers(s.Blog, s.Accounts, s.MaxUploadBytes)

	account.Register(routes, account.NewHandlers(s.Accounts, s.Apps, s.Blog))
	profile.Register(routes, profile.NewHandlers(s.Accounts, s.Pages, s.Versions, s.Notifications, s.MaxUploadBytes))
	notify.Register(routes, notify.NewHandlers(s.Notifications, s.Sends))
	page.Register(routes, page.NewHandlers(s.Pages, s.Accounts, s.Sends, s.Notifications))
	edit.Register(routes, edit.NewHandlers(s.Blocks))
	version.Register(routes, version.NewHandlers(s.Versions, s.Accounts))
	upload.Register(routes, upload.NewHandlers(s.Uploads, s.Pages, s.MaxUploadBytes))
	upload.RegisterGitHubReleases(routes, s.GitHubReleases)
	download.Register(routes, downloads)
	image.Register(routes, image.NewHandlers(s.Works, s.Accounts, s.Blog, s.MaxUploadBytes))
	private.Register(routes, private.NewHandlers(private.NewService(s.Works.Pool())))
	connect.Register(routes, connect.NewHandlers(s.Apps, s.Sends, downloads))
	blog.Register(routes, posts)
	integration.Register(routes, integration.NewHandlers(s.Integrations))
	staff.Register(routes, staff.NewHandlers(staff.NewService(s.Works)))
	return nil
}

func ok(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func protocolDocument(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", protocol.Protocol)
}
