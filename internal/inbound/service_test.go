package inbound

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type mockDomainStore struct {
	byName map[string]*models.Domain
}

func newMockDomainStore() *mockDomainStore {
	return &mockDomainStore{byName: map[string]*models.Domain{}}
}

func (m *mockDomainStore) CreateDomain(_ context.Context, _ int64, _, _ string) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *mockDomainStore) GetDomainsByUserID(_ context.Context, _ int64) ([]models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *mockDomainStore) GetDomainByID(_ context.Context, _ int64) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *mockDomainStore) GetDomainByPublicID(_ context.Context, _ uuid.UUID) (*models.Domain, error) {
	return nil, errors.New("not implemented")
}
func (m *mockDomainStore) GetDomainByName(_ context.Context, name string) (*models.Domain, error) {
	d, ok := m.byName[name]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return d, nil
}
func (m *mockDomainStore) MarkDomainVerified(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}
func (m *mockDomainStore) DeleteDomain(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type mockInboundEmailStore struct {
	items []models.InboundEmailCreateParams
}

func (m *mockInboundEmailStore) CreateInboundEmail(_ context.Context, params models.InboundEmailCreateParams) (*models.InboundEmail, error) {
	m.items = append(m.items, params)
	return &models.InboundEmail{
		ID:        int64(len(m.items)),
		PublicID:  uuid.New(),
		UserID:    params.UserID,
		DomainID:  params.DomainID,
		Recipient: params.Recipient,
		Sender:    params.Sender,
		Subject:   params.Subject,
		TextBody:  params.TextBody,
		HTMLBody:  params.HTMLBody,
		MessageID: params.MessageID,
		IsRead:    false,
		CreatedAt: time.Now(),
	}, nil
}
func (m *mockInboundEmailStore) ListInboundEmailsByUserID(_ context.Context, _ int64, _, _ int) ([]models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *mockInboundEmailStore) SearchInboundEmailsByUserID(_ context.Context, _ int64, _ models.InboundEmailQuery) ([]models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *mockInboundEmailStore) GetInboundEmailByID(_ context.Context, _ int64) (*models.InboundEmail, error) {
	return nil, errors.New("not implemented")
}
func (m *mockInboundEmailStore) CreateInboundEmailRaw(_ context.Context, _ models.InboundEmailRawCreateParams) error {
	return nil
}
func (m *mockInboundEmailStore) CreateInboundEmailAttachment(_ context.Context, _ models.InboundEmailAttachmentCreateParams) (*models.InboundEmailAttachment, error) {
	return nil, nil
}
func (m *mockInboundEmailStore) ListInboundEmailAttachmentsByEmailID(_ context.Context, _ int64) ([]models.InboundEmailAttachment, error) {
	return nil, nil
}
func (m *mockInboundEmailStore) GetInboundEmailAttachmentByID(_ context.Context, _ int64) (*models.InboundEmailAttachment, error) {
	return nil, errors.New("not implemented")
}
func (m *mockInboundEmailStore) MarkInboundEmailRead(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}
func (m *mockInboundEmailStore) DeleteInboundEmail(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type mockInboundDomainConfigStore struct {
	byDomainID map[int64]*models.InboundDomainConfig
}

func newMockInboundDomainConfigStore() *mockInboundDomainConfigStore {
	return &mockInboundDomainConfigStore{byDomainID: map[int64]*models.InboundDomainConfig{}}
}

func (m *mockInboundDomainConfigStore) UpsertInboundDomainConfig(_ context.Context, domainID int64, mxTarget string) (*models.InboundDomainConfig, error) {
	cfg, ok := m.byDomainID[domainID]
	if !ok {
		now := time.Now()
		cfg = &models.InboundDomainConfig{
			DomainID:   domainID,
			MXTarget:   mxTarget,
			MXVerified: false,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		m.byDomainID[domainID] = cfg
	}
	return cfg, nil
}

func (m *mockInboundDomainConfigStore) GetInboundDomainConfigByDomainID(_ context.Context, domainID int64) (*models.InboundDomainConfig, error) {
	cfg, ok := m.byDomainID[domainID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return cfg, nil
}

func (m *mockInboundDomainConfigStore) UpdateInboundDomainVerification(_ context.Context, domainID int64, verified bool, lastError string) error {
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

type mockInboundRuleStore struct {
	byDomain map[int64][]models.InboundRecipientRule
}

func newMockInboundRuleStore() *mockInboundRuleStore {
	return &mockInboundRuleStore{byDomain: map[int64][]models.InboundRecipientRule{}}
}

func (m *mockInboundRuleStore) CreateInboundRecipientRule(_ context.Context, domainID int64, ruleType, pattern, action string) (*models.InboundRecipientRule, error) {
	r := models.InboundRecipientRule{
		ID:       int64(len(m.byDomain[domainID]) + 1),
		DomainID: domainID,
		RuleType: ruleType,
		Pattern:  pattern,
		Action:   action,
		IsActive: true,
	}
	m.byDomain[domainID] = append(m.byDomain[domainID], r)
	return &r, nil
}

func (m *mockInboundRuleStore) ListInboundRecipientRulesByDomainID(_ context.Context, domainID int64) ([]models.InboundRecipientRule, error) {
	return m.byDomain[domainID], nil
}

func (m *mockInboundRuleStore) DeleteInboundRecipientRule(_ context.Context, domainID, ruleID int64) error {
	rules := m.byDomain[domainID]
	for i, r := range rules {
		if r.ID == ruleID {
			m.byDomain[domainID] = append(rules[:i], rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestIngest_RoutesToVerifiedDomain(t *testing.T) {
	ds := newMockDomainStore()
	ds.byName["example.com"] = &models.Domain{
		ID:       11,
		UserID:   7,
		Name:     "example.com",
		Verified: true,
	}
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	rs := newMockInboundRuleStore()
	cs.byDomainID[11] = &models.InboundDomainConfig{
		DomainID:   11,
		MXTarget:   "mx.example.com",
		MXVerified: true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	svc := NewService(ds, es, cs, rs, nil)

	res, err := svc.Ingest(context.Background(), Message{
		Sender:     "sender@outside.com",
		Recipients: []string{"ideas@example.com"},
		Subject:    "Hello",
		TextBody:   "world",
		MessageID:  "abc-123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Accepted != 1 || res.Dropped != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(es.items) != 1 {
		t.Fatalf("expected 1 email create call, got %d", len(es.items))
	}
	if es.items[0].UserID != 7 {
		t.Fatalf("expected user 7, got %d", es.items[0].UserID)
	}
}

func TestIngest_DropsUnknownDomain(t *testing.T) {
	ds := newMockDomainStore()
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	res, err := svc.Ingest(context.Background(), Message{
		Sender:     "sender@outside.com",
		Recipients: []string{"ideas@missing.com"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.Accepted != 0 || res.Dropped != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestIngest_RequiresSender(t *testing.T) {
	ds := newMockDomainStore()
	es := &mockInboundEmailStore{}
	cs := newMockInboundDomainConfigStore()
	rs := newMockInboundRuleStore()
	svc := NewService(ds, es, cs, rs, nil)

	_, err := svc.Ingest(context.Background(), Message{
		Recipients: []string{"ideas@example.com"},
	})
	if !errors.Is(err, ErrSenderRequired) {
		t.Fatalf("expected ErrSenderRequired, got %v", err)
	}
}
