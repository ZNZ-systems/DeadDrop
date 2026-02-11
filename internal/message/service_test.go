package message

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockMessageStore struct {
	messages map[int64]*models.Message
	byDomain map[int64][]*models.Message
	nextID   int64
}

func newMockMessageStore() *mockMessageStore {
	return &mockMessageStore{
		messages: make(map[int64]*models.Message),
		byDomain: make(map[int64][]*models.Message),
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
	m.byDomain[domainID] = append(m.byDomain[domainID], msg)
	return msg, nil
}

func (m *mockMessageStore) GetMessagesByDomainID(_ context.Context, domainID int64, limit, offset int) ([]models.Message, error) {
	msgs := m.byDomain[domainID]
	result := make([]models.Message, 0)
	start := offset
	if start >= len(msgs) {
		return result, nil
	}
	end := start + limit
	if end > len(msgs) {
		end = len(msgs)
	}
	for _, msg := range msgs[start:end] {
		result = append(result, *msg)
	}
	return result, nil
}

func (m *mockMessageStore) MarkMessageRead(_ context.Context, id int64) error {
	msg, ok := m.messages[id]
	if !ok {
		return errors.New("message not found")
	}
	msg.IsRead = true
	return nil
}

func (m *mockMessageStore) DeleteMessage(_ context.Context, id int64) error {
	msg, ok := m.messages[id]
	if !ok {
		return errors.New("message not found")
	}
	delete(m.messages, id)
	// Remove from domain list
	domainMsgs := m.byDomain[msg.DomainID]
	for i, dm := range domainMsgs {
		if dm.ID == id {
			m.byDomain[msg.DomainID] = append(domainMsgs[:i], domainMsgs[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockMessageStore) CountUnreadByDomainID(_ context.Context, domainID int64) (int, error) {
	count := 0
	for _, msg := range m.byDomain[domainID] {
		if !msg.IsRead {
			count++
		}
	}
	return count, nil
}

type mockDomainStore struct {
	domains    map[uuid.UUID]*models.Domain
	byID       map[int64]*models.Domain
}

func newMockDomainStore() *mockDomainStore {
	return &mockDomainStore{
		domains: make(map[uuid.UUID]*models.Domain),
		byID:    make(map[int64]*models.Domain),
	}
}

func (m *mockDomainStore) addDomain(d *models.Domain) {
	m.domains[d.PublicID] = d
	m.byID[d.ID] = d
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
	return nil, nil
}

func (m *mockDomainStore) MarkDomainVerified(_ context.Context, _ int64) error {
	return nil
}

func (m *mockDomainStore) DeleteDomain(_ context.Context, _ int64) error {
	return nil
}

// --- Mock notifier ---

type mockNotifier struct {
	called atomic.Int32
}

func (m *mockNotifier) NotifyNewMessage(_ context.Context, _ *models.Domain, _ *models.Message) error {
	m.called.Add(1)
	return nil
}

// --- Helper to create a verified domain ---

func makeVerifiedDomain(ds *mockDomainStore) *models.Domain {
	d := &models.Domain{
		ID:                1,
		PublicID:          uuid.New(),
		UserID:            1,
		Name:              "example.com",
		VerificationToken: uuid.New().String(),
		Verified:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	ds.addDomain(d)
	return d
}

func makeUnverifiedDomain(ds *mockDomainStore) *models.Domain {
	d := &models.Domain{
		ID:                2,
		PublicID:          uuid.New(),
		UserID:            1,
		Name:              "unverified.com",
		VerificationToken: uuid.New().String(),
		Verified:          false,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	ds.addDomain(d)
	return d
}

// --- Tests ---

func TestSubmit_Success(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &mockNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeVerifiedDomain(ds)

	err := svc.Submit(context.Background(), domain.PublicID, "John", "john@example.com", "Hello!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify message was created
	msgs, _ := svc.List(context.Background(), domain.ID, 10, 0)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "Hello!" {
		t.Errorf("expected body 'Hello!', got %q", msgs[0].Body)
	}
	if msgs[0].SenderName != "John" {
		t.Errorf("expected sender 'John', got %q", msgs[0].SenderName)
	}
}

func TestSubmit_DomainNotFound(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &mockNotifier{}
	svc := NewService(ms, ds, notifier)

	err := svc.Submit(context.Background(), uuid.New(), "John", "john@example.com", "Hello!")
	if !errors.Is(err, ErrDomainNotFound) {
		t.Fatalf("expected ErrDomainNotFound, got %v", err)
	}
}

func TestSubmit_DomainNotVerified(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &mockNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeUnverifiedDomain(ds)

	err := svc.Submit(context.Background(), domain.PublicID, "John", "john@example.com", "Hello!")
	if !errors.Is(err, ErrDomainNotVerified) {
		t.Fatalf("expected ErrDomainNotVerified, got %v", err)
	}
}

func TestList_WithPagination(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &NoopNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeVerifiedDomain(ds)

	// Create 5 messages
	for i := 0; i < 5; i++ {
		_ = svc.Submit(context.Background(), domain.PublicID, "User", "u@e.com", "msg")
	}

	// Page 1: first 3
	msgs, err := svc.List(context.Background(), domain.ID, 3, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}

	// Page 2: next 2
	msgs, err = svc.List(context.Background(), domain.ID, 3, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestMarkRead(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &NoopNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeVerifiedDomain(ds)
	_ = svc.Submit(context.Background(), domain.PublicID, "User", "u@e.com", "Hello")

	// Get the message
	msgs, _ := svc.List(context.Background(), domain.ID, 10, 0)
	if msgs[0].IsRead {
		t.Error("message should start as unread")
	}

	err := svc.MarkRead(context.Background(), msgs[0].ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's marked read in the store
	if !ms.messages[msgs[0].ID].IsRead {
		t.Error("message should be marked as read")
	}
}

func TestDelete(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &NoopNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeVerifiedDomain(ds)
	_ = svc.Submit(context.Background(), domain.PublicID, "User", "u@e.com", "Hello")

	msgs, _ := svc.List(context.Background(), domain.ID, 10, 0)

	err := svc.Delete(context.Background(), msgs[0].ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	msgs, _ = svc.List(context.Background(), domain.ID, 10, 0)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(msgs))
	}
}

func TestCountUnread(t *testing.T) {
	ms := newMockMessageStore()
	ds := newMockDomainStore()
	notifier := &NoopNotifier{}
	svc := NewService(ms, ds, notifier)

	domain := makeVerifiedDomain(ds)

	// Submit 3 messages
	_ = svc.Submit(context.Background(), domain.PublicID, "A", "a@e.com", "m1")
	_ = svc.Submit(context.Background(), domain.PublicID, "B", "b@e.com", "m2")
	_ = svc.Submit(context.Background(), domain.PublicID, "C", "c@e.com", "m3")

	count, err := svc.CountUnread(context.Background(), domain.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 unread, got %d", count)
	}

	// Mark one as read
	msgs, _ := svc.List(context.Background(), domain.ID, 10, 0)
	_ = svc.MarkRead(context.Background(), msgs[0].ID)

	count, _ = svc.CountUnread(context.Background(), domain.ID)
	if count != 2 {
		t.Errorf("expected 2 unread after marking one read, got %d", count)
	}
}
