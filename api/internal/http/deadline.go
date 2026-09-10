package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/openapi"
	"github.com/gin-gonic/gin"
)

type Deadlines struct {
	JSON     time.Duration
	Upload   time.Duration
	Download time.Duration
	Deliver  time.Duration
	Verify   time.Duration
}

type Readiness func(context.Context) error

func DefaultDeadlines() Deadlines {
	return Deadlines{
		JSON:     5 * time.Second,
		Upload:   15 * time.Minute,
		Download: 15 * time.Minute,
		Deliver:  45 * time.Second,
		Verify:   20 * time.Second,
	}
}

func Register(r *gin.Engine, h *Handlers, d Deadlines, readiness Readiness) error {
	if readiness == nil {
		return fmt.Errorf("readiness check is required")
	}
	limits := map[string]time.Duration{
		routeKey(http.MethodGet, "/healthz"):                                                 d.JSON,
		routeKey(http.MethodGet, "/readyz"):                                                  d.JSON,
		routeKey(http.MethodGet, "/protocol"):                                                d.JSON,
		routeKey(http.MethodGet, "/openapi.yaml"):                                            d.JSON,
		routeKey(http.MethodPost, "/v1/link/requests"):                                       d.JSON,
		routeKey(http.MethodPost, "/v1/link/poll"):                                           d.JSON,
		routeKey(http.MethodPost, "/v1/link/authorizations"):                                 d.JSON,
		routeKey(http.MethodGet, "/v1/link/authorizations/:requestCode"):                     d.JSON,
		routeKey(http.MethodPost, "/v1/link/authorizations/:requestCode/approve"):            d.JSON,
		routeKey(http.MethodPost, "/v1/link/authorizations/:requestCode/deny"):               d.JSON,
		routeKey(http.MethodPost, "/v1/link/token"):                                          d.JSON,
		routeKey(http.MethodPost, "/v1/link/refresh"):                                        d.JSON,
		routeKey(http.MethodGet, "/v1/link/requests/:userCode"):                              d.JSON,
		routeKey(http.MethodPost, "/v1/link/requests/:userCode/approve"):                     d.JSON,
		routeKey(http.MethodPost, "/v1/link/requests/:userCode/deny"):                        d.JSON,
		routeKey(http.MethodGet, "/v1/instances"):                                            d.JSON,
		routeKey(http.MethodGet, "/v1/instances/me"):                                         d.JSON,
		routeKey(http.MethodPut, "/v1/instances/me"):                                         d.JSON,
		routeKey(http.MethodDelete, "/v1/instances/:id"):                                     d.JSON,
		routeKey(http.MethodPost, "/v1/deliveries/collect"):                                  d.Deliver,
		routeKey(http.MethodDelete, "/v1/deliveries/:id"):                                    d.JSON,
		routeKey(http.MethodPost, "/v1/library/sync"):                                        d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/instances"):                                 d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/deliveries"):                               d.JSON,
		routeKey(http.MethodPatch, "/v1/account/email"):                                      d.JSON,
		routeKey(http.MethodPatch, "/v1/account/handle"):                                     d.JSON,
		routeKey(http.MethodPut, "/v1/account/password"):                                     d.JSON,
		routeKey(http.MethodPut, "/v1/account/nsfw-visibility"):                              d.JSON,
		routeKey(http.MethodPut, "/v1/account/profile"):                                      d.JSON,
		routeKey(http.MethodPut, "/v1/account/profile/avatar"):                               d.Upload,
		routeKey(http.MethodDelete, "/v1/account/profile/avatar"):                            d.JSON,
		routeKey(http.MethodDelete, "/v1/account/discord"):                                   d.JSON,
		routeKey(http.MethodGet, "/v1/auth/session"):                                         d.JSON,
		routeKey(http.MethodGet, "/v1/auth/discord"):                                         d.JSON,
		routeKey(http.MethodGet, "/v1/auth/discord/callback"):                                d.JSON,
		routeKey(http.MethodPost, "/v1/auth/sign-in"):                                        d.JSON,
		routeKey(http.MethodPost, "/v1/auth/sign-out"):                                       d.JSON,
		routeKey(http.MethodPost, "/v1/auth/sign-up"):                                        d.JSON,
		routeKey(http.MethodPost, "/v1/auth/verify-email"):                                   d.JSON,
		routeKey(http.MethodPost, "/v1/auth/password-reset"):                                 d.JSON,
		routeKey(http.MethodPost, "/v1/auth/password-reset/complete"):                        d.JSON,
		routeKey(http.MethodGet, "/v1/distinctions"):                                         d.JSON,
		routeKey(http.MethodPost, "/v1/distinctions"):                                        d.JSON,
		routeKey(http.MethodPut, "/v1/distinctions"):                                         d.JSON,
		routeKey(http.MethodPatch, "/v1/distinctions/:id"):                                   d.JSON,
		routeKey(http.MethodPut, "/v1/distinctions/:id/mark"):                                d.Upload,
		routeKey(http.MethodDelete, "/v1/distinctions/:id/mark"):                             d.JSON,
		routeKey(http.MethodGet, "/v1/accounts/:handle/distinctions"):                        d.JSON,
		routeKey(http.MethodPost, "/v1/accounts/:handle/distinctions"):                       d.JSON,
		routeKey(http.MethodPut, "/v1/accounts/:handle/distinctions"):                        d.JSON,
		routeKey(http.MethodDelete, "/v1/accounts/:handle/distinctions/:assignmentId"):       d.JSON,
		routeKey(http.MethodGet, "/v1/publication/apps"):                                     d.JSON,
		routeKey(http.MethodPost, "/v1/publication/apps"):                                    d.JSON,
		routeKey(http.MethodPut, "/v1/publication/apps"):                                     d.JSON,
		routeKey(http.MethodPatch, "/v1/publication/apps/:id"):                               d.JSON,
		routeKey(http.MethodPut, "/v1/publication/apps/:id/mark"):                            d.Upload,
		routeKey(http.MethodGet, "/v1/publication/categories"):                               d.JSON,
		routeKey(http.MethodPut, "/v1/publication/categories"):                               d.JSON,
		routeKey(http.MethodPatch, "/v1/publication/categories/:id"):                         d.JSON,
		routeKey(http.MethodGet, "/v1/publication/grants"):                                   d.JSON,
		routeKey(http.MethodPost, "/v1/publication/grants"):                                  d.JSON,
		routeKey(http.MethodPatch, "/v1/publication/grants/:id"):                             d.JSON,
		routeKey(http.MethodDelete, "/v1/publication/grants/:id"):                            d.JSON,
		routeKey(http.MethodGet, "/v1/publication/grants/:id/tokens"):                        d.JSON,
		routeKey(http.MethodPost, "/v1/publication/grants/:id/tokens"):                       d.JSON,
		routeKey(http.MethodDelete, "/v1/publication/tokens/:id"):                            d.JSON,
		routeKey(http.MethodGet, "/v1/publication/token"):                                    d.JSON,
		routeKey(http.MethodGet, "/v1/publication/workspace"):                                d.JSON,
		routeKey(http.MethodGet, "/v1/publication/destinations"):                             d.JSON,
		routeKey(http.MethodPost, "/v1/publication/destinations"):                            d.JSON,
		routeKey(http.MethodPatch, "/v1/publication/destinations/:id"):                       d.JSON,
		routeKey(http.MethodDelete, "/v1/publication/destinations/:id"):                      d.JSON,
		routeKey(http.MethodPost, "/v1/publication/destinations/:id/verification"):           d.Verify,
		routeKey(http.MethodDelete, "/v1/publication/destinations/:id/verification"):         d.JSON,
		routeKey(http.MethodPost, "/v1/publication/destinations/:id/secret"):                 d.JSON,
		routeKey(http.MethodPost, "/v1/publication/channels"):                                d.Verify,
		routeKey(http.MethodPatch, "/v1/publication/channels/:id"):                           d.Verify,
		routeKey(http.MethodGet, "/v1/publication/deliveries"):                               d.JSON,
		routeKey(http.MethodGet, "/v1/publication/deliveries/:id/attempts"):                  d.JSON,
		routeKey(http.MethodPost, "/v1/publication/deliveries/:id/replay"):                   d.JSON,
		routeKey(http.MethodPut, "/v1/publication/apps/:id/destinations"):                    d.JSON,
		routeKey(http.MethodPut, "/v1/publication/grants/:id/destinations"):                  d.JSON,
		routeKey(http.MethodGet, "/v1/publication/posts"):                                    d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts"):                                   d.JSON,
		routeKey(http.MethodGet, "/v1/publication/posts/:id"):                                d.JSON,
		routeKey(http.MethodPut, "/v1/publication/posts/:id"):                                d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/media"):                         d.Upload,
		routeKey(http.MethodGet, "/v1/publication/posts/:id/revisions"):                      d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/revisions"):                     d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/revisions/:revisionId/restore"): d.JSON,
		routeKey(http.MethodGet, "/v1/publication/posts/:id/history"):                        d.JSON,
		routeKey(http.MethodGet, "/v1/publication/posts/:id/destinations"):                   d.JSON,
		routeKey(http.MethodGet, "/v1/publication/posts/:id/deliveries"):                     d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/import"):                        d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/publish"):                       d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/withdraw"):                      d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/republish"):                     d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/delete"):                        d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/recover"):                       d.JSON,
		routeKey(http.MethodPost, "/v1/publication/posts/:id/schedule"):                      d.JSON,
		routeKey(http.MethodPut, "/v1/publication/posts/:id/schedule"):                       d.JSON,
		routeKey(http.MethodDelete, "/v1/publication/posts/:id/schedule"):                    d.JSON,
		routeKey(http.MethodPut, "/v1/publication/posts/:id/address"):                        d.JSON,
		routeKey(http.MethodPut, "/v1/publication/posts/:id/byline"):                         d.JSON,
		routeKey(http.MethodGet, "/v1/post-categories"):                                      d.JSON,
		routeKey(http.MethodGet, "/v1/post-apps"):                                            d.JSON,
		routeKey(http.MethodGet, "/v1/posts"):                                                d.JSON,
		routeKey(http.MethodGet, "/v1/posts/:slug"):                                          d.JSON,
		routeKey(http.MethodGet, "/v1/profiles/:handle"):                                     d.JSON,
		routeKey(http.MethodGet, "/v1/profiles/:handle/deleted"):                             d.JSON,
		routeKey(http.MethodGet, "/v1/profiles/:handle/restriction"):                         d.JSON,
		routeKey(http.MethodPut, "/v1/profiles/:handle/restriction"):                         d.JSON,
		routeKey(http.MethodDelete, "/v1/profiles/:handle/restriction"):                      d.JSON,
		routeKey(http.MethodGet, "/v1/assets"):                                               d.JSON,
		routeKey(http.MethodPost, "/v1/assets"):                                              d.Upload,
		routeKey(http.MethodGet, "/v1/assets/:id"):                                           d.JSON,
		routeKey(http.MethodDelete, "/v1/assets/:id"):                                        d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/blocks"):                                   d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/blocks"):                                    d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/blocks/:blockId"):                           d.JSON,
		routeKey(http.MethodDelete, "/v1/assets/:id/blocks/:blockId"):                        d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/blocks/:blockId/move-and-remove"):          d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/revisions"):                                d.Upload,
		routeKey(http.MethodGet, "/v1/assets/:id/revisions"):                                 d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/revisions/:operationId/accept"):            d.JSON,
		routeKey(http.MethodDelete, "/v1/assets/:id/revisions/:operationId"):                 d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/restore"):                                  d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/discovery"):                                 d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/identity"):                                  d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/publish"):                                  d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/updates"):                                  d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/updates"):                                   d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/updates/comparison"):                        d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/updates/protection"):                        d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/updates/:number/protection"):                d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/preserved"):                                 d.JSON,
		routeKey(http.MethodDelete, "/v1/assets/:id/preserved/:namespace"):                   d.JSON,
		routeKey(http.MethodGet, "/v1/assets/:id/sealed"):                                    d.JSON,
		routeKey(http.MethodPut, "/v1/assets/:id/withhold"):                                  d.JSON,
		routeKey(http.MethodDelete, "/v1/assets/:id/withhold"):                               d.JSON,
		routeKey(http.MethodPost, "/v1/assets/:id/media"):                                    d.Upload,
		routeKey(http.MethodGet, "/v1/assets/:id/media"):                                     d.JSON,
		routeKey(http.MethodGet, "/v1/legacy-assets/:author/:name"):                          d.JSON,
		routeKey(http.MethodGet, "/v1/legacy-profiles/:discordId"):                           d.JSON,
		routeKey(http.MethodGet, "/v1/ingests/:id"):                                          d.JSON,
		routeKey(http.MethodPatch, "/v1/ingests/:id"):                                        d.JSON,
		routeKey(http.MethodGet, "/download/:id"):                                            d.Download,
		routeKey(http.MethodGet, "/download/:id/:target"):                                    d.Download,
		routeKey(http.MethodGet, "/media/:media_id/:variant/:derivative_version"):            d.Download,
		routeKey(http.MethodGet, "/delivery/:id/export"):                                     d.Download,
	}

	routes := r.Group(
		"",
		deadlineByRoute(limits),
		noStoreCredentialResponses(),
		h.guardBrowserMutations(),
		h.publicationAPI(),
	)
	routes.GET("/healthz", health)
	routes.GET("/readyz", ready(readiness))
	routes.GET("/protocol", document("text/plain; charset=utf-8", openapi.Guide))
	routes.GET("/openapi.yaml", document("application/yaml", openapi.Contract))
	RegisterHandlersWithOptions(routes, h, GinServerOptions{ErrorHandler: generatedParameterError})

	for _, route := range r.Routes() {
		key := routeKey(route.Method, route.Path)
		if limits[key] == 0 {
			return fmt.Errorf("%s has no deadline, give it one in Register", key)
		}
	}
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

func withDeadline(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(d))

		c.Next()
	}
}

func deadlineByRoute(limits map[string]time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		d, ok := limits[routeKey(c.Request.Method, c.FullPath())]
		if !ok {
			c.Next()
			return
		}
		withDeadline(d)(c)
	}
}

func routeKey(method, path string) string { return method + " " + path }
