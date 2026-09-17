package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCredentialCookiesCarryTheProductName(t *testing.T) {
	t.Parallel()
	cookies := []struct {
		role string
		got  string
		want string
	}{
		{"session", sessionCookieName, "illarin_session"},
		{"oauth state", oauthStateCookieName, "illarin_discord_state"},
		{"oauth return", oauthReturnCookieName, "illarin_discord_return"},
	}
	for _, cookie := range cookies {
		if cookie.got != cookie.want {
			t.Errorf("%s cookie = %q, want %q", cookie.role, cookie.got, cookie.want)
		}
	}
}

func TestSignOutReadsTheMutationHeaderAndClearsTheSessionCookie(t *testing.T) {
	t.Parallel()
	router, session := newVerifiedTestRouter(t)
	if session.Name != "illarin_session" {
		t.Fatalf("sign-up set cookie %q, want illarin_session", session.Name)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-out", nil)
	req.AddCookie(session)
	req.Header.Set("Origin", testBrowserOrigin)
	req.Header.Set("X-Illarin-Request", "1")
	rec := send(t, router, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("sign-out = %d %s, want %d", rec.Code, rec.Body.String(), http.StatusNoContent)
	}
	cleared := rec.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != "illarin_session" || cleared[0].MaxAge >= 0 {
		t.Errorf("sign-out cookies = %+v, want illarin_session cleared", cleared)
	}
}
