package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/config"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/Sillyfrogster/Illarin/api/internal/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/Sillyfrogster/Illarin/api/internal/format/modules"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/postgres"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/summary"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtimeContext, cancelRuntime := context.WithCancel(signalContext)
	defer cancelRuntime()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	pool, err := postgres.NewPool(runtimeContext, cfg.Database)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(runtimeContext); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}

	blob, err := storage.NewStoreWithCapacity(pool, cfg.UploadsDir, storage.Capacity{
		FreeSpaceReserveBytes: cfg.StorageFreeSpaceReserveBytes,
		MaximumBlobWriteBytes: max(cfg.MaxUploadBytes, int64(cfg.ProbeLimits.MaxEntryBytes)),
	})
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	registry, err := modules.Registry()
	if err != nil {
		return err
	}

	svc := work.NewServiceForSite(
		pool, registry, blob, cfg.ProbeLimits, cfg.SiteURL, cfg.AccountStorageCapBytes,
	)
	uploads := upload.NewService(pool, svc)
	recomputed, err := summary.RecomputeStaleFormats(runtimeContext, pool, registry)
	if err != nil {
		return fmt.Errorf("export summaries: %w", err)
	}
	if recomputed > 0 {
		log.Printf("recomputed the export summary for %d works", recomputed)
	}
	remeasured, err := summary.RecomputeStaleFilters(runtimeContext, pool, registry)
	if err != nil {
		return fmt.Errorf("facet summaries: %w", err)
	}
	if remeasured > 0 {
		log.Printf("recomputed the facet summary for %d works", remeasured)
	}
	var background sync.WaitGroup
	background.Add(2)
	go func() {
		defer background.Done()
		uploads.RunIngestWorkers(runtimeContext, cfg.IngestWorkers, func(err error) {
			log.Printf("ingest worker: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		storage.NewSweeper(pool, blob).RunSweeper(runtimeContext, func(err error) {
			log.Printf("blob sweeper: %v", err)
		})
	}()
	defer func() {
		cancelRuntime()
		background.Wait()
	}()
	var verificationSender account.EmailSender = account.NewLogVerificationSender(log.Default())
	if cfg.Microsoft365.ClientID != "" {
		verificationSender, err = account.NewMicrosoftGraphSender(account.MicrosoftGraphSettings{
			TenantID:     cfg.Microsoft365.TenantID,
			ClientID:     cfg.Microsoft365.ClientID,
			ClientSecret: cfg.Microsoft365.ClientSecret,
			Mailbox:      cfg.Microsoft365.Mailbox,
		})
		if err != nil {
			return fmt.Errorf("Microsoft 365 email: %w", err)
		}
	} else if cfg.SMTP.Address != "" {
		verificationSender, err = account.NewSMTPSender(account.SMTPSettings{
			Address:  cfg.SMTP.Address,
			From:     cfg.SMTP.From,
			Username: cfg.SMTP.Username,
			Password: cfg.SMTP.Password,
		})
		if err != nil {
			return fmt.Errorf("verification email: %w", err)
		}
	}
	var discordProvider account.DiscordProvider
	if cfg.Discord.ClientID != "" {
		discordProvider, err = discord.NewClient(discord.DefaultConfig(
			cfg.Discord.ClientID,
			cfg.Discord.ClientSecret,
			strings.TrimRight(cfg.SiteURL, "/")+"/api/v1/auth/discord/callback",
		), nil)
		if err != nil {
			return fmt.Errorf("Discord sign-in: %w", err)
		}
	}
	images := mediaproc.NewLibrary(blob, mediaproc.NewProcessor(mediaproc.DefaultLimits()), 1)
	accounts := account.NewService(pool, verificationSender, discordProvider, images, cfg.SiteURL)
	sealing, err := secrets.NewKey(cfg.PublicationSecretKey)
	if err != nil {
		return fmt.Errorf("publication secret key: %w", err)
	}
	publishing := blog.DefaultPublishing(sealing, cfg.SiteURL, cfg.BlogURL)
	publications := blog.NewService(pool, images, publishing)
	updateDestinations := integration.NewService(pool, sealing, publishing.Sender, cfg.SiteURL)
	versions := version.NewService(pool, svc)
	versions.OnPublished(updateDestinations.Announce, version.TellFollowers)
	links := connect.NewApps(pool, cfg.SiteURL, cfg.LinkingHMACKey)
	deliveries := connect.NewSends(pool, svc, links, connect.DefaultSettings())
	notifications := notify.NewService(pool)
	background.Add(8)
	go func() {
		defer background.Done()
		notifications.RunFanOut(runtimeContext, func(err error) {
			log.Printf("notification fan-out: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		notifications.RunSweeper(runtimeContext, func(err error) {
			log.Printf("notification sweeper: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		updateDestinations.RunSweeper(runtimeContext, func(err error) {
			log.Printf("update destination sweeper: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		updateDestinations.RunAnnouncements(runtimeContext, func(err error) {
			log.Printf("update announcement: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		deliveries.RunSweeper(runtimeContext, func(err error) {
			log.Printf("delivery sweeper: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		publications.RunScheduler(runtimeContext, func(err error) {
			log.Printf("publication scheduler: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		publications.RunRecovery(runtimeContext, func(err error) {
			log.Printf("publication recovery: %v", err)
		})
	}()
	go func() {
		defer background.Done()
		publications.RunDeliveries(runtimeContext, func(err error) {
			log.Printf("publication delivery: %v", err)
		})
	}()

	r := gin.New()
	r.Use(api.Recovery(log.Default()))
	running := services{
		Works:              svc,
		Pages:              page.NewService(pool, svc),
		Blocks:             edit.NewService(pool, svc),
		Versions:           versions,
		Uploads:            uploads,
		Downloads:          download.NewService(pool, svc),
		Accounts:           accounts,
		Links:              links,
		Deliveries:         deliveries,
		Publications:       publications,
		UpdateDestinations: updateDestinations,
		Notifications:      notifications,
		MaxUploadBytes:     cfg.MaxUploadBytes,
	}
	ready := func(ctx context.Context) error {
		if err := pool.Ping(ctx); err != nil {
			return err
		}
		for _, directory := range []string{"blobs", "derivatives"} {
			info, err := os.Stat(filepath.Join(cfg.UploadsDir, directory))
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", directory)
			}
		}
		return nil
	}
	if err := registerRoutes(r, running, cfg.Deadlines, ready); err != nil {
		return fmt.Errorf("routes: %w", err)
	}

	server := newServer(":"+cfg.Port, r, cfg.Server)
	log.Printf("listening on %s", server.Addr)
	serverError := make(chan error, 1)
	go func() { serverError <- server.ListenAndServe() }()

	select {
	case err := <-serverError:
		cancelRuntime()
		background.Wait()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server: %w", err)
	case <-signalContext.Done():
		log.Print("shutting down")
		shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelShutdown()
		shutdownError := server.Shutdown(shutdownContext)
		cancelRuntime()
		background.Wait()
		if shutdownError != nil {
			_ = server.Close()
			return fmt.Errorf("server shutdown: %w", shutdownError)
		}
		return nil
	}
}
