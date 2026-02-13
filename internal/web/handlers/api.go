package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/store"
)

// APIHandler serves the public widget API.
type APIHandler struct {
	streams       store.StreamStore
	conversations *conversation.Service
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(streams store.StreamStore, conversations *conversation.Service) *APIHandler {
	return &APIHandler{
		streams:       streams,
		conversations: conversations,
	}
}

// HandleSubmitMessage accepts a contact-form submission from the public widget.
//
// Expected form fields:
//
//	domain_id  (required, UUID — maps to stream widget_id)
//	name       (optional)
//	email      (optional)
//	subject    (optional, defaults to "New message")
//	message    (required)
//	_gotcha    (honeypot -- if filled in, silently accept)
func (h *APIHandler) HandleSubmitMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "invalid form data"})
		return
	}

	// Honeypot: if the hidden field is filled, silently accept.
	if r.FormValue("_gotcha") != "" {
		writeJSON(w, http.StatusOK, jsonResponse{OK: true})
		return
	}

	// --- Validate domain_id (widget_id) ---
	domainIDRaw := r.FormValue("domain_id")
	if domainIDRaw == "" {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain_id is required"})
		return
	}

	widgetID, err := uuid.Parse(domainIDRaw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain_id must be a valid UUID"})
		return
	}

	// --- Validate message body ---
	body := r.FormValue("message")
	if body == "" {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "message is required"})
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	subject := r.FormValue("subject")
	if subject == "" {
		subject = "New message"
	}

	// --- Look up stream by widget ID ---
	stream, err := h.streams.GetStreamByWidgetID(r.Context(), widgetID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, jsonResponse{Error: "domain not found"})
		return
	}

	// --- Start conversation via stream ---
	if _, err := h.conversations.StartConversation(r.Context(), stream, subject, email, name, body); err != nil {
		if errors.Is(err, conversation.ErrStreamDisabled) {
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain not verified"})
			return
		}
		slog.Error("failed to start conversation", "widget_id", widgetID, "error", err)
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, jsonResponse{OK: true})
}

// jsonResponse is the envelope for all API JSON responses.
type jsonResponse struct {
	OK    bool   `json:"ok,omitempty"`
	Error string `json:"error,omitempty"`
}

// writeJSON serialises v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}
