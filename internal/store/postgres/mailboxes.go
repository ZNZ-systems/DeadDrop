package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type MailboxStore struct {
	db *sql.DB
}

func NewMailboxStore(db *sql.DB) *MailboxStore {
	return &MailboxStore{db: db}
}

func (s *MailboxStore) CreateMailbox(ctx context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error) {
	m := &models.Mailbox{
		PublicID:    uuid.New(),
		UserID:      userID,
		DomainID:    domainID,
		Name:        name,
		FromAddress: fromAddress,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO mailboxes (public_id, user_id, domain_id, name, from_address)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		m.PublicID, m.UserID, m.DomainID, m.Name, m.FromAddress,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) GetMailboxesByUserID(ctx context.Context, userID int64) ([]models.Mailbox, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []models.Mailbox
	for rows.Next() {
		var m models.Mailbox
		if err := rows.Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		mailboxes = append(mailboxes, m)
	}
	return mailboxes, rows.Err()
}

func (s *MailboxStore) GetMailboxByID(ctx context.Context, id int64) (*models.Mailbox, error) {
	m := &models.Mailbox{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE id = $1`, id,
	).Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) GetMailboxByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Mailbox, error) {
	m := &models.Mailbox{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE public_id = $1`, publicID,
	).Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) DeleteMailbox(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mailboxes WHERE id = $1`, id)
	return err
}
