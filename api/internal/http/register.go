package http

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
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/profile"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

type Readiness func(context.Context) error

func Register(r *gin.Engine, h *Handlers, d api.Deadlines, readiness Readiness) error {
	if readiness == nil {
		return fmt.Errorf("readiness check is required")
	}
	if err := d.Check(); err != nil {
		return err
	}

	routes := api.NewRoutes(r.Group(
		"",
		noStoreCredentialResponses(),
		api.GuardBrowserMutations(h.links.BrowserOrigin()),
		api.Sessions(h.accounts.Current),
	), d)
	routes.Handle(http.MethodGet, "/healthz", d.JSON, health)
	routes.Handle(http.MethodGet, "/readyz", d.JSON, ready(readiness))
	routes.Handle(http.MethodGet, "/protocol", d.JSON, document("text/plain; charset=utf-8", protocol.Protocol))
	account.Register(routes, account.NewHandlers(h.accounts, h.links, h.publications))
	profile.Register(routes, profile.NewHandlers(h.accounts, h.maxUploadBytes))
	notify.Register(routes, notify.NewHandlers(h.notifications))
	work.Register(routes, work.NewHandlers(h.works, h.accounts, h.deliveries, h.notifications))
	edit.Register(routes, edit.NewHandlers(h.blocks))
	version.Register(routes, version.NewHandlers(h.versions, h.accounts))
	upload.Register(routes, upload.NewHandlers(h.uploads, h.works, h.maxUploadBytes))
	download.Register(routes, h.downloads)
	private.Register(routes, private.NewHandlers(private.NewService(h.assets.Pool())))
	connect.Register(routes, connect.NewHandlers(h.links, h.deliveries, h.downloads))
	blogHandlers := blog.NewHandlers(h.publications, h.accounts, h.maxUploadBytes)
	blog.Register(routes, blogHandlers)
	integration.Register(routes, integration.NewHandlers(h.publications, h.updateDestinations, blogHandlers.IntegrationAccess()))
	registerRoutes(routes, h)
	return nil
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ready(check Readiness) gin.HandlerFunc {
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
