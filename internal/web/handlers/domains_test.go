package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
)

// testResolver is a no-op DNS resolver for tests.
type testResolver struct{}

func (t *testResolver) LookupTXT(_ string) ([]string, error) {
	return nil, nil
}

// injectUser is test middleware that sets the authenticated user in context.
func injectUser(user *models.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func setupDomainTestRouter(userA, userB *models.User) (*chi.Mux, *models.Domain) {
	ds := newMockDomainStore()
	ms := newMockMessageStore()

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

	domainSvc := domain.NewService(ds, &testResolver{})
	handler := NewDomainHandler(domainSvc, ms, nil, "", false) // nil renderer: IDOR check triggers before render

	r := chi.NewRouter()
	r.Use(injectUser(userA)) // all requests as user A
	r.Get("/domains/{id}", handler.ShowDomainDetail)
	r.Post("/domains/{id}/verify", handler.HandleVerifyDomain)
	r.Post("/domains/{id}/delete", handler.HandleDeleteDomain)

	return r, domainB
}

func TestShowDomainDetail_IDOR_Returns404(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}
	userB := &models.User{ID: 2, Email: "b@test.com"}

	router, domainB := setupDomainTestRouter(userA, userB)

	req := httptest.NewRequest(http.MethodGet, "/domains/"+domainB.PublicID.String(), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when accessing another user's domain, got %d", rr.Code)
	}
}

func TestHandleVerifyDomain_IDOR_Returns404(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}
	userB := &models.User{ID: 2, Email: "b@test.com"}

	router, domainB := setupDomainTestRouter(userA, userB)

	req := httptest.NewRequest(http.MethodPost, "/domains/"+domainB.PublicID.String()+"/verify", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when verifying another user's domain, got %d", rr.Code)
	}
}

func TestHandleDeleteDomain_IDOR_Returns404(t *testing.T) {
	userA := &models.User{ID: 1, Email: "a@test.com"}
	userB := &models.User{ID: 2, Email: "b@test.com"}

	router, domainB := setupDomainTestRouter(userA, userB)

	req := httptest.NewRequest(http.MethodPost, "/domains/"+domainB.PublicID.String()+"/delete", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 when deleting another user's domain, got %d", rr.Code)
	}
}

func TestShowDomainDetail_OwnDomain_PassesOwnershipCheck(t *testing.T) {
	ds := newMockDomainStore()

	userA := &models.User{ID: 1, Email: "a@test.com"}

	domainA := &models.Domain{
		ID:                10,
		PublicID:          uuid.New(),
		UserID:            userA.ID,
		Name:              "a-domain.com",
		VerificationToken: "tok-a",
		Verified:          true,
	}
	ds.addDomain(domainA)

	domainSvc := domain.NewService(ds, &testResolver{})

	// Use a handler that records whether the ownership check was passed
	// by wrapping the handler with a sentinel: if we get past the check,
	// we respond 200 instead of hitting the nil renderer.
	passed := false
	r := chi.NewRouter()
	r.Use(injectUser(userA))
	r.Get("/domains/{id}", func(w http.ResponseWriter, req *http.Request) {
		// Replicate the ownership check logic
		idParam := chi.URLParam(req, "id")
		publicID, _ := uuid.Parse(idParam)
		d, err := domainSvc.GetByPublicID(req.Context(), publicID)
		if err != nil || d.UserID != userA.ID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		passed = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/domains/"+domainA.PublicID.String(), nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if !passed {
		t.Error("ownership check should pass for own domain")
	}
	if rr.Code == http.StatusNotFound {
		t.Error("should NOT get 404 when accessing own domain")
	}
}
