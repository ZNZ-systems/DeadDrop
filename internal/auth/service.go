package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

// Service provides authentication business logic.
type Service struct {
	users    store.UserStore
	sessions store.SessionStore
	maxAge   time.Duration
}

// NewService creates a new auth service with the given stores and session max age in hours.
func NewService(users store.UserStore, sessions store.SessionStore, maxAgeHours int) *Service {
	return &Service{
		users:    users,
		sessions: sessions,
		maxAge:   time.Duration(maxAgeHours) * time.Hour,
	}
}

// Signup registers a new user with the given email and password.
// It validates that email is not empty and password is at least 8 characters.
func (s *Service) Signup(ctx context.Context, email, password string) (*models.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, email, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user by email and password, returning a new session.
func (s *Service) Login(ctx context.Context, email, password string) (*models.Session, error) {
	user, err := s.users.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if err := CheckPassword(user.PasswordHash, password); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	expiresAt := time.Now().Add(s.maxAge)
	session, err := s.sessions.CreateSession(ctx, token, user.ID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// Logout deletes the session identified by the given token.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.DeleteSession(ctx, token)
}

// ValidateSession checks if the given token corresponds to a valid session
// and returns the associated user.
func (s *Service) ValidateSession(ctx context.Context, token string) (*models.User, error) {
	session, err := s.sessions.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, errors.New("invalid session")
	}

	user, err := s.users.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}
