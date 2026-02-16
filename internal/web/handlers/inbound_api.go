package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/znz-systems/deaddrop/internal/inbound"
)

const defaultInboundAPIMaxBodyBytes int64 = 1024 * 1024

type InboundAPIHandler struct {
	service            *inbound.Service
	apiToken           string
	maxBodyBytes       int64
	maxAttachmentBytes int64
}

func NewInboundAPIHandler(service *inbound.Service, apiToken string, maxBodyBytes int64) *InboundAPIHandler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultInboundAPIMaxBodyBytes
	}
	return &InboundAPIHandler{
		service:            service,
		apiToken:           strings.TrimSpace(apiToken),
		maxBodyBytes:       maxBodyBytes,
		maxAttachmentBytes: 5 * 1024 * 1024,
	}
}

func (h *InboundAPIHandler) HandleReceiveEmail(w http.ResponseWriter, r *http.Request) {
	if h.apiToken == "" {
		writeJSON(w, http.StatusServiceUnavailable, jsonResponse{Error: "inbound api is not configured"})
		return
	}
	if !validBearerToken(r.Header.Get("Authorization"), h.apiToken) {
		writeJSON(w, http.StatusUnauthorized, jsonResponse{Error: "unauthorized"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBodyBytes)
	var payload struct {
		Sender     string   `json:"sender"`
		Recipients []string `json:"recipients"`
		Subject    string   `json:"subject"`
		TextBody   string   `json:"text_body"`
		HTMLBody   string   `json:"html_body"`
		MessageID  string   `json:"message_id"`
		RawRFC822  string   `json:"raw_rfc822"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, jsonResponse{Error: "payload too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "invalid JSON payload"})
		return
	}

	msg := inbound.Message{
		Sender:     payload.Sender,
		Recipients: payload.Recipients,
		Subject:    payload.Subject,
		TextBody:   payload.TextBody,
		HTMLBody:   payload.HTMLBody,
		MessageID:  payload.MessageID,
		RawRFC822:  payload.RawRFC822,
	}
	if strings.TrimSpace(payload.RawRFC822) != "" {
		parsed, parseErr := inbound.ParseRFC822(payload.RawRFC822, h.maxAttachmentBytes)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "invalid raw_rfc822: " + parseErr.Error()})
			return
		}
		if strings.TrimSpace(msg.Sender) == "" {
			msg.Sender = parsed.Sender
		}
		if len(msg.Recipients) == 0 {
			msg.Recipients = parsed.Recipients
		}
		if strings.TrimSpace(msg.Subject) == "" {
			msg.Subject = parsed.Subject
		}
		if strings.TrimSpace(msg.TextBody) == "" {
			msg.TextBody = parsed.TextBody
		}
		if strings.TrimSpace(msg.HTMLBody) == "" {
			msg.HTMLBody = parsed.HTMLBody
		}
		if strings.TrimSpace(msg.MessageID) == "" {
			msg.MessageID = parsed.MessageID
		}
		msg.Attachments = parsed.Attachments
	}

	result, err := h.service.Ingest(r.Context(), msg)
	if err != nil {
		switch {
		case errors.Is(err, inbound.ErrSenderRequired), errors.Is(err, inbound.ErrRecipientsRequired):
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: err.Error()})
		default:
			writeJSON(w, http.StatusBadRequest, jsonResponse{Error: err.Error()})
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"accepted": result.Accepted,
		"dropped":  result.Dropped,
	})
}

func validBearerToken(headerValue, expected string) bool {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(headerValue, prefix) {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(headerValue, prefix))
	return token != "" && token == expected
}
