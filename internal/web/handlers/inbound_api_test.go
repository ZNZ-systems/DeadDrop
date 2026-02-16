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

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/inbound"
	"github.com/znz-systems/deaddrop/internal/models"
)

type inboundTestDomainStore struct {
	byName map[string]*models.Domain
}

func (m *inboundTestDomainStore) CreateDomain(_ context.Context, _ int64, _, _ string) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestDomainStore) GetDomainsByUserID(_ context.Context, _ int64) ([]models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestDomainStore) GetDomainByID(_ context.Context, _ int64) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestDomainStore) GetDomainByPublicID(_ context.Context, _ uuid.UUID) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestDomainStore) GetDomainByName(_ context.Context, name string) (*models.Domain, error) {
	d, ok := m.byName[name]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return d, nil
}
func (m *inboundTestDomainStore) MarkDomainVerified(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}
func (m *inboundTestDomainStore) DeleteDomain(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type inboundTestEmailStore struct {
	count int
}

func (m *inboundTestEmailStore) CreateInboundEmail(_ context.Context, _ models.InboundEmailCreateParams) (*models.InboundEmail, error) {
	m.count++
	return &models.InboundEmail{ID: int64(m.count), PublicID: uuid.New(), CreatedAt: time.Now()}, nil
}
func (m *inboundTestEmailStore) ListInboundEmailsByUserID(_ context.Context, _ int64, _, _ int) ([]models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestEmailStore) SearchInboundEmailsByUserID(_ context.Context, _ int64, _ models.InboundEmailQuery) ([]models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestEmailStore) GetInboundEmailByID(_ context.Context, _ int64) (*models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestEmailStore) CreateInboundEmailRaw(_ context.Context, _ int64, _ string) error {
	return nil
}
func (m *inboundTestEmailStore) CreateInboundEmailAttachment(_ context.Context, _ models.InboundEmailAttachmentCreateParams) (*models.InboundEmailAttachment, error) {
	return nil, nil
}
func (m *inboundTestEmailStore) ListInboundEmailAttachmentsByEmailID(_ context.Context, _ int64) ([]models.InboundEmailAttachment, error) {
	return nil, nil
}
func (m *inboundTestEmailStore) GetInboundEmailAttachmentByID(_ context.Context, _ int64) (*models.InboundEmailAttachment, error) {
	return nil, errors.New("not implemented")
}
func (m *inboundTestEmailStore) MarkInboundEmailRead(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}
func (m *inboundTestEmailStore) DeleteInboundEmail(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type inboundTestDomainConfigStore struct {
	byDomainID map[int64]*models.InboundDomainConfig
}

func (m *inboundTestDomainConfigStore) UpsertInboundDomainConfig(_ context.Context, domainID int64, mxTarget string) (*models.InboundDomainConfig, error) {
	cfg, ok := m.byDomainID[domainID]
	if !ok {
		cfg = &models.InboundDomainConfig{
			DomainID:   domainID,
			MXTarget:   mxTarget,
			MXVerified: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		m.byDomainID[domainID] = cfg
	}
	return cfg, nil
}
func (m *inboundTestDomainConfigStore) GetInboundDomainConfigByDomainID(_ context.Context, domainID int64) (*models.InboundDomainConfig, error) {
	cfg, ok := m.byDomainID[domainID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return cfg, nil
}
func (m *inboundTestDomainConfigStore) UpdateInboundDomainVerification(_ context.Context, domainID int64, verified bool, lastError string) error {
	cfg, ok := m.byDomainID[domainID]
	if !ok {
		return sql.ErrNoRows
	}
	cfg.MXVerified = verified
	cfg.LastError = lastError
	now := time.Now()
	cfg.CheckedAt = &now
	cfg.UpdatedAt = now
	return nil
}

type inboundTestRuleStore struct{}

func (m *inboundTestRuleStore) CreateInboundRecipientRule(_ context.Context, domainID int64, ruleType, pattern, action string) (*models.InboundRecipientRule, error) {
	return &models.InboundRecipientRule{ID: 1, DomainID: domainID, RuleType: ruleType, Pattern: pattern, Action: action, IsActive: true}, nil
}
func (m *inboundTestRuleStore) ListInboundRecipientRulesByDomainID(_ context.Context, _ int64) ([]models.InboundRecipientRule, error) {
	return nil, nil
}
func (m *inboundTestRuleStore) DeleteInboundRecipientRule(_ context.Context, _, _ int64) error {
	return nil
}

func TestInboundAPIHandler_UnauthorizedWithoutToken(t *testing.T) {
	ds := &inboundTestDomainStore{byName: map[string]*models.Domain{}}
	es := &inboundTestEmailStore{}
	cs := &inboundTestDomainConfigStore{byDomainID: map[int64]*models.InboundDomainConfig{}}
	rs := &inboundTestRuleStore{}
	svc := inbound.NewService(ds, es, cs, rs)
	h := NewInboundAPIHandler(svc, "secret", 1024)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestInboundAPIHandler_BadPayload(t *testing.T) {
	ds := &inboundTestDomainStore{byName: map[string]*models.Domain{}}
	es := &inboundTestEmailStore{}
	cs := &inboundTestDomainConfigStore{byDomainID: map[int64]*models.InboundDomainConfig{}}
	rs := &inboundTestRuleStore{}
	svc := inbound.NewService(ds, es, cs, rs)
	h := NewInboundAPIHandler(svc, "secret", 1024)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inbound/emails", bytes.NewBufferString(`{"sender":""}`))
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.HandleReceiveEmail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestInboundAPIHandler_AcceptsKnownDomainRecipient(t *testing.T) {
	ds := &inboundTestDomainStore{
		byName: map[string]*models.Domain{
			"example.com": {
				ID:       11,
				UserID:   7,
				Name:     "example.com",
				Verified: true,
			},
		},
	}
	es := &inboundTestEmailStore{}
	cs := &inboundTestDomainConfigStore{byDomainID: map[int64]*models.InboundDomainConfig{
		11: {
			DomainID:   11,
			MXTarget:   "mx.example.com",
			MXVerified: true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}}
	rs := &inboundTestRuleStore{}
	svc := inbound.NewService(ds, es, cs, rs)
	h := NewInboundAPIHandler(svc, "secret", 1024)

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
	if es.count != 1 {
		t.Fatalf("expected one email insert, got %d", es.count)
	}
}

func TestInboundAPIHandler_AcceptsRawRFC822(t *testing.T) {
	ds := &inboundTestDomainStore{
		byName: map[string]*models.Domain{
			"example.com": {
				ID:       11,
				UserID:   7,
				Name:     "example.com",
				Verified: true,
			},
		},
	}
	es := &inboundTestEmailStore{}
	cs := &inboundTestDomainConfigStore{byDomainID: map[int64]*models.InboundDomainConfig{
		11: {
			DomainID:   11,
			MXTarget:   "mx.example.com",
			MXVerified: true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}}
	rs := &inboundTestRuleStore{}
	svc := inbound.NewService(ds, es, cs, rs)
	h := NewInboundAPIHandler(svc, "secret", 8*1024)

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
	if es.count != 1 {
		t.Fatalf("expected one email insert, got %d", es.count)
	}
}
