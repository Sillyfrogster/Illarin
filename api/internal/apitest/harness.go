package apitest

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/extension"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Services are the running parts a test router serves
type Services struct {
	Works              *work.Service
	Pages              *page.Service
	Blocks             *edit.Service
	Versions           *version.Service
	Uploads            *upload.Service
	Downloads          *download.Service
	Accounts           *account.Service
	Apps               *connect.Apps
	Sends              *connect.Sends
	Publications       *blog.Service
	UpdateDestinations *integration.Service
	Notifications      *notify.Service
	MaxUploadBytes     int64
}

// Register puts every route on a router, the way the server does
type Register func(r *gin.Engine, services Services, deadlines api.Deadlines) error

// Harness builds test routers with one way of registering routes
type Harness struct {
	Register Register
}

// Ready is a readiness check that always passes
func Ready(context.Context) error { return nil }

func (h Harness) NewRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return h.NewRouterWithCeiling(t, 1<<20)
}

func (h Harness) NewRouterWithCeiling(t *testing.T, maxUploadBytes int64) *gin.Engine {
	t.Helper()
	return h.NewRouterWith(t, maxUploadBytes, api.DefaultDeadlines())
}

func (h Harness) NewRouterWith(t *testing.T, maxUploadBytes int64, deadlines api.Deadlines) *gin.Engine {
	t.Helper()
	return h.NewRouterWithSender(t, maxUploadBytes, deadlines, &VerificationOutbox{})
}

func (h Harness) NewRouterWithSender(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
	sender account.EmailSender,
) *gin.Engine {
	t.Helper()
	return h.RegisterRouter(t, NewServices(t, maxUploadBytes, sender), deadlines)
}

func (h Harness) NewRouterWithSenderAndPool(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
	sender account.EmailSender,
) (*gin.Engine, *pgxpool.Pool) {
	t.Helper()
	router, pool, _ := h.NewRouterWithSenderPoolAndServices(t, maxUploadBytes, deadlines, sender)
	return router, pool
}

func (h Harness) NewRouterWithSenderPoolAndServices(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
	sender account.EmailSender,
) (*gin.Engine, *pgxpool.Pool, Services) {
	t.Helper()
	pool := testdb.Connect(t)
	services := NewServicesWithPool(t, pool, maxUploadBytes, sender)
	return h.RegisterRouter(t, services, deadlines), pool, services
}

func NewServices(t *testing.T, maxUploadBytes int64, sender account.EmailSender) Services {
	t.Helper()
	return NewServicesWithPool(t, testdb.Connect(t), maxUploadBytes, sender)
}

func NewServicesWithPool(
	t *testing.T,
	pool *pgxpool.Pool,
	maxUploadBytes int64,
	sender account.EmailSender,
) Services {
	t.Helper()
	return NewServicesWithSends(t, pool, maxUploadBytes, sender, SendSettings(), nil)
}

func NewServicesWithSends(
	t *testing.T,
	pool *pgxpool.Pool,
	maxUploadBytes int64,
	sender account.EmailSender,
	settings connect.Settings,
	to blog.Sender,
) Services {
	t.Helper()
	blob, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	works := work.NewService(pool, Registry(t), blob)
	accounts := NewAccounts(pool, sender, nil, MediaLibrary(blob))
	apps := NewAppsService(pool)
	updateDestinations := integration.NewService(
		pool, SealingKey(), Publishing(to).Sender, "http://localhost:3000",
	)
	versions := version.NewService(pool, works)
	versions.OnPublished(updateDestinations.Announce, version.TellFollowers)
	return Services{
		Works:              works,
		Pages:              page.NewService(pool, works),
		Blocks:             edit.NewService(pool, works),
		Versions:           versions,
		Uploads:            upload.NewService(pool, works),
		Downloads:          download.NewService(pool, works),
		Accounts:           accounts,
		Apps:               apps,
		Sends:              connect.NewSends(pool, works, apps, settings),
		Publications:       blog.NewService(pool, MediaLibrary(blob), Publishing(to)),
		UpdateDestinations: updateDestinations,
		Notifications:      NewNotifications(pool),
		MaxUploadBytes:     maxUploadBytes,
	}
}

// NewServicesOver builds services over an existing pool, store and work service
func NewServicesOver(
	pool *pgxpool.Pool,
	blobs storage.Store,
	works *work.Service,
	sender account.EmailSender,
	provider account.DiscordProvider,
) Services {
	apps := NewAppsService(pool)
	destinations := NewUpdateDestinations(pool)
	versions := version.NewService(pool, works)
	versions.OnPublished(destinations.Announce, version.TellFollowers)
	return Services{
		Works:              works,
		Pages:              page.NewService(pool, works),
		Blocks:             edit.NewService(pool, works),
		Versions:           versions,
		Uploads:            upload.NewService(pool, works),
		Downloads:          download.NewService(pool, works),
		Accounts:           NewAccounts(pool, sender, provider, MediaLibrary(blobs)),
		Apps:               apps,
		Sends:              NewSendsService(pool, works, apps),
		Publications:       NewPublicationService(pool, blobs),
		UpdateDestinations: destinations,
		Notifications:      NewNotifications(pool),
		MaxUploadBytes:     1 << 20,
	}
}

func (h Harness) NewRouterWithDiscord(t *testing.T, provider account.DiscordProvider) *gin.Engine {
	t.Helper()
	r, _, _ := h.NewDiscordStack(t, provider)
	return r
}

func (h Harness) NewRouterWithDiscordAndOutbox(
	t *testing.T,
	provider account.DiscordProvider,
) (*gin.Engine, *VerificationOutbox) {
	t.Helper()
	r, outbox, _ := h.NewDiscordStack(t, provider)
	return r, outbox
}

func (h Harness) NewDiscordStack(
	t *testing.T,
	provider account.DiscordProvider,
) (*gin.Engine, *VerificationOutbox, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	blob, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	works := work.NewService(pool, Registry(t), blob)
	outbox := &VerificationOutbox{}
	services := NewServicesOver(pool, blob, works, outbox, provider)
	return h.RegisterRouter(t, services, api.DefaultDeadlines()), outbox, pool
}

func (h Harness) RegisterRouter(t *testing.T, services Services, deadlines api.Deadlines) *gin.Engine {
	t.Helper()
	r := gin.New()
	if err := h.Register(r, services, deadlines); err != nil {
		t.Fatalf("register: %v", err)
	}
	return r
}

func (h Harness) NewVerifiedRouter(t *testing.T) (*gin.Engine, *http.Cookie) {
	t.Helper()
	return h.NewVerifiedRouterWith(t, 1<<20, api.DefaultDeadlines())
}

func (h Harness) NewVerifiedRouterWith(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
) (*gin.Engine, *http.Cookie) {
	t.Helper()
	router, session, _ := h.NewVerifiedRouterWithService(t, maxUploadBytes, deadlines)
	return router, session
}

func (h Harness) NewVerifiedRouterWithService(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
) (*gin.Engine, *http.Cookie, *work.Service) {
	t.Helper()
	_, router, session, works := h.NewVerifiedRoutersWithService(t, maxUploadBytes, deadlines)
	return router, session, works
}

func (h Harness) NewVerifiedRoutersWithService(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
) (*gin.Engine, *gin.Engine, *http.Cookie, *work.Service) {
	t.Helper()
	setupRouter, router, session, works, _ := h.NewVerifiedRoutersWithPool(t, maxUploadBytes, deadlines)
	return setupRouter, router, session, works
}

func (h Harness) NewVerifiedRoutersWithPool(
	t *testing.T,
	maxUploadBytes int64,
	deadlines api.Deadlines,
) (*gin.Engine, *gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	outbox := &VerificationOutbox{}
	setupRouter, pool, services := h.NewRouterWithSenderPoolAndServices(
		t, maxUploadBytes, api.DefaultDeadlines(), outbox,
	)
	session := VerifiedSignUp(t, setupRouter, outbox, "verified@example.com", "verified.creator")
	return setupRouter, h.RegisterRouter(t, services, deadlines), session, services.Works, pool
}

func (h Harness) NewVerifiedIngestRouter(
	t *testing.T,
	registry *format.Registry,
) (*gin.Engine, *http.Cookie, *work.Service) {
	t.Helper()
	router, session, works, _ := h.NewVerifiedIngestRouterWithSettings(t, registry, work.DefaultIngestSettings())
	return router, session, works
}

func (h Harness) NewVerifiedIngestRouterWithPool(
	t *testing.T,
	registry *format.Registry,
) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	return h.NewVerifiedIngestRouterWithSettings(t, registry, work.DefaultIngestSettings())
}

func (h Harness) NewVerifiedIngestRouterWithSettings(
	t *testing.T,
	registry *format.Registry,
	settings work.IngestSettings,
) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	return h.NewVerifiedIngestRouterWithStore(t, registry, settings, nil)
}

func (h Harness) NewVerifiedIngestRouterWithStore(
	t *testing.T,
	registry *format.Registry,
	settings work.IngestSettings,
	decorate func(storage.Store) storage.Store,
) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	return h.NewVerifiedIngestRouterWithStoreFactory(t, registry, settings,
		func(pool *pgxpool.Pool) (storage.Store, error) {
			blobs, err := storage.NewStore(pool, t.TempDir())
			if err == nil && decorate != nil {
				blobs = decorate(blobs)
			}
			return blobs, err
		},
	)
}

func (h Harness) NewVerifiedIngestRouterWithStoreFactory(
	t *testing.T,
	registry *format.Registry,
	settings work.IngestSettings,
	storeFactory func(*pgxpool.Pool) (storage.Store, error),
) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	pool := testdb.Connect(t)
	blobs, err := storeFactory(pool)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	if registry.Empty() {
		if err := registry.Register(OpaqueModule{}); err != nil {
			t.Fatalf("register test format: %v", err)
		}
	}
	works := work.NewServiceWithIngestSettings(pool, registry, blobs, settings)
	outbox := &VerificationOutbox{}
	services := NewServicesOver(pool, blobs, works, outbox, nil)
	setup := h.RegisterRouter(t, services, api.DefaultDeadlines())
	session := SignUp(t, setup, "verified@example.com", "verified.creator")
	verificationURL, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	verified := SendJSON(t, setup, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify test account: %d %s", verified.Code, verified.Body.String())
	}
	return h.RegisterRouter(t, services, api.DefaultDeadlines()), session, works, pool
}

// NewExtensionRouter serves a signed-in creator with every extension format registered
func (h Harness) NewExtensionRouter(t *testing.T) (*gin.Engine, *http.Cookie, *work.Service, *pgxpool.Pool) {
	t.Helper()
	registry := format.NewRegistry()
	for _, module := range extension.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	return h.NewVerifiedIngestRouterWithPool(t, registry)
}
