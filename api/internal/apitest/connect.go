package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const ReceivePermission = "work:receive"

const LibrarySyncPermission = "library:sync"

const (
	LumiverseInstalls   = "chat.lumiverse:extension-install"
	SillyTavernInstalls = "app.sillytavern:extension-install"
)

type StartedConnection struct {
	DeviceCode      string    `json:"deviceCode"`
	UserCode        string    `json:"userCode"`
	VerificationURL string    `json:"verificationUrl"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Interval        int       `json:"interval"`
}

type PendingConnection struct {
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

type PolledConnection struct {
	Status               string        `json:"status"`
	AccessToken          *string       `json:"accessToken"`
	AccessTokenExpiresAt *time.Time    `json:"accessTokenExpiresAt"`
	RefreshToken         *string       `json:"refreshToken"`
	ConnectedApp         *ConnectedApp `json:"connectedApp"`
}

type ConnectedApp struct {
	ID              string     `json:"id"`
	AppName         string     `json:"appName"`
	Name            string     `json:"name"`
	AppVersion      *string    `json:"appVersion"`
	ProtocolVersion *int       `json:"protocolVersion"`
	Capabilities    []string   `json:"capabilities"`
	AcceptedFormats []string   `json:"acceptedFormats"`
	Prefix          string     `json:"prefix"`
	Permissions     []string   `json:"permissions"`
	ConnectedAt     time.Time  `json:"connectedAt"`
	LastSeenAt      *time.Time `json:"lastSeenAt"`
	RevokedAt       *time.Time `json:"revokedAt"`
}

type AppCredentials struct {
	AccessToken          string       `json:"accessToken"`
	AccessTokenExpiresAt time.Time    `json:"accessTokenExpiresAt"`
	RefreshToken         string       `json:"refreshToken"`
	ConnectedApp         ConnectedApp `json:"connectedApp"`
}

func ConnectionStartBody(appName, name string, permissions []string) map[string]any {
	return map[string]any{
		"appName":         appName,
		"name":            name,
		"appVersion":      "1.0.0",
		"protocolVersion": 1,
		"capabilities":    []string{"example.client:work-install"},
		"acceptedFormats": []string{"portable-card-v1"},
		"permissions":     permissions,
	}
}

func StartConnection(t *testing.T, r *gin.Engine, body map[string]any) (StartedConnection, map[string]json.RawMessage) {
	t.Helper()
	rec := SendJSON(t, r, http.MethodPost, "/v1/connect/requests", JSONText(t, body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("start connection status = %d, want 201. body: %s", rec.Code, rec.Body.String())
	}
	AssertNoStore(t, rec)
	started := DecodeResponse[StartedConnection](t, rec)
	return started, DecodeResponse[map[string]json.RawMessage](t, rec)
}

func ReviewConnectionRequest(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	userCode string,
) (*httptest.ResponseRecorder, PendingConnection) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/connect/requests/"+userCode, nil)
	rec := Send(t, r, Authorized(req, session))
	AssertNoStore(t, rec)
	if rec.Code != http.StatusOK {
		return rec, PendingConnection{}
	}
	return rec, DecodeResponse[PendingConnection](t, rec)
}

func Poll(t *testing.T, r *gin.Engine, deviceCode string) *httptest.ResponseRecorder {
	t.Helper()
	rec := SendJSON(t, r, http.MethodPost, "/v1/connect/poll",
		JSONText(t, map[string]string{"deviceCode": deviceCode}))
	AssertNoStore(t, rec)
	return rec
}

func ConnectApp(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	appName string,
	name string,
	permissions []string,
) AppCredentials {
	t.Helper()
	started, _ := StartConnection(t, r, ConnectionStartBody(appName, name, permissions))
	review, pending := ReviewConnectionRequest(t, r, session, started.UserCode)
	if review.Code != http.StatusOK || pending.ApprovalToken == "" {
		t.Fatalf("review status = %d, approval token = %q", review.Code, pending.ApprovalToken)
	}
	approve := Send(t, r, BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve",
		map[string]any{"approvalToken": pending.ApprovalToken, "permissions": permissions}, session,
	))
	AssertNoStore(t, approve)
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d, want 200. body: %s", approve.Code, approve.Body.String())
	}
	rec := Poll(t, r, started.DeviceCode)
	if rec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	result := DecodeResponse[PolledConnection](t, rec)
	if result.Status != "connected" || result.AccessToken == nil || result.RefreshToken == nil || result.ConnectedApp == nil {
		t.Fatalf("connected poll = %+v, want a connected app and a token pair", result)
	}
	if result.AccessTokenExpiresAt == nil {
		t.Fatal("connected poll has no access-token expiry")
	}
	return AppCredentials{
		AccessToken:          *result.AccessToken,
		AccessTokenExpiresAt: *result.AccessTokenExpiresAt,
		RefreshToken:         *result.RefreshToken,
		ConnectedApp:         *result.ConnectedApp,
	}
}

type LibraryResult struct {
	Accepted  int              `json:"accepted"`
	Removed   int              `json:"removed"`
	Ignored   int              `json:"ignored"`
	Takedowns []TakedownNotice `json:"takedowns"`
}

type TakedownNotice struct {
	WorkID      string    `json:"workId"`
	Name        string    `json:"name"`
	TakenDownAt time.Time `json:"takenDownAt"`
}

func ReportLibrary(t *testing.T, r http.Handler, token, appVersion string, workIDs ...string) LibraryResult {
	t.Helper()
	entries := make([]map[string]any, 0, len(workIDs))
	for _, workID := range workIDs {
		entries = append(entries, map[string]any{"workId": workID})
	}
	body := map[string]any{"snapshot": false, "entries": entries}
	if appVersion != "" {
		body["appVersion"] = appVersion
	}
	rec := Send(t, r, AsApp(t, http.MethodPost, "/v1/library/sync", token, body))
	if rec.Code != http.StatusOK {
		t.Fatalf("library sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return DecodeResponse[LibraryResult](t, rec)
}
