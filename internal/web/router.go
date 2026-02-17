package web

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/znz-systems/deaddrop/internal/auth"
	"github.com/znz-systems/deaddrop/internal/ratelimit"
	"github.com/znz-systems/deaddrop/internal/web/handlers"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

// RouterDeps holds all dependencies needed to build the router.
type RouterDeps struct {
	AuthHandler    *handlers.AuthHandler
	DomainHandler  *handlers.DomainHandler
	MessageHandler *handlers.MessageHandler
	APIHandler     *handlers.APIHandler
	MailboxHandler *handlers.MailboxHandler
	AuthService    *auth.Service
	Renderer       *render.Renderer
	Limiter        *ratelimit.Limiter
	StaticFS       fs.FS
	SecureCookies  bool
	DB             interface{ PingContext(ctx context.Context) error }
}

// NewRouter wires all routes into a Chi router.
func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.SecurityHeaders)

	// Serve static files
	fileServer := http.FileServer(http.FS(deps.StaticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	csrf := middleware.CSRFWithOptions(deps.SecureCookies)

	// Public auth routes (with CSRF + rate limiting on POST)
	r.Group(func(r chi.Router) {
		r.Use(csrf)
		r.Use(middleware.OptionalAuth(deps.AuthService))

		r.Get("/login", deps.AuthHandler.ShowLogin)
		r.Get("/signup", deps.AuthHandler.ShowSignup)
		r.Post("/logout", deps.AuthHandler.HandleLogout)

		// Rate-limit login and signup POST endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimit(deps.Limiter))
			r.Post("/login", deps.AuthHandler.HandleLogin)
			r.Post("/signup", deps.AuthHandler.HandleSignup)
		})
	})

	// Authenticated dashboard routes (with CSRF + RequireAuth)
	r.Group(func(r chi.Router) {
		r.Use(csrf)
		r.Use(middleware.RequireAuth(deps.AuthService))

		r.Get("/", deps.DomainHandler.ShowDashboard)
		r.Get("/domains/new", deps.DomainHandler.ShowNewDomain)
		r.Post("/domains", deps.DomainHandler.HandleCreateDomain)
		r.Get("/domains/{id}", deps.DomainHandler.ShowDomainDetail)
		r.Post("/domains/{id}/verify", deps.DomainHandler.HandleVerifyDomain)
		r.Post("/domains/{id}/delete", deps.DomainHandler.HandleDeleteDomain)

		r.Post("/messages/{messageID}/read", deps.MessageHandler.HandleMarkRead)
		r.Delete("/messages/{messageID}", deps.MessageHandler.HandleDeleteMessage)

		// Mailbox routes
		r.Get("/mailboxes", deps.MailboxHandler.ShowDashboard)
		r.Get("/mailboxes/new", deps.MailboxHandler.ShowNewMailbox)
		r.Post("/mailboxes", deps.MailboxHandler.HandleCreateMailbox)
		r.Get("/mailboxes/{id}", deps.MailboxHandler.ShowMailboxDetail)
		r.Post("/mailboxes/{id}/delete", deps.MailboxHandler.HandleDeleteMailbox)
		r.Post("/mailboxes/{id}/streams", deps.MailboxHandler.HandleAddStream)
		r.Post("/mailboxes/{id}/streams/{sid}/delete", deps.MailboxHandler.HandleDeleteStream)
		r.Get("/mailboxes/{id}/conversations/{cid}", deps.MailboxHandler.ShowConversation)
		r.Post("/mailboxes/{id}/conversations/{cid}/reply", deps.MailboxHandler.HandleReply)
		r.Post("/mailboxes/{id}/conversations/{cid}/close", deps.MailboxHandler.HandleCloseConversation)
	})

	// Public widget API (CORS, rate limited, no CSRF)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CORS)
		r.Use(middleware.RateLimit(deps.Limiter))

		r.Post("/api/v1/messages", deps.APIHandler.HandleSubmitMessage)
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if deps.DB != nil {
			if err := deps.DB.PingContext(r.Context()); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{"status": "error", "detail": "database unreachable"})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return r
}
