package blog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestNoRouteForAppsGrantsOrTokensRemains(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	id := "00000000-0000-0000-0000-000000000001"

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/publication/apps", ""},
		{http.MethodGet, "/v1/blog/apps", ""},
		{http.MethodPost, "/v1/blog/apps", `{"slug":"x","name":"X","home":"https://x.example"}`},
		{http.MethodGet, "/v1/post-apps", ""},
		{http.MethodGet, "/v1/publication/grants", ""},
		{http.MethodPost, "/v1/publication/grants", `{"handle":"x"}`},
		{http.MethodGet, "/v1/blog/grants", ""},
		{http.MethodPut, "/v1/blog/grants/" + id + "/integrations", `{"integrationIds":[]}`},
		{http.MethodPut, "/v1/publication/grants/" + id + "/destinations", `{"destinationIds":[]}`},
		{http.MethodGet, "/v1/blog/grants/" + id + "/tokens", ""},
		{http.MethodDelete, "/v1/blog/tokens/" + id, ""},
		{http.MethodGet, "/v1/blog/token", ""},
	} {
		request := httptest.NewRequest(route.method, route.path, nil)
		if route.body != "" {
			request = jsonRequest(t, route.method, route.path, route.body)
		}
		response := apitest.Send(t, stack.router, apitest.Authorized(request, stack.admin))
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", route.method, route.path, response.Code)
		}
	}
}

func TestABearerValueReachesNoPost(t *testing.T) {
	t.Parallel()
	stack := newBlogStack(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/blog/posts", nil)
	request.Header.Set("Authorization", "Bearer ip1.BCDFGHJK.abcdefghijklmnopqrstuvwxyz0123456789")

	response := apitest.Send(t, stack.router, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("a bearer value answered %d: %s", response.Code, response.Body.String())
	}
}
