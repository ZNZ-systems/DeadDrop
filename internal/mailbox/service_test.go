package mailbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockMailboxStore struct {
	mailboxes  map[int64]*models.Mailbox
	byPublicID map[uuid.UUID]*models.Mailbox
	byUserID   map[int64][]models.Mailbox
	nextID     int64
}

func newMockMailboxStore() *mockMailboxStore {
	return &mockMailboxStore{
		mailboxes:  make(map[int64]*models.Mailbox),
		byPublicID: make(map[uuid.UUID]*models.Mailbox),
		byUserID:   make(map[int64][]models.Mailbox),
		nextID:     1,
	}
}

func (m *mockMailboxStore) CreateMailbox(_ context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error) {
	mb := &models.Mailbox{
		ID:          m.nextID,
		PublicID:    uuid.New(),
		UserID:      userID,
		DomainID:    domainID,
		Name:        name,
		FromAddress: fromAddress,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.nextID++
	m.mailboxes[mb.ID] = mb
	m.byPublicID[mb.PublicID] = mb
	m.byUserID[userID] = append(m.byUserID[userID], *mb)
	return mb, nil
}

func (m *mockMailboxStore) GetMailboxesByUserID(_ context.Context, userID int64) ([]models.Mailbox, error) {
	return m.byUserID[userID], nil
}

func (m *mockMailboxStore) GetMailboxByID(_ context.Context, id int64) (*models.Mailbox, error) {
	mb, ok := m.mailboxes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStore) GetMailboxByPublicID(_ context.Context, publicID uuid.UUID) (*models.Mailbox, error) {
	mb, ok := m.byPublicID[publicID]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStore) DeleteMailbox(_ context.Context, id int64) error {
	delete(m.mailboxes, id)
	return nil
}

type mockDomainStoreForMailbox struct {
	domains map[int64]*models.Domain
}

func newMockDomainStoreForMailbox() *mockDomainStoreForMailbox {
	return &mockDomainStoreForMailbox{
		domains: make(map[int64]*models.Domain),
	}
}

func (m *mockDomainStoreForMailbox) addDomain(d *models.Domain) {
	m.domains[d.ID] = d
}

func (m *mockDomainStoreForMailbox) GetDomainByID(_ context.Context, id int64) (*models.Domain, error) {
	d, ok := m.domains[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	mb, err := svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mb.Name != "Support" {
		t.Errorf("expected name Support, got %s", mb.Name)
	}
	if mb.FromAddress != "support@example.com" {
		t.Errorf("expected from_address support@example.com, got %s", mb.FromAddress)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "", "support@example.com")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreate_DomainNotVerified(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: false, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	if err == nil {
		t.Fatal("expected error for unverified domain")
	}
}

func TestCreate_FromAddressDomainMismatch(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "Support", "support@other.com")
	if err == nil {
		t.Fatal("expected error for from_address domain mismatch")
	}
}

func TestList_ReturnsMailboxes(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, _ = svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	_, _ = svc.Create(context.Background(), 1, 1, "Sales", "sales@example.com")

	mailboxes, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mailboxes) != 2 {
		t.Errorf("expected 2 mailboxes, got %d", len(mailboxes))
	}
}
