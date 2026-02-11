package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/znz-systems/deaddrop/internal/message"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

// MessageHandler serves dashboard views for messages.
type MessageHandler struct {
	messages *message.Service
	msgStore store.MessageStore
	domains  store.DomainStore
	render   *render.Renderer
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(messages *message.Service, msgStore store.MessageStore, domains store.DomainStore, r *render.Renderer) *MessageHandler {
	return &MessageHandler{
		messages: messages,
		msgStore: msgStore,
		domains:  domains,
		render:   r,
	}
}

// HandleMarkRead marks a message as read.
func (h *MessageHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "messageID")
	messageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("invalid message ID", "raw", idStr, "error", err)
		http.Error(w, "invalid message ID", http.StatusBadRequest)
		return
	}

	msg, err := h.msgStore.GetMessageByID(r.Context(), messageID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	domain, err := h.domains.GetDomainByID(r.Context(), msg.DomainID)
	if err != nil || domain.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.messages.MarkRead(r.Context(), messageID); err != nil {
		slog.Error("failed to mark message read", "message_id", messageID, "error", err)
		http.Error(w, "failed to mark message read", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}

// HandleDeleteMessage deletes a message.
func (h *MessageHandler) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "messageID")
	messageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Error("invalid message ID", "raw", idStr, "error", err)
		http.Error(w, "invalid message ID", http.StatusBadRequest)
		return
	}

	msg, err := h.msgStore.GetMessageByID(r.Context(), messageID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	domain, err := h.domains.GetDomainByID(r.Context(), msg.DomainID)
	if err != nil || domain.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.messages.Delete(r.Context(), messageID); err != nil {
		slog.Error("failed to delete message", "message_id", messageID, "error", err)
		http.Error(w, "failed to delete message", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	http.Redirect(w, r, referer, http.StatusSeeOther)
}
