package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/znz-systems/deaddrop/internal/store"
)

const defaultInboundAPIMaxBodyBytes int64 = 1024 * 1024

type InboundAPIHandler struct {
	jobs         store.InboundIngestJobStore
	apiToken     string
	maxBodyBytes int64
	maxAttempts  int
}

func NewInboundAPIHandler(jobs store.InboundIngestJobStore, apiToken string, maxBodyBytes int64, maxAttempts int) *InboundAPIHandler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultInboundAPIMaxBodyBytes
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return &InboundAPIHandler{
		jobs:         jobs,
		apiToken:     strings.TrimSpace(apiToken),
		maxBodyBytes: maxBodyBytes,
		maxAttempts:  maxAttempts,
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
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, jsonResponse{Error: "payload too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "invalid JSON payload"})
		return
	}

	sender, _ := payload["sender"].(string)
	rawRFC822, _ := payload["raw_rfc822"].(string)
	if strings.TrimSpace(rawRFC822) == "" && strings.TrimSpace(sender) == "" {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "sender is required when raw_rfc822 is missing"})
		return
	}

	recipientsUsable := false
	if recipientsRaw, ok := payload["recipients"].([]interface{}); ok {
		for _, entry := range recipientsRaw {
			if s, ok := entry.(string); ok && strings.TrimSpace(s) != "" {
				recipientsUsable = true
				break
			}
		}
	}
	if strings.TrimSpace(rawRFC822) == "" && !recipientsUsable {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "at least one recipient is required when raw_rfc822 is missing"})
		return
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, jsonResponse{Error: "invalid JSON payload"})
		return
	}

	job, err := h.jobs.EnqueueInboundIngestJob(r.Context(), encoded, h.maxAttempts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, jsonResponse{Error: "failed to enqueue inbound message"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"job_id": job.ID,
		"status": job.Status,
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
