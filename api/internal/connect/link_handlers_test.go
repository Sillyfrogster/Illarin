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

var testLinkHMACKey = []byte("01234567890123456789012345678901")

func listInstances(t *testing.T, r *gin.Engine, session *http.Cookie) []apitest.LinkedInstance {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/instances", nil)
	rec := apitest.Send(t, r, apitest.Authorized(req, session))
	apitest.AssertNoStore(t, rec)
	if rec.Code != http.StatusOK {
		t.Fatalf("list instances status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Items []apitest.LinkedInstance `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode instances: %v", err)
	}
	return list.Items
}

func instanceByID(t *testing.T, instances []apitest.LinkedInstance, id string) apitest.LinkedInstance {
	t.Helper()
	for _, instance := range instances {
		if instance.ID == id {
			return instance
		}
	}
	t.Fatalf("instance %s is absent from %+v", id, instances)
	return apitest.LinkedInstance{}
}

func TestDeviceLinkingRequiresManualReviewAndReturnsATokenPair(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewLinkingRouter(t)
	started, raw := apitest.StartLink(t, r,
		apitest.LinkStartBody("Example client", "studio workstation", []string{"asset:receive"}))

	if _, present := raw["verificationUrlComplete"]; present {
		t.Error("device response includes verificationUrlComplete; the human code must be entered manually")
	}
	if started.VerificationURL != apitest.BrowserOrigin+"/link" {
		t.Errorf("verification URL = %q", started.VerificationURL)
	}
	if len(started.UserCode) != 9 || started.UserCode[4] != '-' {
		t.Errorf("user code = %q, want eight characters split by a dash", started.UserCode)
	}
	if len(started.DeviceCode) < 40 || started.DeviceCode == started.UserCode {
		t.Errorf("private device code = %q, human code = %q", started.DeviceCode, started.UserCode)
	}

	review, pending := apitest.ReviewDeviceLink(t, r, session, started.UserCode)
	if review.Code != http.StatusOK {
		t.Fatalf("review status = %d, want 200. body: %s", review.Code, review.Body.String())
	}
	if pending.ApprovalToken == "" {
		t.Fatal("review response has no one-use approval token")
	}
	if pending.ApplicationName != "Example client" || pending.InstanceName != "studio workstation" {
		t.Errorf("pending identity = %q / %q", pending.ApplicationName, pending.InstanceName)
	}
	if !slices.Equal(pending.Scopes, []string{"asset:receive"}) {
		t.Errorf("pending scopes = %v", pending.Scopes)
	}

	withoutCSRF := httptest.NewRequest(
		http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
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
		t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": pending.ApprovalToken}, session,
	))
	apitest.AssertNoStore(t, approved)
	if approved.Code != http.StatusOK {
		t.Fatalf("approval status = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}

	linked := apitest.Poll(t, r, started.DeviceCode)
	if linked.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", linked.Code, linked.Body.String())
	}
	grant := apitest.DecodeResponse[apitest.PolledLink](t, linked)
	if grant.Status != "linked" || grant.AccessToken == nil || grant.RefreshToken == nil || grant.Instance == nil {
		t.Fatalf("poll after approval = %+v, want a token pair", grant)
	}
	if !strings.HasPrefix(*grant.AccessToken, "ia1.") || !strings.HasPrefix(*grant.RefreshToken, "ir1.") {
		t.Errorf("unexpected token types: access %q refresh %q", *grant.AccessToken, *grant.RefreshToken)
	}
	if grant.Instance.ApplicationName != "Example client" || grant.Instance.InstanceName != "studio workstation" {
		t.Errorf("linked instance = %+v", grant.Instance)
	}
}

func TestDeviceReviewsAreReadOnlyAndApprovalProofsStayWithTheirUser(t *testing.T) {
	t.Parallel()
	r, firstSession, pool := harness.NewLinkingRouter(t)
	secondSession := apitest.AddVerifiedLinkingUser(
		t, r, pool, "second.creator@example.com", "second.creator",
	)
	started, _ := apitest.StartLink(t, r,
		apitest.LinkStartBody("Example client", "review test", []string{"asset:receive"}))

	firstReview, first := apitest.ReviewDeviceLink(t, r, firstSession, started.UserCode)
	reloadedReview, reloaded := apitest.ReviewDeviceLink(t, r, firstSession, started.UserCode)
	if firstReview.Code != http.StatusOK || reloadedReview.Code != http.StatusOK {
		t.Fatalf("repeated review statuses = %d and %d, want 200", firstReview.Code, reloadedReview.Code)
	}
	if first.ApprovalToken == "" || first.ApprovalToken != reloaded.ApprovalToken {
		t.Fatalf("approval proof changed across tabs: %q then %q", first.ApprovalToken, reloaded.ApprovalToken)
	}

	otherReview, other := apitest.ReviewDeviceLink(t, r, secondSession, started.UserCode)
	if otherReview.Code != http.StatusOK {
		t.Fatalf("second user review status = %d, want 200. body: %s", otherReview.Code, otherReview.Body.String())
	}
	if other.ApprovalToken == "" || other.ApprovalToken == first.ApprovalToken {
		t.Fatalf("approval proofs are not user-bound: first %q, second %q", first.ApprovalToken, other.ApprovalToken)
	}

	var reviewerMissing, proofMissing bool
	if err := pool.QueryRow(context.Background(), `
		select reviewed_by is null, review_token_hash is null
		  from link_requests
		 where application_name = 'Example client' and instance_name = 'review test'
	`).Scan(&reviewerMissing, &proofMissing); err != nil {
		t.Fatalf("read pending device review state: %v", err)
	}
	if !reviewerMissing || !proofMissing {
		t.Fatal("review GET claimed the device request or stored an approval proof")
	}

	crossUser := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": first.ApprovalToken}, secondSession,
	))
	if crossUser.Code != http.StatusNotFound {
		t.Fatalf("cross-user approval status = %d, want 404. body: %s", crossUser.Code, crossUser.Body.String())
	}

	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": first.ApprovalToken}, firstSession,
	))
	if approved.Code != http.StatusOK {
		t.Fatalf("approval after repeated reviews = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}
}

func TestDeviceDenialAndFastPollingReturnProtocolErrors(t *testing.T) {
	t.Parallel()
	t.Run("denial", func(t *testing.T) {
		r, session, _ := harness.NewLinkingRouter(t)
		started, _ := apitest.StartLink(t, r,
			apitest.LinkStartBody("Example client", "remote host", []string{"asset:receive"}))
		_, pending := apitest.ReviewDeviceLink(t, r, session, started.UserCode)

		denied := apitest.Send(t, r, apitest.BrowserRequest(
			t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/deny",
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
		r, _, _ := harness.NewLinkingRouter(t)
		started, _ := apitest.StartLink(t, r,
			apitest.LinkStartBody("Example client", "remote host", []string{"asset:receive"}))

		waiting := apitest.Poll(t, r, started.DeviceCode)
		if waiting.Code != http.StatusOK || apitest.DecodeResponse[apitest.PolledLink](t, waiting).Status != "pending" {
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

func TestRefreshingRotatesTokensAndReuseRevokesTheInstance(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewLinkingRouter(t)
	initial := apitest.LinkDeviceInstance(
		t, r, session, "Example client", "refresh test", []string{"asset:receive"},
	)

	rotatedRec := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": initial.RefreshToken}))
	apitest.AssertNoStore(t, rotatedRec)
	if rotatedRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, want 200. body: %s", rotatedRec.Code, rotatedRec.Body.String())
	}
	rotated := apitest.DecodeResponse[apitest.TokenGrant](t, rotatedRec)
	if rotated.RefreshToken == initial.RefreshToken || rotated.AccessToken == initial.AccessToken {
		t.Error("refresh returned one of the old tokens")
	}
	if rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", rotated.AccessToken, nil)); rec.Code != http.StatusOK {
		t.Fatalf("rotated access token status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}

	reused := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": initial.RefreshToken}))
	apitest.AssertNoStore(t, reused)
	if reused.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh token status = %d, want 401. body: %s", reused.Code, reused.Body.String())
	}
	for _, access := range []string{initial.AccessToken, rotated.AccessToken} {
		rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", access, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("access token still works after refresh reuse: status %d", rec.Code)
		}
	}
	revoked := instanceByID(t, listInstances(t, r, session), initial.Instance.ID)
	if revoked.RevokedAt == nil {
		t.Error("refresh-token reuse did not leave a revoked instance record")
	}
}

func TestAnIdleRefreshFamilyExpiresAndRevokesItsInstance(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewLinkingRouter(t)
	grant := apitest.LinkDeviceInstance(
		t, r, session, "Example client", "idle refresh test", []string{"asset:receive"},
	)
	if _, err := pool.Exec(context.Background(),
		`update linked_instances
		    set linked_at = now() - interval '91 days', last_seen_at = null
		  where id = $1`, grant.Instance.ID); err != nil {
		t.Fatalf("age linked instance: %v", err)
	}

	refresh := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": grant.RefreshToken}))
	apitest.AssertNoStore(t, refresh)
	if refresh.Code != http.StatusUnauthorized {
		t.Fatalf("idle refresh status = %d, want 401. body: %s", refresh.Code, refresh.Body.String())
	}
	if rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", grant.AccessToken, nil)); rec.Code != http.StatusUnauthorized {
		t.Errorf("access token survived idle-family revocation: status %d", rec.Code)
	}
	if revoked := instanceByID(t, listInstances(t, r, session), grant.Instance.ID); revoked.RevokedAt == nil {
		t.Error("idle refresh family did not leave a revoked instance record")
	}
}

func TestSameApplicationInstancesStayIndependentThroughUpdateAndRevocation(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewLinkingRouter(t)
	first := apitest.LinkDeviceInstance(
		t, r, session, "Example client", "studio workstation", []string{"asset:receive"},
	)
	second := apitest.LinkDeviceInstance(
		t, r, session, "Example client", "remote host", []string{"asset:receive", "library:sync"},
	)
	if first.Instance.ID == second.Instance.ID || first.RefreshToken == second.RefreshToken {
		t.Fatal("two installations of one application share identity or credentials")
	}

	updatedCapabilities := []string{
		"example.client:library-report",
		"example.client:asset-install",
	}
	updatedTargets := []string{"portable-lore-v1", "portable-card-v1"}
	update := apitest.Send(t, r, apitest.AsInstance(t, http.MethodPut, "/v1/instances/me", first.AccessToken, map[string]any{
		"applicationVersion": "1.1.0",
		"protocolVersion":    1,
		"capabilities":       updatedCapabilities,
		"acceptedTargets":    updatedTargets,
	}))
	apitest.AssertNoStore(t, update)
	if update.Code != http.StatusOK {
		t.Fatalf("update declaration status = %d, want 200. body: %s", update.Code, update.Body.String())
	}
	updated := apitest.DecodeResponse[apitest.LinkedInstance](t, update)
	if !slices.Equal(updated.Capabilities, updatedCapabilities) || !slices.Equal(updated.AcceptedTargets, updatedTargets) {
		t.Errorf("updated declaration lost order: capabilities %v, targets %v", updated.Capabilities, updated.AcceptedTargets)
	}
	if !slices.Equal(updated.Scopes, []string{"asset:receive"}) {
		t.Errorf("declaration update changed granted scopes to %v", updated.Scopes)
	}

	otherRec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", second.AccessToken, nil))
	apitest.AssertNoStore(t, otherRec)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("second instance status = %d, want 200. body: %s", otherRec.Code, otherRec.Body.String())
	}
	other := apitest.DecodeResponse[apitest.LinkedInstance](t, otherRec)
	if !slices.Equal(other.Capabilities, []string{"example.client:asset-install"}) ||
		!slices.Equal(other.AcceptedTargets, []string{"portable-card-v1"}) ||
		!slices.Equal(other.Scopes, []string{"asset:receive", "library:sync"}) {
		t.Errorf("first instance update changed the second: %+v", other)
	}

	revoke := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodDelete, "/v1/instances/"+first.Instance.ID, nil, session,
	))
	apitest.AssertNoStore(t, revoke)
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want 204. body: %s", revoke.Code, revoke.Body.String())
	}
	if rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", first.AccessToken, nil)); rec.Code != http.StatusUnauthorized {
		t.Errorf("revoked first instance status = %d, want 401", rec.Code)
	}
	if rec := apitest.Send(t, r, apitest.AsInstance(t, http.MethodGet, "/v1/instances/me", second.AccessToken, nil)); rec.Code != http.StatusOK {
		t.Errorf("second instance was affected by first revocation: status %d", rec.Code)
	}

	items := listInstances(t, r, session)
	cut := instanceByID(t, items, first.Instance.ID)
	live := instanceByID(t, items, second.Instance.ID)
	if cut.RevokedAt == nil || cut.ApplicationName != "Example client" || cut.InstanceName != "studio workstation" {
		t.Errorf("revoked audit record = %+v", cut)
	}
	if cut.ApplicationVersion != nil || cut.ProtocolVersion != nil || len(cut.Capabilities) != 0 || len(cut.AcceptedTargets) != 0 {
		t.Errorf("revoked instance kept its live declaration: %+v", cut)
	}
	if live.RevokedAt != nil || live.InstanceName != "remote host" {
		t.Errorf("other instance record = %+v", live)
	}
}

func TestLinkSecretsAreHashedAndHumanCodesUseKeyedDigestsAtRest(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewLinkingRouter(t)
	started, _ := apitest.StartLink(t, r,
		apitest.LinkStartBody("Example client", "storage test", []string{"asset:receive"}))
	_, pending := apitest.ReviewDeviceLink(t, r, session, started.UserCode)
	approved := apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": pending.ApprovalToken}, session,
	))
	if approved.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200. body: %s", approved.Code, approved.Body.String())
	}
	polled := apitest.Poll(t, r, started.DeviceCode)
	result := apitest.DecodeResponse[apitest.PolledLink](t, polled)
	if result.AccessToken == nil || result.RefreshToken == nil || result.Instance == nil {
		t.Fatalf("poll did not return a token pair: %+v", result)
	}

	var userCodeHash []byte
	if err := pool.QueryRow(context.Background(), `select user_code_hash from link_requests`).Scan(&userCodeHash); err != nil {
		t.Fatalf("read human-code digest: %v", err)
	}
	normalizedCode := strings.ReplaceAll(started.UserCode, "-", "")
	mac := hmac.New(sha256.New, testLinkHMACKey)
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
		`select refresh_token_hash from linked_instances where id = $1`, result.Instance.ID,
	).Scan(&refreshHash); err != nil {
		t.Fatalf("read refresh-token hash: %v", err)
	}
	wantRefreshHash := sha256.Sum256([]byte(*result.RefreshToken))
	if !bytes.Equal(refreshHash, wantRefreshHash[:]) || bytes.Contains(refreshHash, []byte(*result.RefreshToken)) {
		t.Error("refresh token is not stored solely as its hash")
	}

	var accessHash []byte
	if err := pool.QueryRow(context.Background(),
		`select token_hash from instance_access_tokens where instance_id = $1`, result.Instance.ID,
	).Scan(&accessHash); err != nil {
		t.Fatalf("read access-token hash: %v", err)
	}
	wantAccessHash := sha256.Sum256([]byte(*result.AccessToken))
	if !bytes.Equal(accessHash, wantAccessHash[:]) || bytes.Contains(accessHash, []byte(*result.AccessToken)) {
		t.Error("access token is not stored solely as its hash")
	}
}

func TestLinkBodiesStopAtFourKiBAndResponsesAreNotStored(t *testing.T) {
	t.Parallel()
	r, _, _ := harness.NewLinkingRouter(t)
	body := `{"applicationName":"` + strings.Repeat("x", connect.MaxLinkBodyBytes) + `"}`
	rec := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/requests", body)
	apitest.AssertNoStore(t, rec)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized link body status = %d, want 413. body: %s", rec.Code, rec.Body.String())
	}

	invalid := apitest.SendJSON(t, r, http.MethodPost, "/v1/link/token", `{}`)
	apitest.AssertNoStore(t, invalid)
	if invalid.Code != http.StatusBadRequest {
		t.Errorf("invalid exchange status = %d, want 400", invalid.Code)
	}
}

func TestFiveCodeReviewsCannotBeResetByAValidCode(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewLinkingRouter(t)
	started, _ := apitest.StartLink(t, r,
		apitest.LinkStartBody("Example client", "attempt test", []string{"asset:receive"}))

	for attempt := 1; attempt <= 4; attempt++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/link/requests/BBBB-CCCC", nil)
		rec := apitest.Send(t, r, apitest.Authorized(req, session))
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("wrong-code review %d status = %d, want 404", attempt, rec.Code)
		}
	}
	valid, pending := apitest.ReviewDeviceLink(t, r, session, started.UserCode)
	if valid.Code != http.StatusOK || pending.ApprovalToken == "" {
		t.Fatalf("fifth, valid review = %d, approval token %q", valid.Code, pending.ApprovalToken)
	}

	for _, code := range []string{"BBBB-CCCC", started.UserCode} {
		req := httptest.NewRequest(http.MethodGet, "/v1/link/requests/"+code, nil)
		rec := apitest.Send(t, r, apitest.Authorized(req, session))
		apitest.AssertNoStore(t, rec)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("review after five total attempts for %q = %d, want 429", code, rec.Code)
		}
	}
}
