package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/mailbox"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

type MailboxHandler struct {
	mailboxes     *mailbox.Service
	conversations *conversation.Service
	domains       *domain.Service
	streams       store.StreamStore
	convStore     store.ConversationStore
	render        *render.Renderer
	baseURL       string
	secureCookies bool
}

func NewMailboxHandler(
	mailboxes *mailbox.Service,
	conversations *conversation.Service,
	domains *domain.Service,
	streams store.StreamStore,
	convStore store.ConversationStore,
	r *render.Renderer,
	baseURL string,
	secureCookies bool,
) *MailboxHandler {
	return &MailboxHandler{
		mailboxes:     mailboxes,
		conversations: conversations,
		domains:       domains,
		streams:       streams,
		convStore:     convStore,
		render:        r,
		baseURL:       baseURL,
		secureCookies: secureCookies,
	}
}

func (h *MailboxHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mailboxes, err := h.mailboxes.List(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list mailboxes", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type mailboxWithCount struct {
		Mailbox   interface{}
		OpenCount int
	}

	items := make([]mailboxWithCount, 0, len(mailboxes))
	for _, mb := range mailboxes {
		count, err := h.conversations.CountOpen(r.Context(), mb.ID)
		if err != nil {
			count = 0
		}
		items = append(items, mailboxWithCount{Mailbox: mb, OpenCount: count})
	}

	h.render.Render(w, r, "mailbox_dashboard.html", map[string]interface{}{
		"User":      user,
		"Mailboxes": items,
	})
}

func (h *MailboxHandler) ShowNewMailbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	domains, _ := h.domains.List(r.Context(), user.ID)
	selectedDomainID := strings.TrimSpace(r.URL.Query().Get("domain_id"))
	fromAddress := strings.TrimSpace(r.URL.Query().Get("from_address"))
	name := strings.TrimSpace(r.URL.Query().Get("name"))

	h.render.Render(w, r, "mailbox_new.html", map[string]interface{}{
		"User":             user,
		"Domains":          domains,
		"SelectedDomainID": selectedDomainID,
		"FromAddress":      fromAddress,
		"Name":             name,
	})
}

func (h *MailboxHandler) HandleCreateMailbox(w http.ResponseWriter, r *http.Request) {
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
	fromAddress := r.FormValue("from_address")
	domainIDStr := r.FormValue("domain_id")

	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.Create(r.Context(), user.ID, domainID, name, fromAddress)
	if err != nil {
		slog.Error("failed to create mailbox", "error", err)
		domains, _ := h.domains.List(r.Context(), user.ID)
		h.render.Render(w, r, "mailbox_new.html", map[string]interface{}{
			"User":    user,
			"Domains": domains,
			"Error":   err.Error(),
			"Name":    name,
		})
		return
	}

	// Bootstrap both channels by default:
	// - form stream: widget submissions
	// - email stream: inbound SMTP to the mailbox address
	if _, err := h.streams.CreateStream(r.Context(), mb.ID, "form", "", uuid.New()); err != nil {
		slog.Warn("failed to bootstrap form stream", "mailbox_id", mb.ID, "error", err)
		setFlash(w, "Mailbox created, but default form stream failed to create. Add it manually in mailbox settings.", h.secureCookies)
		http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
		return
	}
	if _, err := h.streams.CreateStream(r.Context(), mb.ID, "email", strings.ToLower(fromAddress), uuid.Nil); err != nil {
		slog.Warn("failed to bootstrap email stream", "mailbox_id", mb.ID, "error", err)
		setFlash(w, "Mailbox created and form stream ready. Add your email stream manually in mailbox settings.", h.secureCookies)
		http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
		return
	}

	setFlash(w, "Mailbox created with default form and email streams.", h.secureCookies)

	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) ShowMailboxDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid mailbox id", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convos, _ := h.conversations.List(r.Context(), mb.ID, 50, 0)
	streams, _ := h.streams.GetStreamsByMailboxID(r.Context(), mb.ID)

	h.render.Render(w, r, "mailbox_detail.html", map[string]interface{}{
		"User":          user,
		"Mailbox":       mb,
		"Conversations": convos,
		"Streams":       streams,
		"BaseURL":       h.baseURL,
	})
}

func (h *MailboxHandler) ShowConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid mailbox id", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, err := uuid.Parse(chi.URLParam(r, "cid"))
	if err != nil {
		http.Error(w, "invalid conversation id", http.StatusBadRequest)
		return
	}

	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	messages, _ := h.conversations.GetMessages(r.Context(), conv.ID)

	h.render.Render(w, r, "conversation_detail.html", map[string]interface{}{
		"User":         user,
		"Mailbox":      mb,
		"Conversation": conv,
		"Messages":     messages,
	})
}

func (h *MailboxHandler) HandleReply(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, _ := uuid.Parse(chi.URLParam(r, "cid"))
	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body := r.FormValue("body")
	if body == "" {
		setFlash(w, "Reply body cannot be empty", h.secureCookies)
		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	if _, err := h.conversations.Reply(r.Context(), conv.ID, body); err != nil {
		slog.Error("failed to send reply", "error", err)
		setFlash(w, "Failed to send reply: "+err.Error(), h.secureCookies)
	} else {
		setFlash(w, "Reply sent!", h.secureCookies)
	}

	http.Redirect(w, r, fmt.Sprintf("/mailboxes/%s/conversations/%s", mb.PublicID, conv.PublicID), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleCloseConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, _ := uuid.Parse(chi.URLParam(r, "cid"))
	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	_ = h.conversations.Close(r.Context(), conv.ID)
	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleDeleteMailbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	_ = h.mailboxes.Delete(r.Context(), mb.ID)
	http.Redirect(w, r, "/mailboxes", http.StatusSeeOther)
}

func (h *MailboxHandler) HandleAddStream(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	streamType := r.FormValue("type")
	address := r.FormValue("address")

	var widgetID uuid.UUID
	if streamType == "form" {
		widgetID = uuid.New()
	}

	if _, err := h.streams.CreateStream(r.Context(), mb.ID, streamType, address, widgetID); err != nil {
		slog.Error("failed to create stream", "error", err)
		setFlash(w, "Failed to create stream: "+err.Error(), h.secureCookies)
	}

	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleDeleteStream(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	sidStr := chi.URLParam(r, "sid")
	sid, err := strconv.ParseInt(sidStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	_ = h.streams.DeleteStream(r.Context(), sid)
	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}
