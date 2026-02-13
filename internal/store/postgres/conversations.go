package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type ConversationStore struct {
	db *sql.DB
}

func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) CreateConversation(ctx context.Context, mailboxID, streamID int64, subject string) (*models.Conversation, error) {
	c := &models.Conversation{
		PublicID:  uuid.New(),
		MailboxID: mailboxID,
		StreamID:  streamID,
		Subject:   subject,
		Status:    models.ConversationOpen,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO conversations (public_id, mailbox_id, stream_id, subject)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, status, created_at, updated_at`,
		c.PublicID, c.MailboxID, c.StreamID, c.Subject,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationByID(ctx context.Context, id int64) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE id = $1`, id,
	).Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE public_id = $1`, publicID,
	).Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationsByMailboxID(ctx context.Context, mailboxID int64, limit, offset int) ([]models.Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE mailbox_id = $1
		 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		mailboxID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convos = append(convos, c)
	}
	return convos, rows.Err()
}

func (s *ConversationStore) UpdateConversationStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE conversations SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id)
	return err
}

func (s *ConversationStore) CountOpenByMailboxID(ctx context.Context, mailboxID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM conversations WHERE mailbox_id = $1 AND status = 'open'`,
		mailboxID).Scan(&count)
	return count, err
}

func (s *ConversationStore) CreateMessage(ctx context.Context, conversationID int64, direction, senderAddress, senderName, body string) (*models.ConversationMessage, error) {
	m := &models.ConversationMessage{
		PublicID:       uuid.New(),
		ConversationID: conversationID,
		Direction:      models.MessageDirection(direction),
		SenderAddress:  senderAddress,
		SenderName:     senderName,
		Body:           body,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO conversation_messages (public_id, conversation_id, direction, sender_address, sender_name, body)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		m.PublicID, m.ConversationID, string(m.Direction), m.SenderAddress, m.SenderName, m.Body,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Touch the conversation's updated_at
	_, _ = s.db.ExecContext(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID)

	return m, nil
}

func (s *ConversationStore) GetMessagesByConversationID(ctx context.Context, conversationID int64) ([]models.ConversationMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, conversation_id, direction, sender_address, sender_name, body, created_at
		 FROM conversation_messages WHERE conversation_id = $1
		 ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.ConversationMessage
	for rows.Next() {
		var m models.ConversationMessage
		if err := rows.Scan(&m.ID, &m.PublicID, &m.ConversationID, &m.Direction, &m.SenderAddress, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
