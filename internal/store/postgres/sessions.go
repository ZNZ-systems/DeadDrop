package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/znz-systems/deaddrop/internal/models"
)

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) CreateSession(ctx context.Context, token string, userID int64, expiresAt interface{}) (*models.Session, error) {
	exp, ok := expiresAt.(time.Time)
	if !ok {
		exp = time.Now().Add(72 * time.Hour)
	}

	session := &models.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: exp,
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		session.Token, session.UserID, session.ExpiresAt,
	).Scan(&session.ID, &session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *SessionStore) GetSessionByToken(ctx context.Context, token string) (*models.Session, error) {
	session := &models.Session{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, token, user_id, expires_at, created_at
		 FROM sessions WHERE token = $1 AND expires_at > NOW()`,
		token,
	).Scan(&session.ID, &session.Token, &session.UserID, &session.ExpiresAt, &session.CreatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (s *SessionStore) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= NOW()`)
	return err
}
