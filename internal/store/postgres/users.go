package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{
		PublicID:     uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users (public_id, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		user.PublicID, user.Email, user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, email, password_hash, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.PublicID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, email, password_hash, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.PublicID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}
