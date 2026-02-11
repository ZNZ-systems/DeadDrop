package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockUserStore struct {
	users       map[string]*models.User
	usersById   map[int64]*models.User
	createErr   error
	nextID      int64
}

func newMockUserStore() *mockUserStore {
	return &mockUserStore{
		users:     make(map[string]*models.User),
		usersById: make(map[int64]*models.User),
		nextID:    1,
	}
}

func (m *mockUserStore) CreateUser(_ context.Context, email, passwordHash string) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if _, exists := m.users[email]; exists {
		return nil, errors.New("user already exists")
	}
	u := &models.User{
		ID:           m.nextID,
		PublicID:     uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.nextID++
	m.users[email] = u
	m.usersById[u.ID] = u
	return u, nil
}

func (m *mockUserStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserStore) GetUserByID(_ context.Context, id int64) (*models.User, error) {
	u, ok := m.usersById[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

type mockSessionStore struct {
	sessions  map[string]*models.Session
	createErr error
	nextID    int64
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]*models.Session),
		nextID:   1,
	}
}

func (m *mockSessionStore) CreateSession(_ context.Context, token string, userID int64, expiresAt interface{}) (*models.Session, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	exp, ok := expiresAt.(time.Time)
	if !ok {
		return nil, errors.New("expiresAt must be time.Time")
	}
	s := &models.Session{
		ID:        m.nextID,
		Token:     token,
		UserID:    userID,
		ExpiresAt: exp,
		CreatedAt: time.Now(),
	}
	m.nextID++
	m.sessions[token] = s
	return s, nil
}

func (m *mockSessionStore) GetSessionByToken(_ context.Context, token string) (*models.Session, error) {
	s, ok := m.sessions[token]
	if !ok {
		return nil, errors.New("session not found")
	}
	return s, nil
}

func (m *mockSessionStore) DeleteSession(_ context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockSessionStore) DeleteExpiredSessions(_ context.Context) error {
	now := time.Now()
	for token, s := range m.sessions {
		if s.ExpiresAt.Before(now) {
			delete(m.sessions, token)
		}
	}
	return nil
}

// --- Tests ---

func TestSignup_Success(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	user, err := svc.Signup(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}
	// Verify we can check the password against the hash
	if err := CheckPassword(user.PasswordHash, "password123"); err != nil {
		t.Error("password hash should match original password")
	}
}

func TestSignup_EmptyEmail(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, err := svc.Signup(context.Background(), "", "password123")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
	if err.Error() != "email is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSignup_ShortPassword(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, err := svc.Signup(context.Background(), "test@example.com", "short")
	if err == nil {
		t.Fatal("expected error for short password")
	}
	if err.Error() != "password must be at least 8 characters" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, err := svc.Signup(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("first signup failed: %v", err)
	}

	_, err = svc.Signup(context.Background(), "test@example.com", "password456")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestLogin_Success(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	// Create user first
	_, err := svc.Signup(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	session, err := svc.Login(context.Background(), "test@example.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if session.Token == "" {
		t.Error("expected session token to be set")
	}
	if session.UserID != 1 {
		t.Errorf("expected user ID 1, got %d", session.UserID)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")

	_, err := svc.Login(context.Background(), "test@example.com", "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if err.Error() != "invalid email or password" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, err := svc.Login(context.Background(), "nobody@example.com", "password123")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestLogout(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Login(context.Background(), "test@example.com", "password123")

	err := svc.Logout(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Session should no longer be valid
	_, err = svc.ValidateSession(context.Background(), session.Token)
	if err == nil {
		t.Error("expected session to be invalid after logout")
	}
}

func TestValidateSession_Valid(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, _ = svc.Signup(context.Background(), "test@example.com", "password123")
	session, _ := svc.Login(context.Background(), "test@example.com", "password123")

	user, err := svc.ValidateSession(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("validate session failed: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", user.Email)
	}
}

func TestValidateSession_InvalidToken(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionStore()
	svc := NewService(users, sessions, 72)

	_, err := svc.ValidateSession(context.Background(), "bogus-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestHashPassword_And_CheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if err := CheckPassword(hash, "mypassword"); err != nil {
		t.Error("CheckPassword should succeed with correct password")
	}
	if err := CheckPassword(hash, "wrongpassword"); err == nil {
		t.Error("CheckPassword should fail with wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if len(token1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("expected 64-char token, got %d chars", len(token1))
	}

	token2, _ := GenerateToken()
	if token1 == token2 {
		t.Error("expected unique tokens")
	}
}
