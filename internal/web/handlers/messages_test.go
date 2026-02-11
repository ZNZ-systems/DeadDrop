package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/message"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
)

func setupMessageTestRouter(user *models.User, msgStore *mockMessageStore, domainStore *mockDomainStore) *chi.Mux {
	notifier := &mockNotifier{}
	svc := message.NewService(msgStore, domainStore, notifier)

	handler := NewMessageHandler(svc, msgStore, domainStore, nil)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/messages/{messageID}/read", handler.HandleMarkRead)
	r.Delete("/messages/{messageID}", handler.HandleDeleteMessage)

	return r
}

func TestHandleMarkRead_IDOR_Returns404(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}
	userB := &models.User{ID: 2, Email: "b@test.com"}

	ms := newMockMessageStore()
	ds := newMockDomainStore()

	// Domain owned by user B
	domainB := &models.Domain{
		ID:                10,
		PublicID:          uuid.New(),
		UserID:            userB.ID,
		Name:              "b-domain.com",
		VerificationToken: "tok-b",
		Verified:          true,
	}
	ds.addDomain(domainB)

	// Message belonging to user B's domain
	msgB := &models.Message{
		ID:          1,
		PublicID:    uuid.New(),
		DomainID:    domainB.ID,
		SenderName:  "Sender",
		SenderEmail: "sender@example.com",
		Body:        "Hello",
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	ms.messages[msgB.ID] = msgB

	router := setupMessageTestRouter(userA, ms, ds) // requests as user A

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/messages/%d/read", msgB.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when marking another user's message as read, got %d", rr.Code)
	}
}

func TestHandleDeleteMessage_IDOR_Returns404(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}
	userB := &models.User{ID: 2, Email: "b@test.com"}

	ms := newMockMessageStore()
	ds := newMockDomainStore()

	domainB := &models.Domain{
		ID:                10,
		PublicID:          uuid.New(),
		UserID:            userB.ID,
		Name:              "b-domain.com",
		VerificationToken: "tok-b",
		Verified:          true,
	}
	ds.addDomain(domainB)

	msgB := &models.Message{
		ID:          1,
		PublicID:    uuid.New(),
		DomainID:    domainB.ID,
		SenderName:  "Sender",
		SenderEmail: "sender@example.com",
		Body:        "Hello",
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	ms.messages[msgB.ID] = msgB

	router := setupMessageTestRouter(userA, ms, ds)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/messages/%d", msgB.ID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when deleting another user's message, got %d", rr.Code)
	}
}

func TestHandleMarkRead_OwnMessage_Succeeds(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}

	ms := newMockMessageStore()
	ds := newMockDomainStore()

	domainA := &models.Domain{
		ID:                10,
		PublicID:          uuid.New(),
		UserID:            userA.ID,
		Name:              "a-domain.com",
		VerificationToken: "tok-a",
		Verified:          true,
	}
	ds.addDomain(domainA)

	msgA := &models.Message{
		ID:          1,
		PublicID:    uuid.New(),
		DomainID:    domainA.ID,
		SenderName:  "Sender",
		SenderEmail: "sender@example.com",
		Body:        "Hello",
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	ms.messages[msgA.ID] = msgA

	router := setupMessageTestRouter(userA, ms, ds)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/messages/%d/read", msgA.ID), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when marking own message as read, got %d", rr.Code)
	}
}

func TestHandleDeleteMessage_OwnMessage_Succeeds(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}

	ms := newMockMessageStore()
	ds := newMockDomainStore()

	domainA := &models.Domain{
		ID:                10,
		PublicID:          uuid.New(),
		UserID:            userA.ID,
		Name:              "a-domain.com",
		VerificationToken: "tok-a",
		Verified:          true,
	}
	ds.addDomain(domainA)

	msgA := &models.Message{
		ID:          1,
		PublicID:    uuid.New(),
		DomainID:    domainA.ID,
		SenderName:  "Sender",
		SenderEmail: "sender@example.com",
		Body:        "Hello",
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	ms.messages[msgA.ID] = msgA

	router := setupMessageTestRouter(userA, ms, ds)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/messages/%d", msgA.ID), nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when deleting own message, got %d", rr.Code)
	}
}
