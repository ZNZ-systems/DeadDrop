package web

import (
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
	AuthService    *auth.Service
	Renderer       *render.Renderer
	Limiter        *ratelimit.Limiter
	StaticFS       fs.FS
}

// NewRouter wires all routes into a Chi router.
func NewRouter(deps RouterDeps) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)

	// Serve static files
	fileServer := http.FileServer(http.FS(deps.StaticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public auth routes (with CSRF)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)
		r.Use(middleware.OptionalAuth(deps.AuthService))

		r.Get("/login", deps.AuthHandler.ShowLogin)
		r.Post("/login", deps.AuthHandler.HandleLogin)
		r.Get("/signup", deps.AuthHandler.ShowSignup)
		r.Post("/signup", deps.AuthHandler.HandleSignup)
		r.Post("/logout", deps.AuthHandler.HandleLogout)
	})

	// Authenticated dashboard routes (with CSRF + RequireAuth)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CSRF)
		r.Use(middleware.RequireAuth(deps.AuthService))

		r.Get("/", deps.DomainHandler.ShowDashboard)
		r.Get("/domains/new", deps.DomainHandler.ShowNewDomain)
		r.Post("/domains", deps.DomainHandler.HandleCreateDomain)
		r.Get("/domains/{id}", deps.DomainHandler.ShowDomainDetail)
		r.Post("/domains/{id}/verify", deps.DomainHandler.HandleVerifyDomain)
		r.Post("/domains/{id}/delete", deps.DomainHandler.HandleDeleteDomain)

		r.Post("/messages/{messageID}/read", deps.MessageHandler.HandleMarkRead)
		r.Delete("/messages/{messageID}", deps.MessageHandler.HandleDeleteMessage)
	})

	// Public widget API (CORS, rate limited, no CSRF)
	r.Group(func(r chi.Router) {
		r.Use(middleware.CORS)
		r.Use(middleware.RateLimit(deps.Limiter))

		r.Post("/api/v1/messages", deps.APIHandler.HandleSubmitMessage)
	})

	return r
}
