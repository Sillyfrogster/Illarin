package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func serveWithSession(t *testing.T, lookup Lookup, withCookie bool, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.GET("/", Sessions(lookup), handler)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if withCookie {
		request.AddCookie(&http.Cookie{Name: SessionCookie, Value: "token"})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func refusal(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", recorder.Body.String(), err)
	}
	return body.Error
}

func TestASessionIsLookedUpOnlyOncePerRequest(t *testing.T) {
	t.Parallel()
	lookups := 0
	lookup := func(context.Context, string) (*Account, error) {
		lookups++
		return &Account{Handle: "reader"}, nil
	}

	serveWithSession(t, lookup, true, func(c *gin.Context) {
		first, _ := Current(c)
		second, _ := Current(c)
		if first == nil || second == nil || first.Handle != "reader" {
			t.Errorf("current = %+v then %+v, want the reader both times", first, second)
		}
	})

	if lookups != 1 {
		t.Fatalf("looked the session up %d times, want once", lookups)
	}
}

func TestARequestWithoutASessionCookieIsNeverLookedUp(t *testing.T) {
	t.Parallel()
	lookup := func(context.Context, string) (*Account, error) {
		t.Error("looked up a session the request did not carry")
		return nil, nil
	}

	recorder := serveWithSession(t, lookup, false, func(c *gin.Context) {
		SignedIn(c, "following a work")
	})

	if recorder.Code != http.StatusUnauthorized || refusal(t, recorder) != "Sign in before following a work." {
		t.Fatalf("signed out = %d %s, want 401 naming the action", recorder.Code, recorder.Body.String())
	}
}

func TestVerifiedAndAdminRefuseWithTheActionNamed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		account Account
		check   func(*gin.Context, string) (Account, bool)
		action  string
		status  int
		message string
	}{
		{"unverified", Account{}, Verified, "uploading", http.StatusForbidden, "Verify your email before uploading."},
		{"verified", Account{EmailVerified: true}, Verified, "uploading", http.StatusOK, ""},
		{"not an admin", Account{EmailVerified: true, Role: RoleUser}, Admin, "restrict a profile", http.StatusForbidden, "Only an admin can restrict a profile."},
		{"admin", Account{EmailVerified: true, Role: RoleAdmin}, Admin, "restrict a profile", http.StatusOK, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			lookup := func(context.Context, string) (*Account, error) { return &test.account, nil }
			recorder := serveWithSession(t, lookup, true, func(c *gin.Context) {
				if _, ok := test.check(c, test.action); ok {
					c.Status(http.StatusOK)
				}
			})
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.status, recorder.Body.String())
			}
			if test.message != "" && refusal(t, recorder) != test.message {
				t.Fatalf("refusal = %q, want %q", refusal(t, recorder), test.message)
			}
		})
	}
}

func TestADeadlineThatIsNotSetIsRefused(t *testing.T) {
	t.Parallel()
	if err := DefaultDeadlines().Check(); err != nil {
		t.Fatalf("default deadlines refused: %v", err)
	}
	missing := DefaultDeadlines()
	missing.Verify = 0
	if err := missing.Check(); err == nil {
		t.Fatal("a route could run with no Verify deadline")
	}
	past := DefaultDeadlines()
	past.JSON = -time.Second
	if err := past.Check(); err == nil {
		t.Fatal("a negative deadline was accepted")
	}
}
