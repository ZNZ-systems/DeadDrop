package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type MessageStore struct {
	db *sql.DB
}

func NewMessageStore(db *sql.DB) *MessageStore {
	return &MessageStore{db: db}
}

func (s *MessageStore) CreateMessage(ctx context.Context, domainID int64, senderName, senderEmail, body string) (*models.Message, error) {
	msg := &models.Message{
		PublicID:    uuid.New(),
		DomainID:   domainID,
		SenderName:  senderName,
		SenderEmail: senderEmail,
		Body:        body,
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO messages (public_id, domain_id, sender_name, sender_email, body)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, is_read, created_at`,
		msg.PublicID, msg.DomainID, msg.SenderName, msg.SenderEmail, msg.Body,
	).Scan(&msg.ID, &msg.IsRead, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *MessageStore) GetMessagesByDomainID(ctx context.Context, domainID int64, limit, offset int) ([]models.Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, domain_id, sender_name, sender_email, body, is_read, created_at
		 FROM messages WHERE domain_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		domainID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.PublicID, &m.DomainID, &m.SenderName, &m.SenderEmail, &m.Body, &m.IsRead, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *MessageStore) MarkMessageRead(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET is_read = TRUE WHERE id = $1`, id)
	return err
}

func (s *MessageStore) DeleteMessage(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

func (s *MessageStore) CountUnreadByDomainID(ctx context.Context, domainID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages WHERE domain_id = $1 AND is_read = FALSE`,
		domainID,
	).Scan(&count)
	return count, err
}
