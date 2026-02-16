package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
)

type inboundTestJobStore struct {
	jobs       []models.InboundIngestJob
	enqueueErr error
}

func (m *inboundTestJobStore) EnqueueInboundIngestJob(_ context.Context, payload []byte, maxAttempts int) (*models.InboundIngestJob, error) {
	if m.enqueueErr != nil {
		return nil, m.enqueueErr
	}
	id := int64(len(m.jobs) + 1)
	job := models.InboundIngestJob{
		ID:          id,
		Status:      "queued",
		Payload:     payload,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		AvailableAt: time.Now(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.jobs = append(m.jobs, job)
	return &job, nil
}

func (m *inboundTestJobStore) ClaimNextInboundIngestJob(_ context.Context) (*models.InboundIngestJob, error) {
	return nil, sql.ErrNoRows
}
func (m *inboundTestJobStore) MarkInboundIngestJobDone(_ context.Context, _ int64, _, _ int) error {
	return nil
}
func (m *inboundTestJobStore) MarkInboundIngestJobRetry(_ context.Context, _ int64, _ time.Time, _ string) error {
	return nil
}
func (m *inboundTestJobStore) MarkInboundIngestJobFailed(_ context.Context, _ int64, _ string) error {
	return nil
}

func TestInboundAPIHandler_UnauthorizedWithoutToken(t *testing.T) {
	jobs := &inboundTestJobStore{}
	h := NewInboundAPIHandler(jobs, "secret", 1024, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestInboundAPIHandler_BadPayload(t *testing.T) {
	jobs := &inboundTestJobStore{}
	h := NewInboundAPIHandler(jobs, "secret", 1024, 5)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBufferString(`{"sender":""}`))
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInboundAPIHandler_AcceptsSenderRecipientsPayload(t *testing.T) {
	jobs := &inboundTestJobStore{}
	h := NewInboundAPIHandler(jobs, "secret", 1024, 5)

	body, _ := json.Marshal(map[string]interface{}{
		"sender":     "sender@outside.com",
		"recipients": []string{"ideas@example.com"},
		"subject":    "Test",
		"text_body":  "Hello",
		"message_id": "abc-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	if len(jobs.jobs) != 1 {
		t.Fatalf("expected one queued job, got %d", len(jobs.jobs))
	}
}

func TestInboundAPIHandler_AcceptsRawRFC822(t *testing.T) {
	jobs := &inboundTestJobStore{}
	h := NewInboundAPIHandler(jobs, "secret", 8*1024, 5)

	raw := "From: Sender <sender@outside.com>\r\nTo: ideas@example.com\r\nSubject: Parsed Subject\r\nMessage-ID: <m1@test>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello from raw"
	body, _ := json.Marshal(map[string]interface{}{
		"raw_rfc822": raw,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	if len(jobs.jobs) != 1 {
		t.Fatalf("expected one queued job, got %d", len(jobs.jobs))
	}
}

func TestInboundAPIHandler_EnqueueFailure(t *testing.T) {
	jobs := &inboundTestJobStore{enqueueErr: errors.New("db down")}
	h := NewInboundAPIHandler(jobs, "secret", 1024, 5)

	body, _ := json.Marshal(map[string]interface{}{
		"sender":     "sender@outside.com",
		"recipients": []string{"ideas@example.com"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
