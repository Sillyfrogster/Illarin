package account_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestCredentialCookiesCarryTheProductName(t *testing.T) {
	t.Parallel()
	cookies := []struct {
		role string
		got  string
		want string
	}{
		{"session", api.SessionCookie, "illarin_session"},
		{"oauth state", account.OAuthStateCookieName, "illarin_discord_state"},
		{"oauth return", account.OAuthReturnCookieName, "illarin_discord_return"},
	}
	for _, cookie := range cookies {
		if cookie.got != cookie.want {
			t.Errorf("%s cookie = %q, want %q", cookie.role, cookie.got, cookie.want)
		}
	}
}

func TestSignOutReadsTheMutationHeaderAndClearsTheSessionCookie(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)
	if session.Name != "illarin_session" {
		t.Fatalf("sign-up set cookie %q, want illarin_session", session.Name)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-out", nil)
	req.AddCookie(session)
	req.Header.Set("Origin", apitest.BrowserOrigin)
	req.Header.Set("X-Illarin-Request", "1")
	rec := apitest.Send(t, router, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("sign-out = %d %s, want %d", rec.Code, rec.Body.String(), http.StatusNoContent)
	}
	cleared := rec.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != "illarin_session" || cleared[0].MaxAge >= 0 {
		t.Errorf("sign-out cookies = %+v, want illarin_session cleared", cleared)
	}
}
