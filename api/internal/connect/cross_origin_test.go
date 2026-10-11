package connect_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAppRoutesAnswerCrossOriginRequestsWithoutCookieCredentials(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{apitest.LibrarySyncPermission})
	const origin = "http://192.168.1.2:8000"

	preflight := httptest.NewRequest(http.MethodOptions, "/v1/library/sync", nil)
	preflight.Header.Set("Origin", origin)
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set("Access-Control-Request-Headers", "authorization, content-type")
	rec := apitest.Send(t, router, preflight)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight = %d, want 204", rec.Code)
	}
	assertAppCORS(t, rec)
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), http.MethodPost) {
		t.Error("preflight does not allow POST")
	}
	allowed := strings.ToLower(rec.Header().Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allowed, "authorization") || !strings.Contains(allowed, "content-type") {
		t.Errorf("allowed headers = %q", allowed)
	}

	request := apitest.AsApp(t, http.MethodPost, "/v1/library/sync", credentials.AccessToken,
		map[string]any{"snapshot": false, "entries": []any{}})
	request.Header.Set("Origin", origin)
	rec = apitest.Send(t, router, request)
	if rec.Code != http.StatusOK {
		t.Fatalf("cross-origin library sync = %d: %s", rec.Code, rec.Body.String())
	}
	assertAppCORS(t, rec)
	if result := apitest.DecodeResponse[apitest.LibraryResult](t, rec); result.Accepted != 0 {
		t.Errorf("empty library report accepted %d works", result.Accepted)
	}

	cookieOnly := httptest.NewRequest(http.MethodGet, "/v1/connected-apps/me", nil)
	cookieOnly.Header.Set("Origin", origin)
	cookieOnly.AddCookie(session)
	rec = apitest.Send(t, router, cookieOnly)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie instead of app credential = %d, want 401", rec.Code)
	}
	assertAppCORS(t, rec)

	signOut := apitest.BrowserRequest(t, http.MethodPost, "/v1/auth/sign-out", nil, session)
	signOut.Header.Set("Origin", origin)
	rec = apitest.Send(t, router, signOut)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin sign-out = %d, want 403", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("cookie route permits CORS: %v", rec.Header())
	}
}

func TestOnlyAppEndpointsAnswerCrossOriginPreflights(t *testing.T) {
	t.Parallel()
	router := harness.NewRouter(t)
	for _, endpoint := range []struct {
		path    string
		methods []string
	}{
		{"/v1/connect/requests", []string{"POST"}},
		{"/v1/connect/poll", []string{"POST"}},
		{"/v1/connect/authorizations", []string{"POST"}},
		{"/v1/connect/token", []string{"POST"}},
		{"/v1/connect/refresh", []string{"POST"}},
		{"/v1/connected-apps/me", []string{"GET", "PUT"}},
		{"/v1/sends/collect", []string{"POST"}},
		{"/v1/library/sync", []string{"POST"}},
		{"/send/00000000-0000-0000-0000-000000000001/export", []string{"GET"}},
		{"/v1/link/requests", []string{"POST"}},
		{"/v1/link/poll", []string{"POST"}},
		{"/v1/link/authorizations", []string{"POST"}},
		{"/v1/link/token", []string{"POST"}},
		{"/v1/link/refresh", []string{"POST"}},
		{"/v1/instances/me", []string{"GET", "PUT"}},
		{"/v1/deliveries/collect", []string{"POST"}},
		{"/delivery/00000000-0000-0000-0000-000000000001/export", []string{"GET"}},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			for _, method := range endpoint.methods {
				req := httptest.NewRequest(http.MethodOptions, endpoint.path, nil)
				req.Header.Set("Origin", "null")
				req.Header.Set("Access-Control-Request-Method", method)
				rec := apitest.Send(t, router, req)
				if rec.Code != http.StatusNoContent {
					t.Fatalf("preflight for %s = %d, want 204", method, rec.Code)
				}
				assertAppCORS(t, rec)
				if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), method) {
					t.Errorf("preflight does not allow %s", method)
				}
			}
		})
	}

	for _, path := range []string{
		"/v1/auth/sign-out", "/v1/connected-apps", "/v1/instances",
		"/v1/connected-apps/00000000-0000-0000-0000-000000000001/permissions",
		"/v1/connect/requests/ABCD-EFGH", "/v1/connect/requests/ABCD-EFGH/approve",
		"/v1/connect/requests/ABCD-EFGH/deny", "/v1/connect/authorizations/code/approve",
		"/v1/link/requests/ABCD-EFGH/approve", "/v1/link/authorizations/code/approve",
		"/v1/works/00000000-0000-0000-0000-000000000001/sends",
		"/v1/sends/00000000-0000-0000-0000-000000000001",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, path, nil)
			req.Header.Set("Origin", "https://elsewhere.example")
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			rec := apitest.Send(t, router, req)
			if rec.Code >= 200 && rec.Code < 300 {
				t.Errorf("cookie route preflight = %d", rec.Code)
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != "" || rec.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Errorf("cookie route permits CORS: %v", rec.Header())
			}
		})
	}
}

func assertAppCORS(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("allowed origin = %q, want *", got)
	}
	if _, sent := rec.Header()["Access-Control-Allow-Credentials"]; sent {
		t.Error("response carries Access-Control-Allow-Credentials")
	}
}
