package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestNoRouteForOutsideBlogAccessRemains(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	grant := stack.contributor(t, "writer@example.com", "writer.dev").grant
	illarin := stack.appBySlug(t, "illarin")

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/publication/apps", ""},
		{http.MethodPost, "/v1/publication/apps", `{"slug":"x","name":"X","home":"https://x.example"}`},
		{http.MethodPut, "/v1/publication/apps", `{"ids":[]}`},
		{http.MethodPatch, "/v1/publication/apps/" + illarin.ID, `{"retired":true}`},
		{http.MethodPut, "/v1/publication/apps/" + illarin.ID + "/mark", ""},
		{http.MethodPut, "/v1/publication/apps/" + illarin.ID + "/destinations", `{"destinationIds":[]}`},
		{http.MethodGet, "/v1/publication/grants/" + grant.ID + "/tokens", ""},
		{http.MethodPost, "/v1/publication/grants/" + grant.ID + "/tokens", `{"name":"Robot"}`},
		{http.MethodDelete, "/v1/publication/tokens/" + grant.ID, ""},
		{http.MethodGet, "/v1/publication/token", ""},
	} {
		request := httptest.NewRequest(route.method, route.path, nil)
		if route.body != "" {
			request = jsonRequest(t, route.method, route.path, route.body)
		}
		response := apitest.Send(t, stack.router, apitest.Authorized(request, stack.authority))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", route.method, route.path, response.Code)
		}
	}
}

func TestABearerValueReachesNoPost(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/publication/posts", nil)
	request.Header.Set("Authorization", "Bearer ip1.BCDFGHJK.abcdefghijklmnopqrstuvwxyz0123456789")

	response := apitest.Send(t, stack.router, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("a bearer value answered %d: %s", response.Code, response.Body.String())
	}
}
