package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

// DomainHandler contains HTTP handlers for domain CRUD operations.
type DomainHandler struct {
	domains  *domain.Service
	messages store.MessageStore
	render   *render.Renderer
}

// NewDomainHandler creates a new DomainHandler.
func NewDomainHandler(domains *domain.Service, messages store.MessageStore, r *render.Renderer) *DomainHandler {
	return &DomainHandler{
		domains:  domains,
		messages: messages,
		render:   r,
	}
}

// ShowDashboard lists all domains for the current user with their unread
// message counts and renders the dashboard page.
func (h *DomainHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	domains, err := h.domains.List(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list domains", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type domainWithUnread struct {
		Domain      interface{}
		UnreadCount int
	}

	items := make([]domainWithUnread, 0, len(domains))
	for _, d := range domains {
		count, err := h.messages.CountUnreadByDomainID(r.Context(), d.ID)
		if err != nil {
			slog.Error("failed to count unread messages", "domain_id", d.ID, "error", err)
			count = 0
		}
		items = append(items, domainWithUnread{Domain: d, UnreadCount: count})
	}

	h.render.Render(w, r, "dashboard.html", map[string]interface{}{
		"User":    user,
		"Domains": items,
	})
}

// ShowNewDomain renders the form for adding a new domain.
func (h *DomainHandler) ShowNewDomain(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	h.render.Render(w, r, "domain_new.html", map[string]interface{}{
		"User": user,
	})
}

// HandleCreateDomain processes the new-domain form submission.
func (h *DomainHandler) HandleCreateDomain(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")

	d, err := h.domains.Create(r.Context(), user.ID, name)
	if err != nil {
		slog.Error("failed to create domain", "error", err)
		h.render.Render(w, r, "domain_new.html", map[string]interface{}{
			"User":  user,
			"Error": err.Error(),
			"Name":  name,
		})
		return
	}

	http.Redirect(w, r, "/domains/"+d.PublicID.String(), http.StatusSeeOther)
}

// ShowDomainDetail renders the detail page for a single domain.
func (h *DomainHandler) ShowDomainDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idParam := chi.URLParam(r, "id")
	publicID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	d, err := h.domains.GetByPublicID(r.Context(), publicID)
	if err != nil {
		slog.Error("failed to get domain", "error", err)
		http.Error(w, "domain not found", http.StatusNotFound)
		return
	}

	messages, err := h.messages.GetMessagesByDomainID(r.Context(), d.ID, 50, 0)
	if err != nil {
		slog.Error("failed to get messages", "error", err)
		messages = nil
	}

	h.render.Render(w, r, "domain_detail.html", map[string]interface{}{
		"User":     user,
		"Domain":   d,
		"Messages": messages,
	})
}

// HandleVerifyDomain initiates DNS verification for the domain.
func (h *DomainHandler) HandleVerifyDomain(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idParam := chi.URLParam(r, "id")
	publicID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	d, err := h.domains.GetByPublicID(r.Context(), publicID)
	if err != nil {
		slog.Error("failed to get domain", "error", err)
		http.Error(w, "domain not found", http.StatusNotFound)
		return
	}

	if err := h.domains.Verify(r.Context(), d); err != nil {
		slog.Warn("domain verification failed", "domain", d.Name, "error", err)
		setFlash(w, "Verification failed: "+err.Error())
	} else {
		setFlash(w, "Domain verified successfully!")
	}

	http.Redirect(w, r, "/domains/"+d.PublicID.String(), http.StatusSeeOther)
}

// HandleDeleteDomain removes the domain and redirects to the dashboard.
func (h *DomainHandler) HandleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idParam := chi.URLParam(r, "id")
	publicID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid domain id", http.StatusBadRequest)
		return
	}

	d, err := h.domains.GetByPublicID(r.Context(), publicID)
	if err != nil {
		slog.Error("failed to get domain", "error", err)
		http.Error(w, "domain not found", http.StatusNotFound)
		return
	}

	if err := h.domains.Delete(r.Context(), d.ID); err != nil {
		slog.Error("failed to delete domain", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
