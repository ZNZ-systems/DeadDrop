package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type DomainStore struct {
	db *sql.DB
}

func NewDomainStore(db *sql.DB) *DomainStore {
	return &DomainStore{db: db}
}

func (s *DomainStore) CreateDomain(ctx context.Context, userID int64, name, verificationToken string) (*models.Domain, error) {
	domain := &models.Domain{
		PublicID:          uuid.New(),
		UserID:            userID,
		Name:              name,
		VerificationToken: verificationToken,
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO domains (public_id, user_id, name, verification_token)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, verified, created_at, updated_at`,
		domain.PublicID, domain.UserID, domain.Name, domain.VerificationToken,
	).Scan(&domain.ID, &domain.Verified, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return domain, nil
}

func (s *DomainStore) GetDomainsByUserID(ctx context.Context, userID int64) ([]models.Domain, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, user_id, name, verification_token, verified, created_at, updated_at
		 FROM domains WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []models.Domain
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.PublicID, &d.UserID, &d.Name, &d.VerificationToken, &d.Verified, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

func (s *DomainStore) GetDomainByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Domain, error) {
	d := &models.Domain{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, name, verification_token, verified, created_at, updated_at
		 FROM domains WHERE public_id = $1`,
		publicID,
	).Scan(&d.ID, &d.PublicID, &d.UserID, &d.Name, &d.VerificationToken, &d.Verified, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DomainStore) GetDomainByName(ctx context.Context, name string) (*models.Domain, error) {
	d := &models.Domain{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, name, verification_token, verified, created_at, updated_at
		 FROM domains WHERE name = $1`,
		name,
	).Scan(&d.ID, &d.PublicID, &d.UserID, &d.Name, &d.VerificationToken, &d.Verified, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DomainStore) MarkDomainVerified(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE domains SET verified = TRUE, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *DomainStore) DeleteDomain(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM domains WHERE id = $1`, id)
	return err
}
