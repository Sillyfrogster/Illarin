package apitest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SendFile struct {
	Type    string  `json:"type"`
	URL     string  `json:"url"`
	MediaID *string `json:"mediaId"`
	Role    *string `json:"role"`
	IsCover *bool   `json:"isCover"`
}

type CollectedSend struct {
	ID             string     `json:"id"`
	WorkID         string     `json:"workId"`
	VersionNumber  int        `json:"versionNumber"`
	Type           string     `json:"type"`
	Name           string     `json:"name"`
	Format         string     `json:"format"`
	Label          string     `json:"label"`
	QueuedAt       time.Time  `json:"queuedAt"`
	LeaseExpiresAt time.Time  `json:"leaseExpiresAt"`
	Files          []SendFile `json:"files"`
}

type CollectedSends struct {
	Sends    []CollectedSend  `json:"sends"`
	Withheld []WithheldNotice `json:"withheld"`
}

type QueuedSend struct {
	ID             string     `json:"id"`
	ConnectedAppID string     `json:"connectedAppId"`
	WorkID         string     `json:"workId"`
	State          string     `json:"state"`
	Reason         *string    `json:"reason"`
	QueuedAt       time.Time  `json:"queuedAt"`
	SettledAt      *time.Time `json:"settledAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	UpdatesInstall bool       `json:"updatesInstall"`
}

type WorkConnectedApp struct {
	ConnectedAppID   string      `json:"connectedAppId"`
	AppName          string      `json:"appName"`
	Name             string      `json:"name"`
	LastSeenAt       *time.Time  `json:"lastSeenAt"`
	CanReceive       bool        `json:"canReceive"`
	ReportsLibrary   bool        `json:"reportsLibrary"`
	Send             *QueuedSend `json:"send"`
	InstalledVersion *int        `json:"installedVersion"`
	UpdateAvailable  bool        `json:"updateAvailable"`
}

type WorkConnectedAppList struct {
	VersionNumber int                `json:"versionNumber"`
	Items         []WorkConnectedApp `json:"items"`
}

func (h Harness) NewConnectRouter(t *testing.T) (*gin.Engine, *http.Cookie, *pgxpool.Pool) {
	t.Helper()
	return h.NewConnectRouterWith(t, SendSettings())
}

func (h Harness) NewConnectRouterWith(
	t *testing.T,
	settings connect.Settings,
) (*gin.Engine, *http.Cookie, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	outbox := &VerificationOutbox{}
	handlers := NewServicesWithSends(t, pool, 1<<20, outbox, settings, nil)
	router := h.RegisterRouter(t, handlers, api.DefaultDeadlines())

	session := SignUp(t, router, "creator@example.com", "connect.creator")
	link, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	rec := SendJSON(t, router, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+link.Query().Get("token")+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify creator: %d %s", rec.Code, rec.Body.String())
	}
	return router, session, pool
}

func AddVerifiedUser(
	t *testing.T,
	r *gin.Engine,
	pool *pgxpool.Pool,
	email string,
	handle string,
) *http.Cookie {
	t.Helper()
	session := SignUp(t, r, email, handle)
	if _, err := pool.Exec(
		context.Background(),
		`update users set email_verified_at = now() where email = $1`,
		email,
	); err != nil {
		t.Fatalf("verify second user: %v", err)
	}
	return session
}

func DeclareCapabilities(t *testing.T, r *gin.Engine, token string, capabilities, formats []string) {
	t.Helper()
	rec := Send(t, r, AsApp(t, http.MethodPut, "/v1/connected-apps/me", token, map[string]any{
		"appVersion":      "1.0.0",
		"protocolVersion": 1,
		"capabilities":    capabilities,
		"acceptedFormats": formats,
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func DeclareFormats(t *testing.T, r *gin.Engine, token string, formats []string) {
	t.Helper()
	DeclareCapabilities(t, r, token, []string{}, formats)
}

func SendToApp(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	workID string,
	connectedAppID string,
) *httptest.ResponseRecorder {
	t.Helper()
	return Send(t, r, BrowserRequest(
		t, http.MethodPost, "/v1/works/"+workID+"/sends",
		map[string]string{"connectedAppId": connectedAppID}, session,
	))
}

func Collect(t *testing.T, r *gin.Engine, token string, acknowledge []string) *httptest.ResponseRecorder {
	t.Helper()
	if acknowledge == nil {
		acknowledge = []string{}
	}
	return Send(t, r, AsApp(t, http.MethodPost, "/v1/sends/collect", token,
		map[string]any{"acknowledge": acknowledge}))
}

func WorkConnectedApps(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	workID string,
) WorkConnectedAppList {
	t.Helper()
	rec := Send(t, r, Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"/connected-apps", nil), session))
	if rec.Code != http.StatusOK {
		t.Fatalf("work connected apps status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return DecodeResponse[WorkConnectedAppList](t, rec)
}

func FetchSigned(t *testing.T, r *gin.Engine, address string) *httptest.ResponseRecorder {
	t.Helper()
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatalf("parse a send address: %v", err)
	}
	return Send(t, r, httptest.NewRequest(http.MethodGet, parsed.RequestURI(), nil))
}

func DownloadEventCount(t *testing.T, pool *pgxpool.Pool, class string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from download_events where authorization_class = $1`, class,
	).Scan(&count); err != nil {
		t.Fatalf("count download events: %v", err)
	}
	return count
}
