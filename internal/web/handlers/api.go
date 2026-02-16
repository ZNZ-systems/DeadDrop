package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/message"
	"github.com/znz-systems/deaddrop/internal/store"
)

const (
	defaultAPIMaxBodyBytes int64 = 16 * 1024
	maxSenderNameLen             = 120
	maxSenderEmailLen            = 254
	maxMessageBodyLen            = 5000
)

// APIHandler serves the public widget API.
type APIHandler struct {
	messages     *message.Service
	domains      store.DomainStore
	maxBodyBytes int64
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(messages *message.Service, domains store.DomainStore, maxBodyBytes int64) *APIHandler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultAPIMaxBodyBytes
	}
	return &APIHandler{
		messages:     messages,
		domains:      domains,
		maxBodyBytes: maxBodyBytes,
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
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	if err := r.ParseForm(); err != nil {
		if isBodyTooLargeError(err) {
			writeJSON(w, http.StatusRequestEntityTooLarge, jsonResponse{Error: "payload too large"})
			return
		}
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

	domain, err := h.domains.GetDomainByPublicID(r.Context(), domainPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, jsonResponse{Error: "domain not found"})
			return
		}
		slog.Error("failed to fetch domain for submission", "domain_public_id", domainPublicID, "error", err)
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Error: "internal server error"})
		return
	}
	if !isOriginAllowedForDomain(r.Header.Get("Origin"), domain.Name) {
		writeJSON(w, http.StatusForbidden, jsonResponse{Error: "origin not allowed for domain"})
		return
	}
	if !domain.Verified {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "domain not verified"})
		return
	}

	// --- Validate message body ---
	body := strings.TrimSpace(r.FormValue("message"))
	if body == "" {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "message is required"})
		return
	}
	if len(body) > maxMessageBodyLen {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "message is too long"})
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if len(name) > maxSenderNameLen {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "name is too long"})
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if len(email) > maxSenderEmailLen {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "email is too long"})
		return
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "email must be valid"})
			return
		}
	}

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

func isBodyTooLargeError(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

func isOriginAllowedForDomain(origin, domainName string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := normalizeHost(u.Hostname())
	domain := normalizeHost(domainName)
	if host == "" || domain == "" {
		return false
	}

	return host == domain || strings.HasSuffix(host, "."+domain)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")
	return host
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
