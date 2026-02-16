package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/znz-systems/deaddrop/internal/auth"
	"github.com/znz-systems/deaddrop/internal/blob"
	"github.com/znz-systems/deaddrop/internal/config"
	"github.com/znz-systems/deaddrop/internal/database"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/inbound"
	"github.com/znz-systems/deaddrop/internal/mail"
	"github.com/znz-systems/deaddrop/internal/message"
	"github.com/znz-systems/deaddrop/internal/ratelimit"
	"github.com/znz-systems/deaddrop/internal/store/postgres"
	"github.com/znz-systems/deaddrop/internal/web"
	"github.com/znz-systems/deaddrop/internal/web/handlers"
	"github.com/znz-systems/deaddrop/internal/web/render"
	"github.com/znz-systems/deaddrop/migrations"
	"github.com/znz-systems/deaddrop/static"
	"github.com/znz-systems/deaddrop/templates"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Migrations
	if err := database.RunMigrations(migrations.FS, cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Stores
	userStore := postgres.NewUserStore(db)
	sessionStore := postgres.NewSessionStore(db)
	domainStore := postgres.NewDomainStore(db)
	messageStore := postgres.NewMessageStore(db)
	inboundEmailStore := postgres.NewInboundEmailStore(db)
	inboundDomainConfigStore := postgres.NewInboundDomainConfigStore(db)
	inboundRuleStore := postgres.NewInboundRecipientRuleStore(db)
	inboundJobStore := postgres.NewInboundIngestJobStore(db)

	blobStore, err := blob.NewFromConfig(context.Background(), blob.Config{
		Backend:           cfg.BlobBackend,
		FSRoot:            cfg.BlobFSRoot,
		S3Bucket:          cfg.BlobS3Bucket,
		S3Region:          cfg.BlobS3Region,
		S3Endpoint:        cfg.BlobS3Endpoint,
		S3AccessKeyID:     cfg.BlobS3AccessKeyID,
		S3SecretAccessKey: cfg.BlobS3SecretAccessKey,
		S3ForcePathStyle:  cfg.BlobS3ForcePathStyle,
	})
	if err != nil {
		slog.Error("failed to initialize blob store", "backend", cfg.BlobBackend, "error", err)
		os.Exit(1)
	}

	// Services
	authService := auth.NewService(userStore, sessionStore, cfg.SessionMaxAge)
	domainService := domain.NewService(domainStore, &domain.NetResolver{})

	var notifier message.Notifier
	if cfg.SMTPEnabled {
		smtpClient := mail.NewSMTPClient(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
		notifier = mail.NewService(smtpClient, userStore)
	} else {
		notifier = &message.NoopNotifier{}
	}
	messageService := message.NewService(messageStore, domainStore, notifier)
	inboundDomainService := inbound.NewDomainService(inboundDomainConfigStore, &inbound.NetMXResolver{}, cfg.InboundMXTarget)
	inboundService := inbound.NewService(domainStore, inboundEmailStore, inboundDomainConfigStore, inboundRuleStore, blobStore)
	inboundWorker := inbound.NewWorker(inboundJobStore, inboundService, inbound.WorkerOptions{
		PollInterval: time.Duration(cfg.InboundWorkerPollMS) * time.Millisecond,
	})

	// Rate limiter
	limiter := ratelimit.NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Renderer
	renderer := render.NewRenderer(templates.FS)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, renderer, cfg.SecureCookies)
	domainHandler := handlers.NewDomainHandler(domainService, messageStore, renderer, cfg.BaseURL, cfg.SecureCookies)
	inboundDomainHandler := handlers.NewInboundDomainHandler(domainService, inboundDomainService, inboundRuleStore, renderer, cfg.SecureCookies)
	messageHandler := handlers.NewMessageHandler(messageService, messageStore, domainStore, renderer)
	apiHandler := handlers.NewAPIHandler(messageService, domainStore, cfg.APIMaxBodyBytes)
	inboxHandler := handlers.NewInboxHandler(inboundEmailStore, blobStore, renderer, cfg.SecureCookies)
	inboundAPIHandler := handlers.NewInboundAPIHandler(inboundJobStore, cfg.InboundAPIToken, cfg.InboundAPIMaxBodyBytes, cfg.InboundJobMaxAttempts)

	// Router
	router := web.NewRouter(web.RouterDeps{
		AuthHandler:          authHandler,
		DomainHandler:        domainHandler,
		InboundDomainHandler: inboundDomainHandler,
		MessageHandler:       messageHandler,
		APIHandler:           apiHandler,
		InboxHandler:         inboxHandler,
		InboundAPIHandler:    inboundAPIHandler,
		AuthService:          authService,
		Renderer:             renderer,
		Limiter:              limiter,
		StaticFS:             static.FS,
		SecureCookies:        cfg.SecureCookies,
		DB:                   db,
	})

	// Session cleanup goroutine
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := sessionStore.DeleteExpiredSessions(context.Background()); err != nil {
				slog.Error("failed to clean up expired sessions", "error", err)
			}
		}
	}()

	go inboundWorker.Run(appCtx)

	// Server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("DeadDrop starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down...")
	cancelApp()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
