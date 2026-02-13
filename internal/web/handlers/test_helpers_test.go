package handlers

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Shared mock stores used by messages_test.go and domains_test.go ---

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

func (m *mockMessageStore) GetMessageByID(_ context.Context, id int64) (*models.Message, error) {
	msg, ok := m.messages[id]
	if !ok {
		return nil, errors.New("not found")
	}
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
func (m *mockDomainStore) GetDomainByID(_ context.Context, id int64) (*models.Domain, error) {
	for _, d := range m.domains {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, sql.ErrNoRows
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
