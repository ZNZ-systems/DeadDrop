package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/inbound"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

type InboundDomainHandler struct {
	domains       *domain.Service
	inbound       *inbound.DomainService
	rules         store.InboundRecipientRuleStore
	render        *render.Renderer
	secureCookies bool
}

func NewInboundDomainHandler(domains *domain.Service, inboundSvc *inbound.DomainService, rules store.InboundRecipientRuleStore, r *render.Renderer, secureCookies bool) *InboundDomainHandler {
	return &InboundDomainHandler{
		domains:       domains,
		inbound:       inboundSvc,
		rules:         rules,
		render:        r,
		secureCookies: secureCookies,
	}
}

func (h *InboundDomainHandler) ShowInboundSetup(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	d, ok := h.loadOwnedDomain(w, r, user.ID)
	if !ok {
		return
	}

	cfg, err := h.inbound.EnsureConfig(r.Context(), d.ID)
	if err != nil {
		slog.Error("failed to ensure inbound config", "domain_id", d.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	rules, err := h.rules.ListInboundRecipientRulesByDomainID(r.Context(), d.ID)
	if err != nil {
		slog.Error("failed to list inbound rules", "domain_id", d.ID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"User":          user,
		"Domain":        d,
		"InboundConfig": cfg,
		"Rules":         rules,
	}
	if msg, msgType := consumeFlash(w, r, h.secureCookies); msg != "" {
		data["Flash"] = msg
		data["FlashType"] = msgType
	}
	h.render.Render(w, r, "inbound_setup.html", data)
}

func (h *InboundDomainHandler) HandleVerifyInbound(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	d, ok := h.loadOwnedDomain(w, r, user.ID)
	if !ok {
		return
	}

	verified, err := h.inbound.VerifyMX(r.Context(), d)
	if err != nil {
		slog.Warn("inbound mx verification failed", "domain", d.Name, "error", err)
		setFlashError(w, "MX verification failed: "+err.Error(), h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}
	if verified {
		setFlashSuccess(w, "Inbound MX verified. Catch-all is active for this domain.", h.secureCookies)
	} else {
		setFlashError(w, "MX record not found yet. Update DNS and try again.", h.secureCookies)
	}
	http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
}

func (h *InboundDomainHandler) HandleCreateRule(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	d, ok := h.loadOwnedDomain(w, r, user.ID)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		setFlashError(w, "Invalid rule form.", h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}

	ruleType := strings.ToLower(strings.TrimSpace(r.FormValue("rule_type")))
	pattern := strings.ToLower(strings.TrimSpace(r.FormValue("pattern")))
	action := strings.ToLower(strings.TrimSpace(r.FormValue("action")))
	if ruleType != "exact" && ruleType != "wildcard" {
		setFlashError(w, "Rule type must be exact or wildcard.", h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}
	if pattern == "" {
		setFlashError(w, "Pattern is required.", h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}
	if action == "" {
		action = "inbox"
	}
	if action != "inbox" && action != "drop" {
		setFlashError(w, "Action must be inbox or drop.", h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}

	if _, err := h.rules.CreateInboundRecipientRule(r.Context(), d.ID, ruleType, pattern, action); err != nil {
		setFlashError(w, "Failed to create rule: "+err.Error(), h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}
	setFlashSuccess(w, "Rule created.", h.secureCookies)
	http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
}

func (h *InboundDomainHandler) HandleDeleteRule(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	d, ok := h.loadOwnedDomain(w, r, user.ID)
	if !ok {
		return
	}

	ruleIDRaw := chi.URLParam(r, "ruleID")
	ruleID, err := strconv.ParseInt(ruleIDRaw, 10, 64)
	if err != nil || ruleID <= 0 {
		http.Error(w, "invalid rule id", http.StatusBadRequest)
		return
	}
	if err := h.rules.DeleteInboundRecipientRule(r.Context(), d.ID, ruleID); err != nil {
		setFlashError(w, "Failed to delete rule.", h.secureCookies)
		http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
		return
	}
	setFlashSuccess(w, "Rule deleted.", h.secureCookies)
	http.Redirect(w, r, "/domains/"+d.PublicID.String()+"/inbound", http.StatusSeeOther)
}

func (h *InboundDomainHandler) loadOwnedDomain(w http.ResponseWriter, r *http.Request, userID int64) (*models.Domain, bool) {
	idParam := chi.URLParam(r, "id")
	publicID, err := uuid.Parse(idParam)
	if err != nil {
		http.Error(w, "invalid domain id", http.StatusBadRequest)
		return nil, false
	}

	d, err := h.domains.GetByPublicID(r.Context(), publicID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	if d.UserID != userID {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	return d, true
}
