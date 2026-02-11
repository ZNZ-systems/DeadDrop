package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/message"
)

// APIHandler serves the public widget API.
type APIHandler struct {
	messages *message.Service
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(messages *message.Service) *APIHandler {
	return &APIHandler{
		messages: messages,
	}
}

// HandleSubmitMessage accepts a contact-form submission from the public widget.
//
// Expected form fields:
//
//	domain_id  (required, UUID)
//	name       (optional)
//	email      (optional)
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

	// --- Validate domain_id ---
	domainIDRaw := r.FormValue("domain_id")
	if domainIDRaw == "" {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain_id is required"})
		return
	}

	domainPublicID, err := uuid.Parse(domainIDRaw)
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

	// --- Submit via service ---
	if err := h.messages.Submit(r.Context(), domainPublicID, name, email, body); err != nil {
		switch {
		case errors.Is(err, message.ErrDomainNotFound):
			writeJSON(w, http.StatusNotFound, jsonResponse{Error: "domain not found"})
		case errors.Is(err, message.ErrDomainNotVerified):
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain not verified"})
		default:
			slog.Error("failed to submit message", "domain_public_id", domainPublicID, "error", err)
			writeJSON(w, http.StatusInternalServerError, jsonResponse{Error: "internal server error"})
		}
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
