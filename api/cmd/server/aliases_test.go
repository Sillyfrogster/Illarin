package main

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

// renamedPaths is every path the API answered to before a rename, with the path it answers to now
var renamedPaths = []struct{ method, was, now string }{
	{"GET", "/v1/assets", "/v1/works"},
	{"POST", "/v1/assets", "/v1/works"},
	{"GET", "/v1/assets/:id", "/v1/works/:id"},
	{"DELETE", "/v1/assets/:id", "/v1/works/:id"},
	{"POST", "/v1/assets/:id/restore", "/v1/works/:id/restore"},
	{"PUT", "/v1/assets/:id/identity", "/v1/works/:id/details"},
	{"POST", "/v1/assets/:id/publish", "/v1/works/:id/publish"},
	{"PUT", "/v1/assets/:id/discovery", "/v1/works/:id/visibility"},
	{"GET", "/v1/assets/:id/preserved", "/v1/works/:id/preserved"},
	{"DELETE", "/v1/assets/:id/preserved/:namespace", "/v1/works/:id/preserved/:namespace"},
	{"PUT", "/v1/assets/:id/watch", "/v1/works/:id/follow"},
	{"DELETE", "/v1/assets/:id/watch", "/v1/works/:id/follow"},
	{"POST", "/v1/assets/:id/blocks", "/v1/works/:id/blocks"},
	{"PUT", "/v1/assets/:id/blocks", "/v1/works/:id/blocks"},
	{"PUT", "/v1/assets/:id/blocks/:blockId", "/v1/works/:id/blocks/:blockId"},
	{"DELETE", "/v1/assets/:id/blocks/:blockId", "/v1/works/:id/blocks/:blockId"},
	{"POST", "/v1/assets/:id/blocks/:blockId/move-and-remove", "/v1/works/:id/blocks/:blockId/move-and-remove"},
	{"GET", "/v1/assets/:id/media", "/v1/works/:id/media"},
	{"POST", "/v1/assets/:id/media", "/v1/works/:id/media"},
	{"GET", "/v1/assets/:id/revisions", "/v1/works/:id/original-file"},
	{"POST", "/v1/assets/:id/revisions", "/v1/works/:id/original-file"},
	{"POST", "/v1/assets/:id/revisions/:operationId/accept", "/v1/works/:id/original-file/:operationId/accept"},
	{"DELETE", "/v1/assets/:id/revisions/:operationId", "/v1/works/:id/original-file/:operationId"},
	{"GET", "/v1/works/:id/revisions", "/v1/works/:id/original-file"},
	{"POST", "/v1/works/:id/revisions", "/v1/works/:id/original-file"},
	{"POST", "/v1/works/:id/revisions/:operationId/accept", "/v1/works/:id/original-file/:operationId/accept"},
	{"DELETE", "/v1/works/:id/revisions/:operationId", "/v1/works/:id/original-file/:operationId"},
	{"GET", "/v1/assets/:id/vault", "/v1/works/:id/vault"},
	{"DELETE", "/v1/assets/:id/vault/:pictureId", "/v1/works/:id/vault/:pictureId"},
	{"POST", "/v1/assets/:id/vault/:pictureId/place", "/v1/works/:id/vault/:pictureId/place"},
	{"GET", "/v1/assets/:id/updates", "/v1/works/:id/versions"},
	{"POST", "/v1/assets/:id/updates", "/v1/works/:id/versions"},
	{"GET", "/v1/assets/:id/updates/comparison", "/v1/works/:id/versions/comparison"},
	{"GET", "/v1/assets/:id/updates/protection", "/v1/works/:id/versions/private-prompts"},
	{"PUT", "/v1/assets/:id/updates/:number/protection", "/v1/works/:id/versions/:number/private-prompts"},
	{"POST", "/v1/assets/:id/updates/:number/restore", "/v1/works/:id/versions/:number/restore"},
	{"POST", "/v1/assets/:id/updates/:number/withdraw", "/v1/works/:id/versions/:number/withdraw"},
	{"PATCH", "/v1/assets/:id/updates/:number/notes", "/v1/works/:id/versions/:number/notes"},
	{"GET", "/v1/assets/:id/updates/:number/downloads", "/v1/works/:id/versions/:number/downloads"},
	{"GET", "/v1/works/:id/updates", "/v1/works/:id/versions"},
	{"POST", "/v1/works/:id/updates", "/v1/works/:id/versions"},
	{"GET", "/v1/works/:id/updates/comparison", "/v1/works/:id/versions/comparison"},
	{"GET", "/v1/works/:id/updates/protection", "/v1/works/:id/versions/private-prompts"},
	{"PUT", "/v1/works/:id/updates/:number/protection", "/v1/works/:id/versions/:number/private-prompts"},
	{"POST", "/v1/works/:id/updates/:number/restore", "/v1/works/:id/versions/:number/restore"},
	{"POST", "/v1/works/:id/updates/:number/withdraw", "/v1/works/:id/versions/:number/withdraw"},
	{"PATCH", "/v1/works/:id/updates/:number/notes", "/v1/works/:id/versions/:number/notes"},
	{"GET", "/v1/works/:id/updates/:number/downloads", "/v1/works/:id/versions/:number/downloads"},
	{"GET", "/v1/assets/:id/sealed", "/v1/works/:id/preserved-prompts"},
	{"GET", "/v1/works/:id/sealed", "/v1/works/:id/preserved-prompts"},
	{"GET", "/v1/works/:id/versions/protection", "/v1/works/:id/versions/private-prompts"},
	{"PUT", "/v1/works/:id/versions/:number/protection", "/v1/works/:id/versions/:number/private-prompts"},
	{"GET", "/v1/assets/:id/instances", "/v1/works/:id/connected-apps"},
	{"POST", "/v1/assets/:id/deliveries", "/v1/works/:id/sends"},
	{"GET", "/v1/works/:id/instances", "/v1/works/:id/connected-apps"},
	{"POST", "/v1/works/:id/deliveries", "/v1/works/:id/sends"},
	{"POST", "/v1/link/requests", "/v1/connect/requests"},
	{"POST", "/v1/link/poll", "/v1/connect/poll"},
	{"GET", "/v1/link/requests/:userCode", "/v1/connect/requests/:userCode"},
	{"POST", "/v1/link/requests/:userCode/approve", "/v1/connect/requests/:userCode/approve"},
	{"POST", "/v1/link/requests/:userCode/deny", "/v1/connect/requests/:userCode/deny"},
	{"POST", "/v1/link/authorizations", "/v1/connect/authorizations"},
	{"GET", "/v1/link/authorizations/:requestCode", "/v1/connect/authorizations/:requestCode"},
	{"POST", "/v1/link/authorizations/:requestCode/approve", "/v1/connect/authorizations/:requestCode/approve"},
	{"POST", "/v1/link/authorizations/:requestCode/deny", "/v1/connect/authorizations/:requestCode/deny"},
	{"POST", "/v1/link/token", "/v1/connect/token"},
	{"POST", "/v1/link/refresh", "/v1/connect/refresh"},
	{"GET", "/v1/instances", "/v1/connected-apps"},
	{"GET", "/v1/instances/me", "/v1/connected-apps/me"},
	{"PUT", "/v1/instances/me", "/v1/connected-apps/me"},
	{"DELETE", "/v1/instances/:id", "/v1/connected-apps/:id"},
	{"POST", "/v1/deliveries/collect", "/v1/sends/collect"},
	{"DELETE", "/v1/deliveries/:id", "/v1/sends/:id"},
	{"GET", "/delivery/:id/export", "/send/:id/export"},
	{"GET", "/v1/assets/:id/update-destinations", "/v1/works/:id/integrations"},
	{"PUT", "/v1/assets/:id/update-destinations", "/v1/works/:id/integrations"},
	{"GET", "/v1/works/:id/update-destinations", "/v1/works/:id/integrations"},
	{"PUT", "/v1/works/:id/update-destinations", "/v1/works/:id/integrations"},
	{"GET", "/v1/assets/:id/announcements", "/v1/works/:id/announcement-attempts"},
	{"GET", "/v1/works/:id/announcements", "/v1/works/:id/announcement-attempts"},
	{"GET", "/v1/account/update-destinations", "/v1/account/integrations"},
	{"POST", "/v1/account/update-destinations", "/v1/account/integrations"},
	{"GET", "/v1/account/update-destinations/:id", "/v1/account/integrations/:id"},
	{"PATCH", "/v1/account/update-destinations/:id", "/v1/account/integrations/:id"},
	{"DELETE", "/v1/account/update-destinations/:id", "/v1/account/integrations/:id"},
	{"POST", "/v1/account/update-destinations/:id/verification", "/v1/account/integrations/:id/verification"},
	{"DELETE", "/v1/account/update-destinations/:id/verification", "/v1/account/integrations/:id/verification"},
	{"POST", "/v1/account/update-destinations/:id/secret", "/v1/account/integrations/:id/secret"},
	{"GET", "/v1/publication/destinations", "/v1/blog/integrations"},
	{"POST", "/v1/publication/destinations", "/v1/blog/integrations"},
	{"PATCH", "/v1/publication/destinations/:id", "/v1/blog/integrations/:id"},
	{"DELETE", "/v1/publication/destinations/:id", "/v1/blog/integrations/:id"},
	{"POST", "/v1/publication/destinations/:id/verification", "/v1/blog/integrations/:id/verification"},
	{"DELETE", "/v1/publication/destinations/:id/verification", "/v1/blog/integrations/:id/verification"},
	{"POST", "/v1/publication/destinations/:id/secret", "/v1/blog/integrations/:id/secret"},
	{"POST", "/v1/publication/channels", "/v1/blog/channels"},
	{"PATCH", "/v1/publication/channels/:id", "/v1/blog/channels/:id"},
	{"GET", "/v1/publication/deliveries", "/v1/blog/announcement-attempts"},
	{"GET", "/v1/publication/deliveries/:id/attempts", "/v1/blog/announcement-attempts/:id/tries"},
	{"POST", "/v1/publication/deliveries/:id/replay", "/v1/blog/announcement-attempts/:id/replay"},
	{"POST", "/v1/publication/deliveries/:id/repair", "/v1/blog/announcement-attempts/:id/repair"},
	{"PUT", "/v1/publication/grants/:id/destinations", "/v1/blog/grants/:id/integrations"},
	{"GET", "/v1/publication/posts/:id/destinations", "/v1/blog/posts/:id/integrations"},
	{"GET", "/v1/publication/posts/:id/deliveries", "/v1/blog/posts/:id/announcement-attempts"},
	{"PUT", "/v1/assets/:id/withhold", "/v1/works/:id/withhold"},
	{"DELETE", "/v1/assets/:id/withhold", "/v1/works/:id/withhold"},
	{"PUT", "/v1/account/nsfw-visibility", "/v1/account/nsfw-preference"},
}

func TestEveryPathFromBeforeTheRenameReachesTheHandlerItsNewNameReaches(t *testing.T) {
	t.Parallel()
	served := handlerNames(t)

	for _, renamed := range renamedPaths {
		t.Run(renamed.method+" "+renamed.was, func(t *testing.T) {
			t.Parallel()
			was, answered := served[renamed.method+" "+renamed.was]
			if !answered {
				t.Fatalf("%s %s is not served any more", renamed.method, renamed.was)
			}
			now, answered := served[renamed.method+" "+renamed.now]
			if !answered {
				t.Fatalf("%s %s is not served", renamed.method, renamed.now)
			}
			if was != now {
				t.Fatalf("%s reaches %s and %s reaches %s", renamed.was, was, renamed.now, now)
			}
		})
	}
}

func handlerNames(t *testing.T) map[string]string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	err := registerRoutes(
		r, services{Works: &work.Service{}, Apps: &connect.Apps{}}, api.DefaultDeadlines(), apitest.Ready)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	named := make(map[string]string, len(r.Routes()))
	for _, route := range r.Routes() {
		named[route.Method+" "+route.Path] = route.Handler
	}
	return named
}
