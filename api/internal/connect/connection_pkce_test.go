package connect_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

type startedAuthorization struct {
	AuthorizationURL string    `json:"authorizationUrl"`
	UserCode         string    `json:"userCode"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type pendingAuthorization struct {
	AppName         string    `json:"appName"`
	Name            string    `json:"name"`
	AppVersion      *string   `json:"appVersion"`
	ProtocolVersion int       `json:"protocolVersion"`
	Capabilities    []string  `json:"capabilities"`
	AcceptedFormats []string  `json:"acceptedFormats"`
	Permissions     []string  `json:"permissions"`
	ExpiresAt       time.Time `json:"expiresAt"`
	ApprovalToken   string    `json:"approvalToken"`
}

// startAuthorization starts a browser authorization and returns its request code from the link and the code the app shows
func startAuthorization(t *testing.T, r http.Handler, name string) (string, string) {
	t.Helper()
	digest := sha256.Sum256([]byte(strings.Repeat("A", 43)))
	body := apitest.ConnectionStartBody("Example browser client", name, []string{"work:receive"})
	body["state"] = strings.Repeat("s", 43)
	body["redirectUri"] = "http://127.0.0.1:49152/illarin/callback"
	body["codeChallenge"] = base64.RawURLEncoding.EncodeToString(digest[:])
	body["codeChallengeMethod"] = "S256"
	rec := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/authorizations", apitest.JSONText(t, body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("start authorization status = %d, want 201. body: %s", rec.Code, rec.Body.String())
	}
	started := apitest.DecodeResponse[startedAuthorization](t, rec)
	link, err := url.Parse(started.AuthorizationURL)
	if err != nil || link.Query().Get("request") == "" || started.UserCode == "" {
		t.Fatalf("started authorization = %+v", started)
	}
	return link.Query().Get("request"), started.UserCode
}

func reviewAuthorization(t *testing.T, r http.Handler, session *http.Cookie, requestCode, userCode string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet,
		"/v1/connect/authorizations/"+requestCode+"?userCode="+url.QueryEscape(userCode), nil)
	return apitest.Send(t, r, apitest.Authorized(req, session))
}

func TestLoopbackPKCEReviewApprovalAndOneUseExchange(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	verifier := strings.Repeat("A", 43)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	state := strings.Repeat("s", 43)
	redirectURI := "http://127.0.0.1:49152/illarin/callback"

	body := apitest.ConnectionStartBody("Example client", "studio workstation", []string{"work:receive"})
	body["state"] = state
	body["redirectUri"] = redirectURI
	body["codeChallenge"] = challenge
	body["codeChallengeMethod"] = "S256"
	startedRec := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/authorizations", apitest.JSONText(t, body))
	apitest.AssertNoStore(t, startedRec)
	if startedRec.Code != http.StatusCreated {
		t.Fatalf("start authorization status = %d, want 201. body: %s", startedRec.Code, startedRec.Body.String())
	}
	started := apitest.DecodeResponse[startedAuthorization](t, startedRec)
	if started.ExpiresAt.Before(time.Now()) {
		t.Errorf("authorization already expired at %s", started.ExpiresAt)
	}
	authorizationURL, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	requestCode := authorizationURL.Query().Get("request")
	if requestCode == "" || authorizationURL.Scheme+"://"+authorizationURL.Host+authorizationURL.Path != apitest.BrowserOrigin+"/connect" {
		t.Fatalf("authorization URL = %q", started.AuthorizationURL)
	}

	if started.UserCode == "" || strings.Contains(started.AuthorizationURL, started.UserCode) {
		t.Fatalf("authorization code %q is missing or in the link %q", started.UserCode, started.AuthorizationURL)
	}

	reviewRec := reviewAuthorization(t, r, session, requestCode, started.UserCode)
	apitest.AssertNoStore(t, reviewRec)
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("review authorization status = %d, want 200. body: %s", reviewRec.Code, reviewRec.Body.String())
	}
	pending := apitest.DecodeResponse[pendingAuthorization](t, reviewRec)
	if pending.AppName != "Example client" || pending.Name != "studio workstation" {
		t.Errorf("pending authorization identity = %q / %q", pending.AppName, pending.Name)
	}
	if !slices.Equal(pending.Capabilities, []string{"example.client:work-install"}) ||
		!slices.Equal(pending.AcceptedFormats, []string{"portable-card-v1"}) ||
		!slices.Equal(pending.Permissions, []string{"work:receive"}) {
		t.Errorf("pending authorization capabilities = %+v", pending)
	}

	approveTarget := "/v1/connect/authorizations/" + requestCode + "/approve"
	csrfCases := []struct {
		name   string
		origin string
		header string
	}{
		{name: "neither"},
		{name: "origin only", origin: apitest.BrowserOrigin},
		{name: "header only", header: "1"},
		{name: "wrong origin", origin: "https://attacker.invalid", header: "1"},
	}
	for _, test := range csrfCases {
		req := httptest.NewRequest(http.MethodPost, approveTarget, nil)
		req.Header.Set("Origin", test.origin)
		req.Header.Set(api.BrowserHeader, test.header)
		req.AddCookie(session)
		rec := apitest.Send(t, r, req)
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusForbidden {
			t.Errorf("approve with %s = %d, want 403", test.name, rec.Code)
		}
	}

	approved := apitest.Send(t, r, apitest.BrowserRequest(t, http.MethodPost, approveTarget,
		map[string]string{"approvalToken": pending.ApprovalToken}, session))
	apitest.AssertNoStore(t, approved)
	if approved.Code != http.StatusOK {
		t.Fatalf("approve authorization status = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}
	var redirect struct {
		URL string `json:"redirectUrl"`
	}
	if err := json.Unmarshal(approved.Body.Bytes(), &redirect); err != nil {
		t.Fatalf("decode redirect: %v", err)
	}
	callback, err := url.Parse(redirect.URL)
	if err != nil {
		t.Fatalf("parse callback: %v", err)
	}
	if callback.Scheme+"://"+callback.Host+callback.Path != redirectURI {
		t.Errorf("redirect base = %q, want exact callback %q", callback.Scheme+"://"+callback.Host+callback.Path, redirectURI)
	}
	if callback.Query().Get("state") != state {
		t.Errorf("redirect state = %q, want %q", callback.Query().Get("state"), state)
	}
	authorizationCode := callback.Query().Get("code")
	if authorizationCode == "" || len(callback.Query()) != 2 {
		t.Fatalf("redirect query = %v, want only code and state", callback.Query())
	}

	exchange := func(code, candidateVerifier, candidateRedirect string) *httptest.ResponseRecorder {
		t.Helper()
		return apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/token", apitest.JSONText(t, map[string]string{
			"authorizationCode": code,
			"codeVerifier":      candidateVerifier,
			"redirectUri":       candidateRedirect,
		}))
	}
	wrongRedirect := exchange(authorizationCode, verifier,
		"http://127.0.0.1:49153/illarin/callback")
	apitest.AssertNoStore(t, wrongRedirect)
	if wrongRedirect.Code != http.StatusBadRequest {
		t.Errorf("exchange with another loopback callback = %d, want 400", wrongRedirect.Code)
	}
	wrongVerifier := exchange(authorizationCode, strings.Repeat("B", 43), redirectURI)
	apitest.AssertNoStore(t, wrongVerifier)
	if wrongVerifier.Code != http.StatusBadRequest {
		t.Errorf("exchange with wrong verifier = %d, want 400", wrongVerifier.Code)
	}

	exchanged := exchange(authorizationCode, verifier, redirectURI)
	apitest.AssertNoStore(t, exchanged)
	if exchanged.Code != http.StatusOK {
		t.Fatalf("exact code exchange status = %d, want 200. body: %s", exchanged.Code, exchanged.Body.String())
	}
	credentials := apitest.DecodeResponse[apitest.AppCredentials](t, exchanged)
	if credentials.AccessToken == "" || credentials.RefreshToken == "" || credentials.ConnectedApp.ID == "" {
		t.Fatalf("code exchange credentials = %+v", credentials)
	}
	if credentials.ConnectedApp.AppName != "Example client" || credentials.ConnectedApp.Name != "studio workstation" {
		t.Errorf("linked connected app = %+v", credentials.ConnectedApp)
	}

	secondExchange := exchange(authorizationCode, verifier, redirectURI)
	apitest.AssertNoStore(t, secondExchange)
	if secondExchange.Code != http.StatusBadRequest {
		t.Errorf("second exchange status = %d, want 400", secondExchange.Code)
	}
}

func TestBrowserAuthorizationReviewsAreReadOnlyAndTheFirstDecisionBindsTheUser(t *testing.T) {
	t.Parallel()
	r, firstSession, pool := harness.NewConnectRouter(t)
	secondSession := apitest.AddVerifiedUser(
		t, r, pool, "second.browser@example.com", "second.browser",
	)
	requestCode, userCode := startAuthorization(t, r, "review test")

	tokens := map[*http.Cookie]string{}
	for index, session := range []*http.Cookie{firstSession, firstSession, secondSession} {
		rec := reviewAuthorization(t, r, session, requestCode, userCode)
		if rec.Code != http.StatusOK {
			t.Fatalf("review %d status = %d, want 200. body: %s", index+1, rec.Code, rec.Body.String())
		}
		tokens[session] = apitest.DecodeResponse[pendingAuthorization](t, rec).ApprovalToken
	}

	var reviewerMissing bool
	if err := pool.QueryRow(context.Background(), `
		select reviewed_by is null
		  from connection_authorizations
		 where app_name = 'Example browser client' and name = 'review test'
	`).Scan(&reviewerMissing); err != nil {
		t.Fatalf("read pending browser review state: %v", err)
	}
	if !reviewerMissing {
		t.Fatal("review GET claimed the browser authorization")
	}

	denyTarget := "/v1/connect/authorizations/" + requestCode + "/deny"
	denied := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, denyTarget, map[string]string{"approvalToken": tokens[firstSession]}, firstSession,
	))
	if denied.Code != http.StatusOK {
		t.Fatalf("first decision status = %d, want 200. body: %s", denied.Code, denied.Body.String())
	}
	var redirect struct {
		URL string `json:"redirectUrl"`
	}
	if err := json.Unmarshal(denied.Body.Bytes(), &redirect); err != nil {
		t.Fatalf("decode denial redirect: %v", err)
	}
	callback, err := url.Parse(redirect.URL)
	if err != nil || callback.Query().Get("error") != "access_denied" {
		t.Fatalf("denial redirect = %q, want access_denied", redirect.URL)
	}

	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/authorizations/"+requestCode+"/approve",
		map[string]string{"approvalToken": tokens[secondSession]}, secondSession,
	))
	if approved.Code != http.StatusNotFound {
		t.Fatalf("second user's decision status = %d, want 404. body: %s", approved.Code, approved.Body.String())
	}
}

func TestBrowserApprovalNeedsTheAppsCodeEnteredForThatRequest(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	other := apitest.AddVerifiedUser(t, r, pool, "other.browser@example.com", "other.browser")
	requestCode, userCode := startAuthorization(t, r, "first request")
	otherRequest, otherCode := startAuthorization(t, r, "second request")

	for _, test := range []struct{ name, code string }{
		{"no code", ""},
		{"another request's code", otherCode},
	} {
		if rec := reviewAuthorization(t, r, session, requestCode, test.code); rec.Code != http.StatusNotFound {
			t.Errorf("review with %s = %d, want 404. body: %s", test.name, rec.Code, rec.Body.String())
		}
	}
	reviewed := reviewAuthorization(t, r, session, otherRequest, otherCode)
	if reviewed.Code != http.StatusOK {
		t.Fatalf("review the second request = %d. body: %s", reviewed.Code, reviewed.Body.String())
	}
	otherToken := apitest.DecodeResponse[pendingAuthorization](t, reviewed).ApprovalToken
	reviewed = reviewAuthorization(t, r, other, requestCode, userCode)
	if reviewed.Code != http.StatusOK {
		t.Fatalf("review by another account = %d. body: %s", reviewed.Code, reviewed.Body.String())
	}
	otherAccountToken := apitest.DecodeResponse[pendingAuthorization](t, reviewed).ApprovalToken

	for _, test := range []struct{ name, path, token string }{
		{"no proof", "/v1/connect/authorizations/" + requestCode + "/approve", ""},
		{"another request's proof", "/v1/connect/authorizations/" + requestCode + "/approve", otherToken},
		{"another account's proof", "/v1/connect/authorizations/" + requestCode + "/approve", otherAccountToken},
		{"no proof on the old path", "/v1/link/authorizations/" + requestCode + "/approve", ""},
		{"no proof to deny", "/v1/connect/authorizations/" + requestCode + "/deny", ""},
		{"no proof to deny on the old path", "/v1/link/authorizations/" + requestCode + "/deny", ""},
	} {
		rec := apitest.Send(t, r, apitest.BrowserRequest(
			t, http.MethodPost, test.path, map[string]string{"approvalToken": test.token}, session,
		))
		if rec.Code != http.StatusNotFound {
			t.Errorf("decision with %s = %d, want 404. body: %s", test.name, rec.Code, rec.Body.String())
		}
	}

	var undecided bool
	if err := pool.QueryRow(context.Background(), `
		select approved_at is null and denied_at is null and reviewed_by is null
		  from connection_authorizations
		 where name = 'first request'
	`).Scan(&undecided); err != nil {
		t.Fatalf("read the first request: %v", err)
	}
	if !undecided {
		t.Fatal("a decision without code-entry proof changed the request")
	}
}

func TestSameDeviceConnectingRejectsNonLoopbackRedirects(t *testing.T) {
	t.Parallel()
	r, _, _ := harness.NewConnectRouter(t)
	verifier := strings.Repeat("A", 43)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])

	for _, redirectURI := range []string{
		"http://localhost:49152/illarin/callback",
		"https://client.example/callback",
		"http://127.0.0.1/illarin/callback",
		"http://127.0.0.1:49152/illarin/callback?next=/admin",
	} {
		body := apitest.ConnectionStartBody("Example client", "studio workstation", []string{"work:receive"})
		body["state"] = strings.Repeat("s", 43)
		body["redirectUri"] = redirectURI
		body["codeChallenge"] = challenge
		body["codeChallengeMethod"] = "S256"
		rec := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/authorizations", apitest.JSONText(t, body))
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("redirect %q status = %d, want 400. body: %s", redirectURI, rec.Code, rec.Body.String())
		}
	}
}
