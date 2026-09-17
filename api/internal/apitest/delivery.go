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

type DeliveryArtifact struct {
	Kind    string  `json:"kind"`
	URL     string  `json:"url"`
	MediaID *string `json:"mediaId"`
	Role    *string `json:"role"`
	IsCover *bool   `json:"isCover"`
}

type DeliveryWork struct {
	ID                string             `json:"id"`
	AssetID           string             `json:"assetId"`
	ContentGeneration int                `json:"contentGeneration"`
	Kind              string             `json:"kind"`
	Name              string             `json:"name"`
	Format            string             `json:"format"`
	Label             string             `json:"label"`
	QueuedAt          time.Time          `json:"queuedAt"`
	LeaseExpiresAt    time.Time          `json:"leaseExpiresAt"`
	Artifacts         []DeliveryArtifact `json:"artifacts"`
}

type DeliveryWorkList struct {
	Deliveries []DeliveryWork   `json:"deliveries"`
	Withheld   []WithheldNotice `json:"withheld"`
}

type QueuedDelivery struct {
	ID             string     `json:"id"`
	InstanceID     string     `json:"instanceId"`
	AssetID        string     `json:"assetId"`
	State          string     `json:"state"`
	Reason         *string    `json:"reason"`
	QueuedAt       time.Time  `json:"queuedAt"`
	SettledAt      *time.Time `json:"settledAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	UpdatesInstall bool       `json:"updatesInstall"`
}

type AssetInstance struct {
	InstanceID          string          `json:"instanceId"`
	ApplicationName     string          `json:"applicationName"`
	InstanceName        string          `json:"instanceName"`
	LastSeenAt          *time.Time      `json:"lastSeenAt"`
	CanReceive          bool            `json:"canReceive"`
	ReportsLibrary      bool            `json:"reportsLibrary"`
	Delivery            *QueuedDelivery `json:"delivery"`
	InstalledGeneration *int            `json:"installedGeneration"`
	UpdateAvailable     bool            `json:"updateAvailable"`
}

type AssetInstanceList struct {
	ContentGeneration int             `json:"contentGeneration"`
	Items             []AssetInstance `json:"items"`
}

func (h Harness) NewLinkingRouter(t *testing.T) (*gin.Engine, *http.Cookie, *pgxpool.Pool) {
	t.Helper()
	return h.NewLinkingRouterWith(t, DeliverySettings())
}

func (h Harness) NewLinkingRouterWith(
	t *testing.T,
	settings connect.Settings,
) (*gin.Engine, *http.Cookie, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	outbox := &VerificationOutbox{}
	handlers := NewServicesWithDelivery(t, pool, 1<<20, outbox, settings, nil)
	router := h.RegisterRouter(t, handlers, api.DefaultDeadlines())

	session := SignUp(t, router, "creator@example.com", "linking.creator")
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

func AddVerifiedLinkingUser(
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
		t.Fatalf("verify second linking user: %v", err)
	}
	return session
}

func Declare(t *testing.T, r *gin.Engine, token string, capabilities, targets []string) {
	t.Helper()
	rec := Send(t, r, AsInstance(t, http.MethodPut, "/v1/instances/me", token, map[string]any{
		"applicationVersion": "1.0.0",
		"protocolVersion":    1,
		"capabilities":       capabilities,
		"acceptedTargets":    targets,
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("declare status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func DeclareTargets(t *testing.T, r *gin.Engine, token string, targets []string) {
	t.Helper()
	Declare(t, r, token, []string{}, targets)
}

func SendToInstance(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	assetID string,
	instanceID string,
) *httptest.ResponseRecorder {
	t.Helper()
	return Send(t, r, BrowserRequest(
		t, http.MethodPost, "/v1/assets/"+assetID+"/deliveries",
		map[string]string{"instanceId": instanceID}, session,
	))
}

func Collect(t *testing.T, r *gin.Engine, token string, acknowledge []string) *httptest.ResponseRecorder {
	t.Helper()
	if acknowledge == nil {
		acknowledge = []string{}
	}
	return Send(t, r, AsInstance(t, http.MethodPost, "/v1/deliveries/collect", token,
		map[string]any{"acknowledge": acknowledge}))
}

func AssetInstances(
	t *testing.T,
	r *gin.Engine,
	session *http.Cookie,
	assetID string,
) AssetInstanceList {
	t.Helper()
	rec := Send(t, r, Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/assets/"+assetID+"/instances", nil), session))
	if rec.Code != http.StatusOK {
		t.Fatalf("asset instances status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	return DecodeResponse[AssetInstanceList](t, rec)
}

func FetchSigned(t *testing.T, r *gin.Engine, address string) *httptest.ResponseRecorder {
	t.Helper()
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatalf("parse a delivery address: %v", err)
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
