package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type StreamStore struct {
	db *sql.DB
}

func NewStreamStore(db *sql.DB) *StreamStore {
	return &StreamStore{db: db}
}

func (s *StreamStore) CreateStream(ctx context.Context, mailboxID int64, streamType string, address string, widgetID uuid.UUID) (*models.Stream, error) {
	st := &models.Stream{
		PublicID:  uuid.New(),
		MailboxID: mailboxID,
		Type:      models.StreamType(streamType),
		Address:   address,
		WidgetID:  widgetID,
		Enabled:   true,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO streams (public_id, mailbox_id, type, address, widget_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, enabled, created_at, updated_at`,
		st.PublicID, st.MailboxID, string(st.Type), st.Address, st.WidgetID,
	).Scan(&st.ID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) GetStreamsByMailboxID(ctx context.Context, mailboxID int64) ([]models.Stream, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE mailbox_id = $1 ORDER BY created_at`, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []models.Stream
	for rows.Next() {
		var st models.Stream
		if err := rows.Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		streams = append(streams, st)
	}
	return streams, rows.Err()
}

func (s *StreamStore) GetStreamByWidgetID(ctx context.Context, widgetID uuid.UUID) (*models.Stream, error) {
	st := &models.Stream{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE widget_id = $1`, widgetID,
	).Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) GetStreamByAddress(ctx context.Context, address string) (*models.Stream, error) {
	st := &models.Stream{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE address = $1 AND type = 'email'`, address,
	).Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) DeleteStream(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM streams WHERE id = $1`, id)
	return err
}
