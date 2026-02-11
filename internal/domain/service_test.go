package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockDomainStore struct {
	domains     map[int64]*models.Domain
	byPublicID  map[uuid.UUID]*models.Domain
	byName      map[string]*models.Domain
	byUserID    map[int64][]models.Domain
	nextID      int64
	createErr   error
}

func newMockDomainStore() *mockDomainStore {
	return &mockDomainStore{
		domains:    make(map[int64]*models.Domain),
		byPublicID: make(map[uuid.UUID]*models.Domain),
		byName:     make(map[string]*models.Domain),
		byUserID:   make(map[int64][]models.Domain),
		nextID:     1,
	}
}

func (m *mockDomainStore) CreateDomain(_ context.Context, userID int64, name, verificationToken string) (*models.Domain, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	d := &models.Domain{
		ID:                m.nextID,
		PublicID:          uuid.New(),
		UserID:            userID,
		Name:              name,
		VerificationToken: verificationToken,
		Verified:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	m.nextID++
	m.domains[d.ID] = d
	m.byPublicID[d.PublicID] = d
	m.byName[d.Name] = d
	m.byUserID[userID] = append(m.byUserID[userID], *d)
	return d, nil
}

func (m *mockDomainStore) GetDomainsByUserID(_ context.Context, userID int64) ([]models.Domain, error) {
	return m.byUserID[userID], nil
}

func (m *mockDomainStore) GetDomainByID(_ context.Context, id int64) (*models.Domain, error) {
	d, ok := m.domains[id]
	if !ok {
		return nil, errors.New("domain not found")
	}
	return d, nil
}

func (m *mockDomainStore) GetDomainByPublicID(_ context.Context, publicID uuid.UUID) (*models.Domain, error) {
	d, ok := m.byPublicID[publicID]
	if !ok {
		return nil, errors.New("domain not found")
	}
	return d, nil
}

func (m *mockDomainStore) GetDomainByName(_ context.Context, name string) (*models.Domain, error) {
	d, ok := m.byName[name]
	if !ok {
		return nil, errors.New("domain not found")
	}
	return d, nil
}

func (m *mockDomainStore) MarkDomainVerified(_ context.Context, id int64) error {
	d, ok := m.domains[id]
	if !ok {
		return errors.New("domain not found")
	}
	d.Verified = true
	return nil
}

func (m *mockDomainStore) DeleteDomain(_ context.Context, id int64) error {
	d, ok := m.domains[id]
	if !ok {
		return errors.New("domain not found")
	}
	delete(m.byPublicID, d.PublicID)
	delete(m.byName, d.Name)
	delete(m.domains, id)
	return nil
}

// --- Mock DNS resolver ---

type mockDNSResolver struct {
	records map[string][]string
	err     error
}

func (m *mockDNSResolver) LookupTXT(host string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.records[host], nil
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	d, err := svc.Create(context.Background(), 1, "example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if d.Name != "example.com" {
		t.Errorf("expected name example.com, got %s", d.Name)
	}
	if d.UserID != 1 {
		t.Errorf("expected userID 1, got %d", d.UserID)
	}
	if d.Verified {
		t.Error("new domain should not be verified")
	}
	if d.VerificationToken == "" {
		t.Error("verification token should be set")
	}
}

func TestCreate_EmptyName(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	_, err := svc.Create(context.Background(), 1, "")
	if err == nil {
		t.Fatal("expected error for empty domain name")
	}
}

func TestCreate_WhitespaceName(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	_, err := svc.Create(context.Background(), 1, "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only domain name")
	}
}

func TestCreate_TrimsWhitespace(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	d, err := svc.Create(context.Background(), 1, "  example.com  ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if d.Name != "example.com" {
		t.Errorf("expected trimmed name, got %q", d.Name)
	}
}

func TestList_ReturnsDomains(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	_, _ = svc.Create(context.Background(), 1, "a.com")
	_, _ = svc.Create(context.Background(), 1, "b.com")
	_, _ = svc.Create(context.Background(), 2, "c.com")

	domains, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(domains) != 2 {
		t.Errorf("expected 2 domains for user 1, got %d", len(domains))
	}
}

func TestGetByPublicID_Found(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	created, _ := svc.Create(context.Background(), 1, "example.com")

	found, err := svc.GetByPublicID(context.Background(), created.PublicID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.Name != "example.com" {
		t.Errorf("expected example.com, got %s", found.Name)
	}
}

func TestGetByPublicID_NotFound(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	_, err := svc.GetByPublicID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for nonexistent domain")
	}
}

func TestVerify_Success(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{
		records: map[string][]string{},
	}
	svc := NewService(store, resolver)

	d, _ := svc.Create(context.Background(), 1, "example.com")

	// Add the expected TXT record
	resolver.records["example.com"] = []string{
		"v=spf1 include:_spf.google.com ~all",
		"deaddrop-verify=" + d.VerificationToken,
	}

	err := svc.Verify(context.Background(), d)
	if err != nil {
		t.Fatalf("expected verification to succeed, got %v", err)
	}

	// Check domain is now verified in the store
	if !store.domains[d.ID].Verified {
		t.Error("domain should be marked as verified")
	}
}

func TestVerify_NoMatchingRecord(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{
		records: map[string][]string{
			"example.com": {"v=spf1 include:_spf.google.com ~all"},
		},
	}
	svc := NewService(store, resolver)

	d, _ := svc.Create(context.Background(), 1, "example.com")

	err := svc.Verify(context.Background(), d)
	if err == nil {
		t.Fatal("expected error when no matching TXT record")
	}
}

func TestVerify_DNSLookupError(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{
		err: errors.New("dns lookup failed"),
	}
	svc := NewService(store, resolver)

	d, _ := svc.Create(context.Background(), 1, "example.com")

	err := svc.Verify(context.Background(), d)
	if err == nil {
		t.Fatal("expected error on DNS failure")
	}
}

func TestDelete_Success(t *testing.T) {
	store := newMockDomainStore()
	resolver := &mockDNSResolver{}
	svc := NewService(store, resolver)

	d, _ := svc.Create(context.Background(), 1, "example.com")

	err := svc.Delete(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not be found anymore
	_, err = svc.GetByPublicID(context.Background(), d.PublicID)
	if err == nil {
		t.Error("expected domain to be deleted")
	}
}
