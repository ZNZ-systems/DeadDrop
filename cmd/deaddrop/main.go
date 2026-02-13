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
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/database"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/inbound"
	"github.com/znz-systems/deaddrop/internal/mail"
	"github.com/znz-systems/deaddrop/internal/mailbox"
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
	mailboxStore := postgres.NewMailboxStore(db)
	streamStore := postgres.NewStreamStore(db)
	conversationStore := postgres.NewConversationStore(db)

	// Services
	authService := auth.NewService(userStore, sessionStore, cfg.SessionMaxAge)
	domainService := domain.NewService(domainStore, &domain.NetResolver{})

	var msgNotifier message.Notifier
	var convNotifier conversation.Notifier
	var sender conversation.Sender
	if cfg.SMTPEnabled {
		smtpClient := mail.NewSMTPClient(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
		mailService := mail.NewService(smtpClient, userStore)
		msgNotifier = mailService
		convNotifier = mailService
		sender = mailService
	} else {
		msgNotifier = &message.NoopNotifier{}
		convNotifier = &conversation.NoopNotifier{}
		sender = &conversation.NoopSender{}
	}
	messageService := message.NewService(messageStore, domainStore, msgNotifier)
	mailboxService := mailbox.NewService(mailboxStore, domainStore)
	conversationService := conversation.NewService(conversationStore, mailboxStore, convNotifier, sender)

	// Rate limiter
	limiter := ratelimit.NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Renderer
	renderer := render.NewRenderer(templates.FS)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, renderer, cfg.SecureCookies)
	domainHandler := handlers.NewDomainHandler(domainService, messageStore, renderer, cfg.SecureCookies)
	messageHandler := handlers.NewMessageHandler(messageService, messageStore, domainStore, renderer)
	apiHandler := handlers.NewAPIHandler(messageService)
	mailboxHandler := handlers.NewMailboxHandler(mailboxService, conversationService, domainService, streamStore, conversationStore, renderer, cfg.SecureCookies)

	// Router
	router := web.NewRouter(web.RouterDeps{
		AuthHandler:    authHandler,
		DomainHandler:  domainHandler,
		MessageHandler: messageHandler,
		APIHandler:     apiHandler,
		MailboxHandler: mailboxHandler,
		AuthService:    authService,
		Renderer:       renderer,
		Limiter:        limiter,
		StaticFS:       static.FS,
		SecureCookies:  cfg.SecureCookies,
		DB:             db,
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

	// Inbound SMTP server
	if cfg.InboundSMTPEnabled {
		smtpSrv := inbound.NewServer(cfg.InboundSMTPAddr, cfg.InboundSMTPDomain, streamStore, conversationService)
		go func() {
			if err := smtpSrv.Start(); err != nil {
				slog.Error("inbound SMTP server error", "error", err)
			}
		}()
	}

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
