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
	{"GET", "/v1/assets/:id/updates/protection", "/v1/works/:id/versions/protection"},
	{"PUT", "/v1/assets/:id/updates/:number/protection", "/v1/works/:id/versions/:number/protection"},
	{"POST", "/v1/assets/:id/updates/:number/restore", "/v1/works/:id/versions/:number/restore"},
	{"POST", "/v1/assets/:id/updates/:number/withdraw", "/v1/works/:id/versions/:number/withdraw"},
	{"PATCH", "/v1/assets/:id/updates/:number/notes", "/v1/works/:id/versions/:number/notes"},
	{"GET", "/v1/assets/:id/updates/:number/downloads", "/v1/works/:id/versions/:number/downloads"},
	{"GET", "/v1/works/:id/updates", "/v1/works/:id/versions"},
	{"POST", "/v1/works/:id/updates", "/v1/works/:id/versions"},
	{"GET", "/v1/works/:id/updates/comparison", "/v1/works/:id/versions/comparison"},
	{"GET", "/v1/works/:id/updates/protection", "/v1/works/:id/versions/protection"},
	{"PUT", "/v1/works/:id/updates/:number/protection", "/v1/works/:id/versions/:number/protection"},
	{"POST", "/v1/works/:id/updates/:number/restore", "/v1/works/:id/versions/:number/restore"},
	{"POST", "/v1/works/:id/updates/:number/withdraw", "/v1/works/:id/versions/:number/withdraw"},
	{"PATCH", "/v1/works/:id/updates/:number/notes", "/v1/works/:id/versions/:number/notes"},
	{"GET", "/v1/works/:id/updates/:number/downloads", "/v1/works/:id/versions/:number/downloads"},
	{"GET", "/v1/assets/:id/sealed", "/v1/works/:id/sealed"},
	{"GET", "/v1/assets/:id/instances", "/v1/works/:id/instances"},
	{"POST", "/v1/assets/:id/deliveries", "/v1/works/:id/deliveries"},
	{"GET", "/v1/assets/:id/update-destinations", "/v1/works/:id/update-destinations"},
	{"PUT", "/v1/assets/:id/update-destinations", "/v1/works/:id/update-destinations"},
	{"GET", "/v1/assets/:id/announcements", "/v1/works/:id/announcements"},
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
		r, services{Works: &work.Service{}, Links: &connect.Apps{}}, api.DefaultDeadlines(), apitest.Ready)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	named := make(map[string]string, len(r.Routes()))
	for _, route := range r.Routes() {
		named[route.Method+" "+route.Path] = route.Handler
	}
	return named
}
