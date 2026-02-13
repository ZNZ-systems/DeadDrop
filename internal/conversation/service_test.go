package conversation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockConversationStore struct {
	conversations map[int64]*models.Conversation
	byPublicID    map[uuid.UUID]*models.Conversation
	byMailbox     map[int64][]models.Conversation
	messages      map[int64][]models.ConversationMessage
	nextID        int64
	nextMsgID     int64
}

func newMockConversationStore() *mockConversationStore {
	return &mockConversationStore{
		conversations: make(map[int64]*models.Conversation),
		byPublicID:    make(map[uuid.UUID]*models.Conversation),
		byMailbox:     make(map[int64][]models.Conversation),
		messages:      make(map[int64][]models.ConversationMessage),
		nextID:        1,
		nextMsgID:     1,
	}
}

func (m *mockConversationStore) CreateConversation(_ context.Context, mailboxID, streamID int64, subject string) (*models.Conversation, error) {
	c := &models.Conversation{
		ID:        m.nextID,
		PublicID:  uuid.New(),
		MailboxID: mailboxID,
		StreamID:  streamID,
		Subject:   subject,
		Status:    models.ConversationOpen,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.nextID++
	m.conversations[c.ID] = c
	m.byPublicID[c.PublicID] = c
	m.byMailbox[mailboxID] = append(m.byMailbox[mailboxID], *c)
	return c, nil
}

func (m *mockConversationStore) GetConversationByID(_ context.Context, id int64) (*models.Conversation, error) {
	c, ok := m.conversations[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (m *mockConversationStore) GetConversationByPublicID(_ context.Context, publicID uuid.UUID) (*models.Conversation, error) {
	c, ok := m.byPublicID[publicID]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (m *mockConversationStore) GetConversationsByMailboxID(_ context.Context, mailboxID int64, limit, offset int) ([]models.Conversation, error) {
	all := m.byMailbox[mailboxID]
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}

func (m *mockConversationStore) UpdateConversationStatus(_ context.Context, id int64, status string) error {
	c, ok := m.conversations[id]
	if !ok {
		return errors.New("not found")
	}
	c.Status = models.ConversationStatus(status)
	return nil
}

func (m *mockConversationStore) CountOpenByMailboxID(_ context.Context, mailboxID int64) (int, error) {
	count := 0
	for _, c := range m.byMailbox[mailboxID] {
		if c.Status == models.ConversationOpen {
			count++
		}
	}
	return count, nil
}

func (m *mockConversationStore) CreateMessage(_ context.Context, conversationID int64, direction, senderAddress, senderName, body string) (*models.ConversationMessage, error) {
	msg := &models.ConversationMessage{
		ID:             m.nextMsgID,
		PublicID:       uuid.New(),
		ConversationID: conversationID,
		Direction:      models.MessageDirection(direction),
		SenderAddress:  senderAddress,
		SenderName:     senderName,
		Body:           body,
		CreatedAt:      time.Now(),
	}
	m.nextMsgID++
	m.messages[conversationID] = append(m.messages[conversationID], *msg)
	return msg, nil
}

func (m *mockConversationStore) GetMessagesByConversationID(_ context.Context, conversationID int64) ([]models.ConversationMessage, error) {
	return m.messages[conversationID], nil
}

type mockMailboxStoreForConv struct {
	mailboxes map[int64]*models.Mailbox
}

func newMockMailboxStoreForConv() *mockMailboxStoreForConv {
	return &mockMailboxStoreForConv{
		mailboxes: make(map[int64]*models.Mailbox),
	}
}

func (m *mockMailboxStoreForConv) addMailbox(mb *models.Mailbox) {
	m.mailboxes[mb.ID] = mb
}

func (m *mockMailboxStoreForConv) CreateMailbox(_ context.Context, _, _ int64, _, _ string) (*models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForConv) GetMailboxesByUserID(_ context.Context, _ int64) ([]models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForConv) GetMailboxByID(_ context.Context, id int64) (*models.Mailbox, error) {
	mb, ok := m.mailboxes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStoreForConv) GetMailboxByPublicID(_ context.Context, _ uuid.UUID) (*models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForConv) DeleteMailbox(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type recordingSender struct {
	calls []sendCall
}

type sendCall struct {
	to, fromAddress, fromName, subject, body string
}

func (s *recordingSender) SendReply(_ context.Context, to, fromAddress, fromName, subject, body string) error {
	s.calls = append(s.calls, sendCall{to, fromAddress, fromName, subject, body})
	return nil
}

// --- Tests ---

func TestStartConversation_Success(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	ms.addMailbox(&models.Mailbox{ID: 1, Name: "Support", FromAddress: "support@example.com"})
	svc := NewService(cs, ms, &NoopNotifier{}, &NoopSender{})

	stream := &models.Stream{
		ID:        1,
		MailboxID: 1,
		Type:      models.StreamTypeForm,
		Enabled:   true,
	}

	conv, err := svc.StartConversation(context.Background(), stream, "Hello", "sender@test.com", "Alice", "Hi there")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if conv.MailboxID != 1 {
		t.Errorf("expected mailbox ID 1, got %d", conv.MailboxID)
	}
	if conv.StreamID != 1 {
		t.Errorf("expected stream ID 1, got %d", conv.StreamID)
	}
	if conv.Subject != "Hello" {
		t.Errorf("expected subject Hello, got %s", conv.Subject)
	}

	// Verify the inbound message was created
	msgs := cs.messages[conv.ID]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Direction != models.MessageInbound {
		t.Errorf("expected inbound direction, got %s", msgs[0].Direction)
	}
	if msgs[0].SenderAddress != "sender@test.com" {
		t.Errorf("expected sender sender@test.com, got %s", msgs[0].SenderAddress)
	}
}

func TestStartConversation_StreamDisabled(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	svc := NewService(cs, ms, &NoopNotifier{}, &NoopSender{})

	stream := &models.Stream{
		ID:        1,
		MailboxID: 1,
		Enabled:   false,
	}

	_, err := svc.StartConversation(context.Background(), stream, "Hello", "sender@test.com", "Alice", "Hi")
	if !errors.Is(err, ErrStreamDisabled) {
		t.Fatalf("expected ErrStreamDisabled, got %v", err)
	}
}

func TestReply_Success(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	ms.addMailbox(&models.Mailbox{ID: 1, Name: "Support", FromAddress: "support@example.com"})
	sender := &recordingSender{}
	svc := NewService(cs, ms, &NoopNotifier{}, sender)

	stream := &models.Stream{ID: 1, MailboxID: 1, Enabled: true}
	conv, _ := svc.StartConversation(context.Background(), stream, "Question", "alice@test.com", "Alice", "Need help")

	msg, err := svc.Reply(context.Background(), conv.ID, "Sure, how can I help?")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msg.Direction != models.MessageOutbound {
		t.Errorf("expected outbound direction, got %s", msg.Direction)
	}
	if msg.Body != "Sure, how can I help?" {
		t.Errorf("unexpected reply body: %s", msg.Body)
	}

	// Verify the sender was called
	if len(sender.calls) != 1 {
		t.Fatalf("expected 1 send call, got %d", len(sender.calls))
	}
	call := sender.calls[0]
	if call.to != "alice@test.com" {
		t.Errorf("expected reply to alice@test.com, got %s", call.to)
	}
	if call.fromAddress != "support@example.com" {
		t.Errorf("expected from support@example.com, got %s", call.fromAddress)
	}
	if call.subject != "Re: Question" {
		t.Errorf("expected subject Re: Question, got %s", call.subject)
	}
}

func TestReply_ConversationClosed(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	ms.addMailbox(&models.Mailbox{ID: 1, Name: "Support", FromAddress: "support@example.com"})
	svc := NewService(cs, ms, &NoopNotifier{}, &NoopSender{})

	stream := &models.Stream{ID: 1, MailboxID: 1, Enabled: true}
	conv, _ := svc.StartConversation(context.Background(), stream, "Question", "alice@test.com", "Alice", "Hi")

	_ = svc.Close(context.Background(), conv.ID)

	_, err := svc.Reply(context.Background(), conv.ID, "Too late")
	if !errors.Is(err, ErrConversationClosed) {
		t.Fatalf("expected ErrConversationClosed, got %v", err)
	}
}

func TestClose_Success(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	svc := NewService(cs, ms, &NoopNotifier{}, &NoopSender{})

	stream := &models.Stream{ID: 1, MailboxID: 1, Enabled: true}
	conv, _ := svc.StartConversation(context.Background(), stream, "Subject", "a@b.com", "A", "body")

	err := svc.Close(context.Background(), conv.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify status changed
	updated, _ := cs.GetConversationByID(context.Background(), conv.ID)
	if updated.Status != models.ConversationClosed {
		t.Errorf("expected closed status, got %s", updated.Status)
	}
}

func TestListConversations(t *testing.T) {
	cs := newMockConversationStore()
	ms := newMockMailboxStoreForConv()
	ms.addMailbox(&models.Mailbox{ID: 1, Name: "Support", FromAddress: "support@example.com"})
	svc := NewService(cs, ms, &NoopNotifier{}, &NoopSender{})

	stream := &models.Stream{ID: 1, MailboxID: 1, Enabled: true}
	_, _ = svc.StartConversation(context.Background(), stream, "First", "a@b.com", "A", "body1")
	_, _ = svc.StartConversation(context.Background(), stream, "Second", "c@d.com", "C", "body2")

	convos, err := svc.List(context.Background(), 1, 50, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(convos) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convos))
	}
}
