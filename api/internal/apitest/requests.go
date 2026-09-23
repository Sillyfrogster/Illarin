package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

func Send(t *testing.T, r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	WithReviewedVersion(t, r, req)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func SendJSON(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return Send(t, handler, req)
}

func Authorized(req *http.Request, session *http.Cookie) *http.Request {
	req.AddCookie(session)
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
		req.Header.Set("Origin", BrowserOrigin)
		req.Header.Set(api.BrowserHeader, "1")
	}
	return req
}

func AuthorizedJSONRequest(
	t *testing.T,
	method string,
	path string,
	body string,
	session *http.Cookie,
) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return Authorized(req, session)
}

const BrowserOrigin = "http://localhost:3000"

func BrowserRequest(
	t *testing.T,
	method string,
	target string,
	body any,
	session *http.Cookie,
) *http.Request {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(JSONText(t, body)))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", BrowserOrigin)
	req.Header.Set(api.BrowserHeader, "1")
	return Authorized(req, session)
}

func BrowserMutation(request *http.Request) *http.Request {
	request.Header.Set("Origin", BrowserOrigin)
	request.Header.Set(api.BrowserHeader, "1")
	return request
}

func AsApp(t *testing.T, method, target, token string, body any) *http.Request {
	t.Helper()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(JSONText(t, body)))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func DecodeResponse[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response: %v. body: %s", err, rec.Body.String())
	}
	return value
}

func JSONText(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
	return string(encoded)
}

func AssertNoStore(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if cache := rec.Header().Get("Cache-Control"); !strings.Contains(cache, "no-store") {
		t.Errorf("Cache-Control = %q, want no-store", cache)
	}
}

func CompactJSON(t *testing.T, raw json.RawMessage) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		t.Fatalf("compact %s: %v", raw, err)
	}
	return out.Bytes()
}
