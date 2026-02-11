package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/message"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores for message.Service ---

type mockMessageStore struct {
	messages map[int64]*models.Message
	nextID   int64
}

func newMockMessageStore() *mockMessageStore {
	return &mockMessageStore{
		messages: make(map[int64]*models.Message),
		nextID:   1,
	}
}

func (m *mockMessageStore) CreateMessage(_ context.Context, domainID int64, senderName, senderEmail, body string) (*models.Message, error) {
	msg := &models.Message{
		ID:          m.nextID,
		PublicID:    uuid.New(),
		DomainID:    domainID,
		SenderName:  senderName,
		SenderEmail: senderEmail,
		Body:        body,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	m.nextID++
	m.messages[msg.ID] = msg
	return msg, nil
}

func (m *mockMessageStore) GetMessagesByDomainID(_ context.Context, _ int64, _, _ int) ([]models.Message, error) {
	return nil, nil
}

func (m *mockMessageStore) MarkMessageRead(_ context.Context, _ int64) error { return nil }
func (m *mockMessageStore) DeleteMessage(_ context.Context, _ int64) error   { return nil }
func (m *mockMessageStore) CountUnreadByDomainID(_ context.Context, _ int64) (int, error) {
	return 0, nil
}

type mockDomainStore struct {
	domains map[uuid.UUID]*models.Domain
}

func newMockDomainStore() *mockDomainStore {
	return &mockDomainStore{domains: make(map[uuid.UUID]*models.Domain)}
}

func (m *mockDomainStore) addDomain(d *models.Domain) {
	m.domains[d.PublicID] = d
}

func (m *mockDomainStore) CreateDomain(_ context.Context, _ int64, _, _ string) (*models.Domain, error) {
	return nil, nil
}
func (m *mockDomainStore) GetDomainsByUserID(_ context.Context, _ int64) ([]models.Domain, error) {
	return nil, nil
}
func (m *mockDomainStore) GetDomainByPublicID(_ context.Context, publicID uuid.UUID) (*models.Domain, error) {
	d, ok := m.domains[publicID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return d, nil
}
func (m *mockDomainStore) GetDomainByName(_ context.Context, _ string) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *mockDomainStore) MarkDomainVerified(_ context.Context, _ int64) error { return nil }
func (m *mockDomainStore) DeleteDomain(_ context.Context, _ int64) error       { return nil }

type mockNotifier struct {
	called atomic.Int32
}

func (m *mockNotifier) NotifyNewMessage(_ context.Context, _ *models.Domain, _ *models.Message) error {
	m.called.Add(1)
	return nil
}

// --- Test helpers ---

func makeTestService(verifiedDomainID uuid.UUID) (*message.Service, *mockDomainStore) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &mockNotifier{}

	if verifiedDomainID != uuid.Nil {
		ds.addDomain(&models.Domain{
			ID:                1,
			PublicID:          verifiedDomainID,
			UserID:            1,
			Name:              "example.com",
			VerificationToken: "tok",
			Verified:          true,
		})
	}

	svc := message.NewService(ms, ds, notifier)
	return svc, ds
}

func postForm(handler http.HandlerFunc, values url.Values) *httptest.ResponseRecorder {
	body := values.Encode()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func parseJSONResponse(t *testing.T, rr *httptest.ResponseRecorder) jsonResponse {
	t.Helper()
	var resp jsonResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	return resp
}

// --- Tests ---

func TestHandleSubmitMessage_Success(t *testing.T) {
	domainID := uuid.New()
	svc, _ := makeTestService(domainID)
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {domainID.String()},
		"name":      {"John Doe"},
		"email":     {"john@example.com"},
		"message":   {"Hello, world!"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	resp := parseJSONResponse(t, rr)
	if !resp.OK {
		t.Error("expected ok: true")
	}
}

func TestHandleSubmitMessage_MissingDomainID(t *testing.T) {
	svc, _ := makeTestService(uuid.Nil)
	handler := NewAPIHandler(svc)

	values := url.Values{
		"message": {"Hello"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if resp.Error != "domain_id is required" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleSubmitMessage_InvalidUUID(t *testing.T) {
	svc, _ := makeTestService(uuid.Nil)
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {"not-a-uuid"},
		"message":   {"Hello"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if resp.Error != "domain_id must be a valid UUID" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleSubmitMessage_MissingMessage(t *testing.T) {
	domainID := uuid.New()
	svc, _ := makeTestService(domainID)
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {domainID.String()},
		"name":      {"John"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if resp.Error != "message is required" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleSubmitMessage_HoneypotFilled(t *testing.T) {
	domainID := uuid.New()
	svc, _ := makeTestService(domainID)
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {domainID.String()},
		"message":   {"Hello"},
		"_gotcha":   {"bot filled this in"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 (silent accept), got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if !resp.OK {
		t.Error("honeypot submissions should silently succeed")
	}
}

func TestHandleSubmitMessage_DomainNotFound(t *testing.T) {
	svc, _ := makeTestService(uuid.Nil) // no domains
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {uuid.New().String()},
		"message":   {"Hello"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleSubmitMessage_DomainNotVerified(t *testing.T) {
	domainID := uuid.New()
	svc, ds := makeTestService(uuid.Nil)
	// Add unverified domain
	ds.addDomain(&models.Domain{
		ID:       2,
		PublicID: domainID,
		UserID:   1,
		Name:     "unverified.com",
		Verified: false,
	})
	handler := NewAPIHandler(svc)

	values := url.Values{
		"domain_id": {domainID.String()},
		"message":   {"Hello"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if resp.Error != "domain not verified" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}
