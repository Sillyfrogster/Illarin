package main

import (
	"context"
	"fmt"
	"net/http"

	protocol "github.com/Sillyfrogster/Illarin/api"
	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
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
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

// services holds the running parts the routes need
type services struct {
	Works              *work.Service
	Pages              *page.Service
	Blocks             *edit.Service
	Versions           *version.Service
	Uploads            *upload.Service
	Downloads          *download.Service
	Accounts           *account.Service
	Apps               *connect.Apps
	Sends              *connect.Sends
	Publications       *blog.Service
	UpdateDestinations *integration.Service
	Notifications      *notify.Service
	MaxUploadBytes     int64
}

// readiness answers whether the server can take traffic
type readiness func(context.Context) error

// registerRoutes puts the shared middleware on the engine and lets each feature register its own routes
func registerRoutes(r *gin.Engine, s services, d api.Deadlines, ready readiness) error {
	if ready == nil {
		return fmt.Errorf("readiness check is required")
	}
	if err := d.Check(); err != nil {
		return err
	}

	routes := api.NewRoutes(r.Group(
		"",
		api.NoStoreCredentialResponses(),
		api.GuardBrowserMutations(s.Apps.BrowserOrigin()),
		api.Sessions(s.Accounts.Current),
	), d)
	routes.Handle(http.MethodGet, "/healthz", d.JSON, alive)
	routes.Handle(http.MethodGet, "/readyz", d.JSON, readyOrNot(ready))
	routes.Handle(http.MethodGet, "/protocol", d.JSON, document("text/plain; charset=utf-8", protocol.Protocol))

	downloads := download.NewHandlers(s.Downloads, s.Accounts)
	posts := blog.NewHandlers(s.Publications, s.Accounts, s.MaxUploadBytes)

	account.Register(routes, account.NewHandlers(s.Accounts, s.Apps, s.Publications))
	profile.Register(routes, profile.NewHandlers(s.Accounts, s.MaxUploadBytes))
	notify.Register(routes, notify.NewHandlers(s.Notifications, s.Sends))
	page.Register(routes, page.NewHandlers(s.Pages, s.Accounts, s.Sends, s.Notifications))
	edit.Register(routes, edit.NewHandlers(s.Blocks))
	version.Register(routes, version.NewHandlers(s.Versions, s.Accounts))
	upload.Register(routes, upload.NewHandlers(s.Uploads, s.Pages, s.MaxUploadBytes))
	download.Register(routes, downloads)
	image.Register(routes, image.NewHandlers(s.Works, s.Accounts, s.Publications, s.MaxUploadBytes))
	private.Register(routes, private.NewHandlers(private.NewService(s.Works.Pool())))
	connect.Register(routes, connect.NewHandlers(s.Apps, s.Sends, downloads))
	blog.Register(routes, posts)
	integration.Register(routes, integration.NewHandlers(
		s.Publications, s.UpdateDestinations, posts.IntegrationAccess()))
	staff.Register(routes, staff.NewHandlers(staff.NewService(s.Works.Pool())))
	return nil
}

func alive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyOrNot(check readiness) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := check(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func document(mediaType string, body []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, mediaType, body)
	}
}
