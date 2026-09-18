package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const ReceiveScope = "asset:receive"

const LibrarySyncScope = "library:sync"

const (
	LumiverseInstalls   = "chat.lumiverse:extension-install"
	SillyTavernInstalls = "app.sillytavern:extension-install"
)

type StartedLink struct {
	DeviceCode      string    `json:"deviceCode"`
	UserCode        string    `json:"userCode"`
	VerificationURL string    `json:"verificationUrl"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Interval        int       `json:"interval"`
}

type PendingDeviceLink struct {
	ApplicationName    string    `json:"applicationName"`
	InstanceName       string    `json:"instanceName"`
	ApplicationVersion *string   `json:"applicationVersion"`
	ProtocolVersion    int       `json:"protocolVersion"`
	Capabilities       []string  `json:"capabilities"`
	AcceptedTargets    []string  `json:"acceptedTargets"`
	Scopes             []string  `json:"scopes"`
	ExpiresAt          time.Time `json:"expiresAt"`
	ApprovalToken      string    `json:"approvalToken"`
}

type PolledLink struct {
	Status               string          `json:"status"`
	AccessToken          *string         `json:"accessToken"`
	AccessTokenExpiresAt *time.Time      `json:"accessTokenExpiresAt"`
	RefreshToken         *string         `json:"refreshToken"`
	Instance             *LinkedInstance `json:"instance"`
}

type LinkedInstance struct {
	ID                 string     `json:"id"`
	ApplicationName    string     `json:"applicationName"`
	InstanceName       string     `json:"instanceName"`
	ApplicationVersion *string    `json:"applicationVersion"`
	ProtocolVersion    *int       `json:"protocolVersion"`
	Capabilities       []string   `json:"capabilities"`
	AcceptedTargets    []string   `json:"acceptedTargets"`
	Prefix             string     `json:"prefix"`
	Scopes             []string   `json:"scopes"`
	LinkedAt           time.Time  `json:"linkedAt"`
	LastSeenAt         *time.Time `json:"lastSeenAt"`
	RevokedAt          *time.Time `json:"revokedAt"`
}

type TokenGrant struct {
	AccessToken          string         `json:"accessToken"`
	AccessTokenExpiresAt time.Time      `json:"accessTokenExpiresAt"`
	RefreshToken         string         `json:"refreshToken"`
	Instance             LinkedInstance `json:"instance"`
}

func LinkStartBody(application, instance string, scopes []string) map[string]any {
	return map[string]any{
		"applicationName":    application,
		"instanceName":       instance,
		"applicationVersion": "1.0.0",
		"protocolVersion":    1,
		"capabilities":       []string{"example.client:asset-install"},
		"acceptedTargets":    []string{"portable-card-v1"},
		"scopes":             scopes,
	}
}

func StartLink(t *testing.T, r *gin.Engine, body map[string]any) (StartedLink, map[string]json.RawMessage) {
	t.Helper()
	rec := SendJSON(t, r, http.MethodPost, "/v1/link/requests", JSONText(t, body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("start link status = %d, want 201. body: %s", rec.Code, rec.Body.String())
	}
	AssertNoStore(t, rec)
	started := DecodeResponse[StartedLink](t, rec)
	return started, DecodeResponse[map[string]json.RawMessage](t, rec)
}

func ReviewDeviceLink(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	userCode string,
) (*httptest.ResponseRecorder, PendingDeviceLink) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/link/requests/"+userCode, nil)
	rec := Send(t, r, Authorized(req, session))
	AssertNoStore(t, rec)
	if rec.Code != http.StatusOK {
		return rec, PendingDeviceLink{}
	}
	return rec, DecodeResponse[PendingDeviceLink](t, rec)
}

func Poll(t *testing.T, r *gin.Engine, deviceCode string) *httptest.ResponseRecorder {
	t.Helper()
	rec := SendJSON(t, r, http.MethodPost, "/v1/link/poll",
		JSONText(t, map[string]string{"deviceCode": deviceCode}))
	AssertNoStore(t, rec)
	return rec
}

func LinkDeviceInstance(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	application string,
	instance string,
	scopes []string,
) TokenGrant {
	t.Helper()
	started, _ := StartLink(t, r, LinkStartBody(application, instance, scopes))
	review, pending := ReviewDeviceLink(t, r, session, started.UserCode)
	if review.Code != http.StatusOK || pending.ApprovalToken == "" {
		t.Fatalf("review status = %d, approval token = %q", review.Code, pending.ApprovalToken)
	}
	approve := Send(t, r, BrowserRequest(
		t, http.MethodPost, "/v1/link/requests/"+started.UserCode+"/approve",
		map[string]string{"approvalToken": pending.ApprovalToken}, session,
	))
	AssertNoStore(t, approve)
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200. body: %s", approve.Code, approve.Body.String())
	}
	rec := Poll(t, r, started.DeviceCode)
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	result := DecodeResponse[PolledLink](t, rec)
	if result.Status != "linked" || result.AccessToken == nil || result.RefreshToken == nil || result.Instance == nil {
		t.Fatalf("linked poll = %+v, want an instance and a token pair", result)
	}
	if result.AccessTokenExpiresAt == nil {
		t.Fatal("linked poll has no access-token expiry")
	}
	return TokenGrant{
		AccessToken:          *result.AccessToken,
		AccessTokenExpiresAt: *result.AccessTokenExpiresAt,
		RefreshToken:         *result.RefreshToken,
		Instance:             *result.Instance,
	}
}

type LibraryResult struct {
	Accepted int              `json:"accepted"`
	Removed  int              `json:"removed"`
	Ignored  int              `json:"ignored"`
	Withheld []WithheldNotice `json:"withheld"`
}

type WithheldNotice struct {
	WorkID     string    `json:"assetId"`
	Name       string    `json:"name"`
	WithheldAt time.Time `json:"withheldAt"`
}

func ReportInstalled(t *testing.T, r http.Handler, token, applicationVersion string, workIDs ...string) LibraryResult {
	t.Helper()
	entries := make([]map[string]any, 0, len(workIDs))
	for _, workID := range workIDs {
		entries = append(entries, map[string]any{"assetId": workID})
	}
	body := map[string]any{"snapshot": false, "entries": entries}
	if applicationVersion != "" {
		body["applicationVersion"] = applicationVersion
	}
	rec := Send(t, r, AsInstance(t, http.MethodPost, "/v1/library/sync", token, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("library sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return DecodeResponse[LibraryResult](t, rec)
}
