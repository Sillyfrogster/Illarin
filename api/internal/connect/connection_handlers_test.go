package connect_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/gin-gonic/gin"
)

var testConnectHMACKey = []byte("01234567890123456789012345678901")

func listConnectedApps(t *testing.T, r *gin.Engine, session *http.Cookie) []apitest.ConnectedApp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/connected-apps", nil)
	rec := apitest.Send(t, r, apitest.Authorized(req, session))
	apitest.AssertNoStore(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("list connected apps status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Items []apitest.ConnectedApp `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode connected apps: %v", err)
	}
	return list.Items
}

func connectedAppByID(t *testing.T, apps []apitest.ConnectedApp, id string) apitest.ConnectedApp {
	t.Helper()
	for _, app := range apps {
		if app.ID == id {
			return app
		}
	}
	t.Fatalf("connected app %s is absent from %+v", id, apps)
	return apitest.ConnectedApp{}
}

func TestADeviceConnectionNeedsManualReviewAndReturnsATokenPair(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	started, raw := apitest.StartConnection(t, r,
		apitest.ConnectionStartBody("Example client", "studio workstation", []string{"work:receive"}))

	if _, present := raw["verificationUrlComplete"]; present {
		t.Error("device response includes verificationUrlComplete; the human code must be entered manually")
	}
	if started.VerificationURL != apitest.BrowserOrigin+"/connect" {
		t.Errorf("verification URL = %q", started.VerificationURL)
	}
	if len(started.UserCode) != 9 || started.UserCode[4] != '-' {
		t.Errorf("user code = %q, want eight characters split by a dash", started.UserCode)
	}
	if len(started.DeviceCode) < 40 || started.DeviceCode == started.UserCode {
		t.Errorf("private device code = %q, human code = %q", started.DeviceCode, started.UserCode)
	}

	review, pending := apitest.ReviewConnectionRequest(t, r, session, started.UserCode)
	if review.Code != http.StatusOK {
		t.Fatalf("review status = %d, want 200. body: %s", review.Code, review.Body.String())
	}
	if pending.ApprovalToken == "" {
		t.Fatal("review response has no one-use approval token")
	}
	if pending.AppName != "Example client" || pending.Name != "studio workstation" {
		t.Errorf("pending identity = %q / %q", pending.AppName, pending.Name)
	}
	if !slices.Equal(pending.Permissions, []string{"work:receive"}) {
		t.Errorf("pending permissions = %v", pending.Permissions)
	}

	withoutCSRF := httptest.NewRequest(
		http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		strings.NewReader(apitest.JSONText(t, map[string]string{"approvalToken": pending.ApprovalToken})),
	)
	withoutCSRF.Header.Set("Content-Type", "application/json")
	withoutCSRF.AddCookie(session)
	rejected := apitest.Send(t, r, withoutCSRF)
	apitest.AssertNoStore(t, rejected)
	if rejected.Code != http.StatusForbidden {
		t.Fatalf("approval without browser proof = %d, want 403", rejected.Code)
	}

	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": pending.ApprovalToken}, session,
	))
	apitest.AssertNoStore(t, approved)
	if approved.Code != http.StatusOK {
		t.Fatalf("approval status = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}

	polled := apitest.Poll(t, r, started.DeviceCode)
	if polled.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", polled.Code, polled.Body.String())
	}
	credentials := apitest.DecodeResponse[apitest.PolledConnection](t, polled)
	if credentials.Status != "connected" || credentials.AccessToken == nil || credentials.RefreshToken == nil || credentials.ConnectedApp == nil {
		t.Fatalf("poll after approval = %+v, want a token pair", credentials)
	}
	if !strings.HasPrefix(*credentials.AccessToken, "ia1.") || !strings.HasPrefix(*credentials.RefreshToken, "ir1.") {
		t.Errorf("unexpected token types: access %q refresh %q", *credentials.AccessToken, *credentials.RefreshToken)
	}
	if credentials.ConnectedApp.AppName != "Example client" || credentials.ConnectedApp.Name != "studio workstation" {
		t.Errorf("linked connected app = %+v", credentials.ConnectedApp)
	}
}

func TestDeviceReviewsAreReadOnlyAndApprovalProofsStayWithTheirUser(t *testing.T) {
	t.Parallel()
	r, firstSession, pool := harness.NewConnectRouter(t)
	secondSession := apitest.AddVerifiedUser(
		t, r, pool, "second.creator@example.com", "second.creator",
	)
	started, _ := apitest.StartConnection(t, r,
		apitest.ConnectionStartBody("Example client", "review test", []string{"work:receive"}))

	firstReview, first := apitest.ReviewConnectionRequest(t, r, firstSession, started.UserCode)
	reloadedReview, reloaded := apitest.ReviewConnectionRequest(t, r, firstSession, started.UserCode)
	if firstReview.Code != http.StatusOK || reloadedReview.Code != http.StatusOK {
		t.Fatalf("repeated review statuses = %d and %d, want 200", firstReview.Code, reloadedReview.Code)
	}
	if first.ApprovalToken == "" || first.ApprovalToken != reloaded.ApprovalToken {
		t.Fatalf("approval proof changed across tabs: %q then %q", first.ApprovalToken, reloaded.ApprovalToken)
	}

	otherReview, other := apitest.ReviewConnectionRequest(t, r, secondSession, started.UserCode)
	if otherReview.Code != http.StatusOK {
		t.Fatalf("second user review status = %d, want 200. body: %s", otherReview.Code, otherReview.Body.String())
	}
	if other.ApprovalToken == "" || other.ApprovalToken == first.ApprovalToken {
		t.Fatalf("approval proofs are not user-bound: first %q, second %q", first.ApprovalToken, other.ApprovalToken)
	}

	var reviewerMissing, proofMissing bool
	if err := pool.QueryRow(context.Background(), `
		select reviewed_by is null, review_token_hash is null
		  from connection_requests
		 where app_name = 'Example client' and name = 'review test'
	`).Scan(&reviewerMissing, &proofMissing); err != nil {
		t.Fatalf("read pending device review state: %v", err)
	}
	if !reviewerMissing || !proofMissing {
		t.Fatal("review GET claimed the device request or stored an approval proof")
	}

	crossUser := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": first.ApprovalToken}, secondSession,
	))
	if crossUser.Code != http.StatusNotFound {
		t.Fatalf("cross-user approval status = %d, want 404. body: %s", crossUser.Code, crossUser.Body.String())
	}

	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": first.ApprovalToken}, firstSession,
	))
	if approved.Code != http.StatusOK {
		t.Fatalf("approval after repeated reviews = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}
}

func TestDeviceDenialAndFastPollingReturnProtocolErrors(t *testing.T) {
	t.Parallel()
	t.Run("denial", func(t *testing.T) {
		r, session, _ := harness.NewConnectRouter(t)
		started, _ := apitest.StartConnection(t, r,
			apitest.ConnectionStartBody("Example client", "remote host", []string{"work:receive"}))
		_, pending := apitest.ReviewConnectionRequest(t, r, session, started.UserCode)

		denied := apitest.Send(t, r, apitest.BrowserRequest(
			t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/deny",
			map[string]string{"approvalToken": pending.ApprovalToken}, session,
		))
		apitest.AssertNoStore(t, denied)
		if denied.Code != http.StatusNoContent {
			t.Fatalf("deny status = %d, want 204. body: %s", denied.Code, denied.Body.String())
		}

		result := apitest.Poll(t, r, started.DeviceCode)
		if result.Code != http.StatusBadRequest || !strings.Contains(result.Body.String(), `"access_denied"`) {
			t.Fatalf("poll after denial = %d %s, want access_denied", result.Code, result.Body.String())
		}
	})

	t.Run("slow down", func(t *testing.T) {
		r, _, _ := harness.NewConnectRouter(t)
		started, _ := apitest.StartConnection(t, r,
			apitest.ConnectionStartBody("Example client", "remote host", []string{"work:receive"}))

		waiting := apitest.Poll(t, r, started.DeviceCode)
		if waiting.Code != http.StatusOK || apitest.DecodeResponse[apitest.PolledConnection](t, waiting).Status != "pending" {
			t.Fatalf("first poll = %d %s, want pending", waiting.Code, waiting.Body.String())
		}
		tooFast := apitest.Poll(t, r, started.DeviceCode)
		if tooFast.Code != http.StatusTooManyRequests || !strings.Contains(tooFast.Body.String(), `"slow_down"`) {
			t.Fatalf("fast poll = %d %s, want slow_down", tooFast.Code, tooFast.Body.String())
		}
		retryAfter, err := strconv.Atoi(tooFast.Header().Get("Retry-After"))
		if err != nil || retryAfter <= started.Interval {
			t.Errorf("Retry-After = %q, want an interval increased past %d", tooFast.Header().Get("Retry-After"), started.Interval)
		}
	})
}

func TestRefreshingRotatesTokensAndReuseRevokesTheConnectedApp(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	initial := apitest.ConnectApp(
		t, r, session, "Example client", "refresh test", []string{"work:receive"},
	)

	rotatedRec := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": initial.RefreshToken}))
	apitest.AssertNoStore(t, rotatedRec)
	if rotatedRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200. body: %s", rotatedRec.Code, rotatedRec.Body.String())
	}
	rotated := apitest.DecodeResponse[apitest.AppCredentials](t, rotatedRec)
	if rotated.RefreshToken == initial.RefreshToken || rotated.AccessToken == initial.AccessToken {
		t.Error("refresh returned one of the old tokens")
	}
	if rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", rotated.AccessToken, nil)); rec.Code != http.StatusOK {
		t.Fatalf("rotated access token status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	reused := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": initial.RefreshToken}))
	apitest.AssertNoStore(t, reused)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh token status = %d, want 401. body: %s", reused.Code, reused.Body.String())
	}
	for _, access := range []string{initial.AccessToken, rotated.AccessToken} {
		rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", access, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("access token still works after refresh reuse: status %d", rec.Code)
		}
	}
	revoked := connectedAppByID(t, listConnectedApps(t, r, session), initial.ConnectedApp.ID)
	if revoked.RevokedAt == nil {
		t.Error("refresh-token reuse did not leave a revoked connected app record")
	}
}

func TestAnIdleRefreshFamilyExpiresAndRevokesItsConnectedApp(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	credentials := apitest.ConnectApp(
		t, r, session, "Example client", "idle refresh test", []string{"work:receive"},
	)
	if _, err := pool.Exec(context.Background(),
		`update connected_apps
		    set connected_at = now() - interval '91 days', last_seen_at = null
		  where id = $1`, credentials.ConnectedApp.ID); err != nil {
		t.Fatalf("age linked connected app: %v", err)
	}

	refresh := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": credentials.RefreshToken}))
	apitest.AssertNoStore(t, refresh)
	if refresh.Code != http.StatusUnauthorized {
		t.Fatalf("idle refresh status = %d, want 401. body: %s", refresh.Code, refresh.Body.String())
	}
	if rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", credentials.AccessToken, nil)); rec.Code != http.StatusUnauthorized {
		t.Errorf("access token survived idle-family revocation: status %d", rec.Code)
	}
	if revoked := connectedAppByID(t, listConnectedApps(t, r, session), credentials.ConnectedApp.ID); revoked.RevokedAt == nil {
		t.Error("idle refresh family did not leave a revoked connected app record")
	}
}

func TestConnectedAppsOfOneAppStayIndependentThroughUpdateAndRevocation(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	first := apitest.ConnectApp(
		t, r, session, "Example client", "studio workstation", []string{"work:receive"},
	)
	second := apitest.ConnectApp(
		t, r, session, "Example client", "remote host", []string{"work:receive", "library:sync"},
	)
	if first.ConnectedApp.ID == second.ConnectedApp.ID || first.RefreshToken == second.RefreshToken {
		t.Fatal("two installations of one application share identity or credentials")
	}

	updatedCapabilities := []string{
		"example.client:library-report",
		"example.client:work-install",
	}
	updatedFormats := []string{"portable-lore-v1", "portable-card-v1"}
	update := apitest.Send(t, r, apitest.AsApp(t, http.MethodPut, "/v1/connected-apps/me", first.AccessToken, map[string]any{
		"appVersion":      "1.1.0",
		"protocolVersion": 1,
		"capabilities":    updatedCapabilities,
		"acceptedFormats": updatedFormats,
	}))
	apitest.AssertNoStore(t, update)
	if update.Code != http.StatusOK {
		t.Fatalf("capabilities update status = %d, want 200. body: %s", update.Code, update.Body.String())
	}
	updated := apitest.DecodeResponse[apitest.ConnectedApp](t, update)
	if !slices.Equal(updated.Capabilities, updatedCapabilities) || !slices.Equal(updated.AcceptedFormats, updatedFormats) {
		t.Errorf("updated capabilities lost order: capabilities %v, formats %v", updated.Capabilities, updated.AcceptedFormats)
	}
	if !slices.Equal(updated.Permissions, []string{"work:receive"}) {
		t.Errorf("capabilities update changed granted permissions to %v", updated.Permissions)
	}

	otherRec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", second.AccessToken, nil))
	apitest.AssertNoStore(t, otherRec)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("second connected app status = %d, want 200. body: %s", otherRec.Code, otherRec.Body.String())
	}
	other := apitest.DecodeResponse[apitest.ConnectedApp](t, otherRec)
	if !slices.Equal(other.Capabilities, []string{"example.client:work-install"}) ||
		!slices.Equal(other.AcceptedFormats, []string{"portable-card-v1"}) ||
		!slices.Equal(other.Permissions, []string{"work:receive", "library:sync"}) {
		t.Errorf("first connected app update changed the second: %+v", other)
	}

	revoke := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/connected-apps/"+first.ConnectedApp.ID, nil, session,
	))
	apitest.AssertNoStore(t, revoke)
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204. body: %s", revoke.Code, revoke.Body.String())
	}
	if rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", first.AccessToken, nil)); rec.Code != http.StatusUnauthorized {
		t.Errorf("revoked first connected app status = %d, want 401", rec.Code)
	}
	if rec := apitest.Send(t, r, apitest.AsApp(t, http.MethodGet, "/v1/connected-apps/me", second.AccessToken, nil)); rec.Code != http.StatusOK {
		t.Errorf("second connected app was affected by first revocation: status %d", rec.Code)
	}

	items := listConnectedApps(t, r, session)
	cut := connectedAppByID(t, items, first.ConnectedApp.ID)
	live := connectedAppByID(t, items, second.ConnectedApp.ID)
	if cut.RevokedAt == nil || cut.AppName != "Example client" || cut.Name != "studio workstation" {
		t.Errorf("revoked audit record = %+v", cut)
	}
	if cut.AppVersion != nil || cut.ProtocolVersion != nil || len(cut.Capabilities) != 0 || len(cut.AcceptedFormats) != 0 {
		t.Errorf("revoked connected app kept its live capabilities: %+v", cut)
	}
	if live.RevokedAt != nil || live.Name != "remote host" {
		t.Errorf("other connected app record = %+v", live)
	}
}

func TestConnectionSecretsAreHashedAndHumanCodesUseKeyedDigestsAtRest(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	started, _ := apitest.StartConnection(t, r,
		apitest.ConnectionStartBody("Example client", "storage test", []string{"work:receive"}))
	_, pending := apitest.ReviewConnectionRequest(t, r, session, started.UserCode)
	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": pending.ApprovalToken}, session,
	))
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}
	polled := apitest.Poll(t, r, started.DeviceCode)
	result := apitest.DecodeResponse[apitest.PolledConnection](t, polled)
	if result.AccessToken == nil || result.RefreshToken == nil || result.ConnectedApp == nil {
		t.Fatalf("poll did not return a token pair: %+v", result)
	}

	var userCodeHash []byte
	if err := pool.QueryRow(context.Background(), `select user_code_hash from connection_requests`).Scan(&userCodeHash); err != nil {
		t.Fatalf("read human-code digest: %v", err)
	}
	normalizedCode := strings.ReplaceAll(started.UserCode, "-", "")
	mac := hmac.New(sha256.New, testConnectHMACKey)
	mac.Write([]byte("user-code"))
	mac.Write([]byte{0})
	mac.Write([]byte(normalizedCode))
	if !hmac.Equal(userCodeHash, mac.Sum(nil)) {
		t.Errorf("stored human-code digest is not the expected keyed digest")
	}
	plainHash := sha256.Sum256([]byte(normalizedCode))
	if bytes.Equal(userCodeHash, plainHash[:]) || bytes.Contains(userCodeHash, []byte(normalizedCode)) {
		t.Error("human code is recoverable from its stored value")
	}

	var refreshHash []byte
	if err := pool.QueryRow(context.Background(),
		`select refresh_token_hash from connected_apps where id = $1`, result.ConnectedApp.ID,
	).Scan(&refreshHash); err != nil {
		t.Fatalf("read refresh-token hash: %v", err)
	}
	wantRefreshHash := sha256.Sum256([]byte(*result.RefreshToken))
	if !bytes.Equal(refreshHash, wantRefreshHash[:]) || bytes.Contains(refreshHash, []byte(*result.RefreshToken)) {
		t.Error("refresh token is not stored solely as its hash")
	}

	var accessHash []byte
	if err := pool.QueryRow(context.Background(),
		`select token_hash from app_access_tokens where connected_app_id = $1`, result.ConnectedApp.ID,
	).Scan(&accessHash); err != nil {
		t.Fatalf("read access-token hash: %v", err)
	}
	wantAccessHash := sha256.Sum256([]byte(*result.AccessToken))
	if !bytes.Equal(accessHash, wantAccessHash[:]) || bytes.Contains(accessHash, []byte(*result.AccessToken)) {
		t.Error("access token is not stored solely as its hash")
	}
}

func TestConnectBodiesStopAtFourKiBAndResponsesAreNotStored(t *testing.T) {
	t.Parallel()
	r, _, _ := harness.NewConnectRouter(t)
	body := `{"appName":"` + strings.Repeat("x", connect.MaxConnectBodyBytes) + `"}`
	rec := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/requests", body)
	apitest.AssertNoStore(t, rec)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized connect body status = %d, want 413. body: %s", rec.Code, rec.Body.String())
	}

	invalid := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/token", `{}`)
	apitest.AssertNoStore(t, invalid)
	if invalid.Code != http.StatusBadRequest {
		t.Errorf("invalid exchange status = %d, want 400", invalid.Code)
	}
}

func TestFiveCodeReviewsCannotBeResetByAValidCode(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)
	started, _ := apitest.StartConnection(t, r,
		apitest.ConnectionStartBody("Example client", "attempt test", []string{"work:receive"}))

	for attempt := 1; attempt <= 4; attempt++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/connect/requests/BBBB-CCCC", nil)
		rec := apitest.Send(t, r, apitest.Authorized(req, session))
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("wrong-code review %d status = %d, want 404", attempt, rec.Code)
		}
	}
	valid, pending := apitest.ReviewConnectionRequest(t, r, session, started.UserCode)
	if valid.Code != http.StatusOK || pending.ApprovalToken == "" {
		t.Fatalf("fifth, valid review = %d, approval token %q", valid.Code, pending.ApprovalToken)
	}

	for _, code := range []string{"BBBB-CCCC", started.UserCode} {
		req := httptest.NewRequest(http.MethodGet, "/v1/connect/requests/"+code, nil)
		rec := apitest.Send(t, r, apitest.Authorized(req, session))
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("review after five total attempts for %q = %d, want 429", code, rec.Code)
		}
	}
}
