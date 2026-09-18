package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestHandlersRefuseMalformedInputBeforeDoingAnything(t *testing.T) {
	t.Parallel()
	router := harness.NewRouter(t)
	const someID = "7f1c1c52-5f9e-4bb1-9d0c-2f4c6a0f6f11"

	cases := []struct {
		name    string
		method  string
		target  string
		headers map[string]string
		status  int
		message string
	}{
		{"path id that is not a uuid", http.MethodGet, "/v1/assets/not-a-uuid", nil,
			http.StatusBadRequest, "Invalid format for parameter id: "},
		{"second path id that is not a uuid", http.MethodDelete, "/v1/assets/" + someID + "/revisions/nope", nil,
			http.StatusBadRequest, "Invalid format for parameter operationId: "},
		{"path number that is not a number", http.MethodPost, "/v1/assets/" + someID + "/updates/two/withdraw", nil,
			http.StatusBadRequest, "Invalid format for parameter number: "},
		{"query number that is not a number", http.MethodGet, "/v1/assets?limit=many", nil,
			http.StatusBadRequest, "Invalid format for parameter limit: "},
		{"query time that is not a time", http.MethodGet, "/v1/notifications?before=yesterday", nil,
			http.StatusBadRequest, "Invalid format for parameter before: "},
		{"query value given twice", http.MethodGet, "/v1/assets?kind=character&kind=preset", nil,
			http.StatusBadRequest, "Invalid format for parameter kind: "},
		{"missing required query value", http.MethodGet, "/v1/auth/discord/callback", nil,
			http.StatusBadRequest, "Query argument state is required, but not found"},
		{"missing working copy version", http.MethodPost, "/v1/assets/" + someID + "/blocks", nil,
			http.StatusBadRequest, "Header parameter X-Working-Copy-Version is required, but not found"},
		{"working copy version that is not a number", http.MethodPost, "/v1/assets/" + someID + "/blocks",
			map[string]string{"X-Working-Copy-Version": "latest"},
			http.StatusBadRequest, "Invalid format for parameter X-Working-Copy-Version: "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(tc.method, tc.target, nil)
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d. body: %s", rec.Code, tc.status, rec.Body.String())
			}
			var body struct {
				Msg string `json:"msg"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body %q: %v", rec.Body.String(), err)
			}
			if !strings.HasPrefix(body.Msg, tc.message) {
				t.Errorf("msg = %q, want it to start with %q", body.Msg, tc.message)
			}
		})
	}
}

func TestAppActionsWithoutTheIllarinHeaderAreRefused(t *testing.T) {
	t.Parallel()
	router := harness.NewRouter(t)
	for _, target := range []string{
		"/v1/link/requests/ABCD-EFGH/approve",
		"/v1/link/authorizations/some-code/deny",
	} {
		rec := apitest.Send(t, router, httptest.NewRequest(http.MethodPost, target, nil))
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Open this action from Illarin") {
			t.Errorf("%s = %d %s, want 403 asking to open it from Illarin", target, rec.Code, rec.Body.String())
		}
	}
}
