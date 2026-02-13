package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores for API tests ---

type mockStreamStoreForAPI struct {
	streams map[uuid.UUID]*models.Stream
}

func newMockStreamStoreForAPI() *mockStreamStoreForAPI {
	return &mockStreamStoreForAPI{
		streams: make(map[uuid.UUID]*models.Stream),
	}
}

func (m *mockStreamStoreForAPI) addStream(s *models.Stream) {
	m.streams[s.WidgetID] = s
}

func (m *mockStreamStoreForAPI) CreateStream(_ context.Context, _ int64, _ string, _ string, _ uuid.UUID) (*models.Stream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStreamStoreForAPI) GetStreamsByMailboxID(_ context.Context, _ int64) ([]models.Stream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStreamStoreForAPI) GetStreamByWidgetID(_ context.Context, widgetID uuid.UUID) (*models.Stream, error) {
	s, ok := m.streams[widgetID]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *mockStreamStoreForAPI) GetStreamByAddress(_ context.Context, _ string) (*models.Stream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockStreamStoreForAPI) DeleteStream(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

type mockConvStoreForAPI struct {
	conversations map[int64]*models.Conversation
	messages      map[int64][]models.ConversationMessage
	nextID        int64
	nextMsgID     int64
}

func newMockConvStoreForAPI() *mockConvStoreForAPI {
	return &mockConvStoreForAPI{
		conversations: make(map[int64]*models.Conversation),
		messages:      make(map[int64][]models.ConversationMessage),
		nextID:        1,
		nextMsgID:     1,
	}
}

func (m *mockConvStoreForAPI) CreateConversation(_ context.Context, mailboxID, streamID int64, subject string) (*models.Conversation, error) {
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
	return c, nil
}

func (m *mockConvStoreForAPI) GetConversationByID(_ context.Context, id int64) (*models.Conversation, error) {
	c, ok := m.conversations[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return c, nil
}

func (m *mockConvStoreForAPI) GetConversationByPublicID(_ context.Context, _ uuid.UUID) (*models.Conversation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockConvStoreForAPI) GetConversationsByMailboxID(_ context.Context, _ int64, _, _ int) ([]models.Conversation, error) {
	return nil, nil
}

func (m *mockConvStoreForAPI) UpdateConversationStatus(_ context.Context, _ int64, _ string) error {
	return nil
}

func (m *mockConvStoreForAPI) CountOpenByMailboxID(_ context.Context, _ int64) (int, error) {
	return 0, nil
}

func (m *mockConvStoreForAPI) CreateMessage(_ context.Context, conversationID int64, direction, senderAddress, senderName, body string) (*models.ConversationMessage, error) {
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

func (m *mockConvStoreForAPI) GetMessagesByConversationID(_ context.Context, conversationID int64) ([]models.ConversationMessage, error) {
	return m.messages[conversationID], nil
}

type mockMailboxStoreForAPI struct {
	mailboxes map[int64]*models.Mailbox
}

func newMockMailboxStoreForAPI() *mockMailboxStoreForAPI {
	return &mockMailboxStoreForAPI{
		mailboxes: make(map[int64]*models.Mailbox),
	}
}

func (m *mockMailboxStoreForAPI) addMailbox(mb *models.Mailbox) {
	m.mailboxes[mb.ID] = mb
}

func (m *mockMailboxStoreForAPI) CreateMailbox(_ context.Context, _, _ int64, _, _ string) (*models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForAPI) GetMailboxesByUserID(_ context.Context, _ int64) ([]models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForAPI) GetMailboxByID(_ context.Context, id int64) (*models.Mailbox, error) {
	mb, ok := m.mailboxes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStoreForAPI) GetMailboxByPublicID(_ context.Context, _ uuid.UUID) (*models.Mailbox, error) {
	return nil, errors.New("not implemented")
}

func (m *mockMailboxStoreForAPI) DeleteMailbox(_ context.Context, _ int64) error {
	return errors.New("not implemented")
}

// --- Test helpers ---

func makeAPIHandler(widgetID uuid.UUID, enabled bool) *APIHandler {
	ss := newMockStreamStoreForAPI()
	cs := newMockConvStoreForAPI()
	ms := newMockMailboxStoreForAPI()
	ms.addMailbox(&models.Mailbox{ID: 1, Name: "Support", FromAddress: "support@example.com"})

	if widgetID != uuid.Nil {
		ss.addStream(&models.Stream{
			ID:        1,
			MailboxID: 1,
			Type:      models.StreamTypeForm,
			WidgetID:  widgetID,
			Enabled:   enabled,
		})
	}

	convService := conversation.NewService(cs, ms, &conversation.NoopNotifier{}, &conversation.NoopSender{})
	return NewAPIHandler(ss, convService)
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
	widgetID := uuid.New()
	handler := makeAPIHandler(widgetID, true)

	values := url.Values{
		"domain_id": {widgetID.String()},
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
	handler := makeAPIHandler(uuid.Nil, true)

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
	handler := makeAPIHandler(uuid.Nil, true)

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
	widgetID := uuid.New()
	handler := makeAPIHandler(widgetID, true)

	values := url.Values{
		"domain_id": {widgetID.String()},
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
	widgetID := uuid.New()
	handler := makeAPIHandler(widgetID, true)

	values := url.Values{
		"domain_id": {widgetID.String()},
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

func TestHandleSubmitMessage_StreamNotFound(t *testing.T) {
	handler := makeAPIHandler(uuid.Nil, true) // no streams registered

	values := url.Values{
		"domain_id": {uuid.New().String()},
		"message":   {"Hello"},
	}

	rr := postForm(handler.HandleSubmitMessage, values)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
	resp := parseJSONResponse(t, rr)
	if resp.Error != "domain not found" {
		t.Errorf("unexpected error: %s", resp.Error)
	}
}

func TestHandleSubmitMessage_StreamDisabled(t *testing.T) {
	widgetID := uuid.New()
	handler := makeAPIHandler(widgetID, false) // stream disabled

	values := url.Values{
		"domain_id": {widgetID.String()},
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
