package inbound

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
)

type mockMXResolver struct {
	records []*net.MX
	err     error
}

func (m *mockMXResolver) LookupMX(_ string) ([]*net.MX, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.records, nil
}

type mockInboundConfigStore struct {
	cfgByDomain map[int64]*models.InboundDomainConfig
}

func newMockInboundConfigStore() *mockInboundConfigStore {
	return &mockInboundConfigStore{cfgByDomain: map[int64]*models.InboundDomainConfig{}}
}

func (m *mockInboundConfigStore) UpsertInboundDomainConfig(_ context.Context, domainID int64, mxTarget string) (*models.InboundDomainConfig, error) {
	cfg, ok := m.cfgByDomain[domainID]
	if !ok {
		now := time.Now()
		cfg = &models.InboundDomainConfig{
			DomainID:   domainID,
			MXTarget:   mxTarget,
			MXVerified: false,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		m.cfgByDomain[domainID] = cfg
	}
	return cfg, nil
}

func (m *mockInboundConfigStore) GetInboundDomainConfigByDomainID(_ context.Context, domainID int64) (*models.InboundDomainConfig, error) {
	cfg, ok := m.cfgByDomain[domainID]
	if !ok {
		return nil, errors.New("not found")
	}
	return cfg, nil
}

func (m *mockInboundConfigStore) UpdateInboundDomainVerification(_ context.Context, domainID int64, verified bool, lastError string) error {
	cfg, ok := m.cfgByDomain[domainID]
	if !ok {
		return errors.New("not found")
	}
	cfg.MXVerified = verified
	cfg.LastError = lastError
	now := time.Now()
	cfg.CheckedAt = &now
	cfg.UpdatedAt = now
	return nil
}

func TestDomainServiceVerifyMX_Success(t *testing.T) {
	store := newMockInboundConfigStore()
	resolver := &mockMXResolver{
		records: []*net.MX{
			{Host: "mx.other.test."},
			{Host: "mx.deaddrop.local."},
		},
	}
	svc := NewDomainService(store, resolver, "mx.deaddrop.local")

	ok, err := svc.VerifyMX(context.Background(), &models.Domain{
		ID:   10,
		Name: "example.com",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatalf("expected verification success")
	}
	cfg, _ := store.GetInboundDomainConfigByDomainID(context.Background(), 10)
	if !cfg.MXVerified {
		t.Fatalf("expected config to be marked verified")
	}
}

func TestDomainServiceVerifyMX_NotFound(t *testing.T) {
	store := newMockInboundConfigStore()
	resolver := &mockMXResolver{
		records: []*net.MX{
			{Host: "mail.provider.com."},
		},
	}
	svc := NewDomainService(store, resolver, "mx.deaddrop.local")

	ok, err := svc.VerifyMX(context.Background(), &models.Domain{
		ID:   11,
		Name: "example.com",
	})
	if err != nil {
		t.Fatalf("expected no hard error, got %v", err)
	}
	if ok {
		t.Fatalf("expected verification failure")
	}
	cfg, _ := store.GetInboundDomainConfigByDomainID(context.Background(), 11)
	if cfg.LastError == "" {
		t.Fatalf("expected last error to be set")
	}
}
