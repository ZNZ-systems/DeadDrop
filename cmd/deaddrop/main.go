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
	"github.com/znz-systems/deaddrop/internal/config"
	"github.com/znz-systems/deaddrop/internal/database"
	"github.com/znz-systems/deaddrop/internal/domain"
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

	// Rate limiter
	limiter := ratelimit.NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Renderer
	renderer := render.NewRenderer(templates.FS)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, renderer)
	domainHandler := handlers.NewDomainHandler(domainService, messageStore, renderer)
	messageHandler := handlers.NewMessageHandler(messageService, renderer)
	apiHandler := handlers.NewAPIHandler(messageService)

	// Router
	router := web.NewRouter(web.RouterDeps{
		AuthHandler:    authHandler,
		DomainHandler:  domainHandler,
		MessageHandler: messageHandler,
		APIHandler:     apiHandler,
		AuthService:    authService,
		Renderer:       renderer,
		Limiter:        limiter,
		StaticFS:       static.FS,
	})

	// Session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := sessionStore.DeleteExpiredSessions(context.Background()); err != nil {
				slog.Error("failed to clean up expired sessions", "error", err)
			}
		}
	}()

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
