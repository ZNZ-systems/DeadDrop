package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type InboundEmailStore struct {
	db *sql.DB
}

func NewInboundEmailStore(db *sql.DB) *InboundEmailStore {
	return &InboundEmailStore{db: db}
}

func (s *InboundEmailStore) CreateInboundEmail(ctx context.Context, params models.InboundEmailCreateParams) (*models.InboundEmail, error) {
	email := &models.InboundEmail{
		PublicID:  uuid.New(),
		UserID:    params.UserID,
		DomainID:  params.DomainID,
		Recipient: params.Recipient,
		Sender:    params.Sender,
		Subject:   params.Subject,
		TextBody:  params.TextBody,
		HTMLBody:  params.HTMLBody,
		MessageID: params.MessageID,
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO inbound_emails
		 (public_id, user_id, domain_id, recipient, sender, subject, text_body, html_body, message_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, is_read, created_at`,
		email.PublicID, email.UserID, email.DomainID, email.Recipient, email.Sender,
		email.Subject, email.TextBody, email.HTMLBody, email.MessageID,
	).Scan(&email.ID, &email.IsRead, &email.CreatedAt)
	if err != nil {
		return nil, err
	}

	return email, nil
}

func (s *InboundEmailStore) ListInboundEmailsByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.InboundEmail, error) {
	return s.SearchInboundEmailsByUserID(ctx, userID, models.InboundEmailQuery{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *InboundEmailStore) SearchInboundEmailsByUserID(ctx context.Context, userID int64, query models.InboundEmailQuery) ([]models.InboundEmail, error) {
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		sb   strings.Builder
		args []interface{}
	)
	sb.WriteString(
		`SELECT ie.id, ie.public_id, ie.user_id, ie.domain_id, d.name, ie.recipient, ie.sender, ie.subject, ie.text_body, ie.html_body, ie.message_id, ie.is_read, ie.created_at
		 FROM inbound_emails ie
		 JOIN domains d ON d.id = ie.domain_id
		 WHERE ie.user_id = $1`,
	)
	args = append(args, userID)

	if q := strings.TrimSpace(query.Q); q != "" {
		args = append(args, "%"+q+"%")
		sb.WriteString(" AND (ie.sender ILIKE $" + itoa(len(args)) + " OR ie.subject ILIKE $" + itoa(len(args)) + ")")
	}
	if domain := strings.TrimSpace(strings.ToLower(query.Domain)); domain != "" {
		args = append(args, domain)
		sb.WriteString(" AND LOWER(d.name) = $" + itoa(len(args)))
	}
	if query.UnreadOnly {
		sb.WriteString(" AND ie.is_read = FALSE")
	}
	if query.From != nil {
		args = append(args, *query.From)
		sb.WriteString(" AND ie.created_at >= $" + itoa(len(args)))
	}
	if query.To != nil {
		args = append(args, *query.To)
		sb.WriteString(" AND ie.created_at < $" + itoa(len(args)))
	}

	args = append(args, limit, offset)
	sb.WriteString(" ORDER BY ie.created_at DESC LIMIT $" + itoa(len(args)-1) + " OFFSET $" + itoa(len(args)))

	rows, err := s.db.QueryContext(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]models.InboundEmail, 0, limit)
	for rows.Next() {
		email, err := scanInboundEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, *email)
	}
	return emails, rows.Err()
}

func (s *InboundEmailStore) GetInboundEmailByID(ctx context.Context, id int64) (*models.InboundEmail, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT ie.id, ie.public_id, ie.user_id, ie.domain_id, d.name, ie.recipient, ie.sender, ie.subject, ie.text_body, ie.html_body, ie.message_id, ie.is_read, ie.created_at
		 FROM inbound_emails ie
		 JOIN domains d ON d.id = ie.domain_id
		 WHERE id = $1`,
		id,
	)
	return scanInboundEmail(row)
}

func (s *InboundEmailStore) CreateInboundEmailRaw(ctx context.Context, params models.InboundEmailRawCreateParams) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO inbound_email_raws (inbound_email_id, raw_source, blob_key)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (inbound_email_id) DO UPDATE
		 SET raw_source = EXCLUDED.raw_source,
		     blob_key = EXCLUDED.blob_key`,
		params.InboundEmailID, params.RawSource, strings.TrimSpace(params.BlobKey),
	)
	return err
}

func (s *InboundEmailStore) CreateInboundEmailAttachment(ctx context.Context, params models.InboundEmailAttachmentCreateParams) (*models.InboundEmailAttachment, error) {
	attachment := &models.InboundEmailAttachment{
		InboundEmailID: params.InboundEmailID,
		FileName:       strings.TrimSpace(params.FileName),
		ContentType:    strings.TrimSpace(params.ContentType),
		BlobKey:        strings.TrimSpace(params.BlobKey),
		Content:        params.Content,
		SizeBytes:      params.SizeBytes,
	}
	if attachment.SizeBytes <= 0 {
		attachment.SizeBytes = int64(len(params.Content))
	}
	if attachment.FileName == "" {
		attachment.FileName = "attachment"
	}
	if attachment.ContentType == "" {
		attachment.ContentType = "application/octet-stream"
	}

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO inbound_email_attachments (inbound_email_id, file_name, content_type, size_bytes, blob_key, content)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		attachment.InboundEmailID, attachment.FileName, attachment.ContentType, attachment.SizeBytes, attachment.BlobKey, attachment.Content,
	).Scan(&attachment.ID, &attachment.CreatedAt)
	if err != nil {
		return nil, err
	}
	return attachment, nil
}

func (s *InboundEmailStore) ListInboundEmailAttachmentsByEmailID(ctx context.Context, inboundEmailID int64) ([]models.InboundEmailAttachment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, inbound_email_id, file_name, content_type, size_bytes, blob_key, created_at
		 FROM inbound_email_attachments
		 WHERE inbound_email_id = $1
		 ORDER BY id ASC`,
		inboundEmailID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]models.InboundEmailAttachment, 0, 8)
	for rows.Next() {
		var a models.InboundEmailAttachment
		if err := rows.Scan(&a.ID, &a.InboundEmailID, &a.FileName, &a.ContentType, &a.SizeBytes, &a.BlobKey, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, rows.Err()
}

func (s *InboundEmailStore) GetInboundEmailAttachmentByID(ctx context.Context, attachmentID int64) (*models.InboundEmailAttachment, error) {
	var a models.InboundEmailAttachment
	err := s.db.QueryRowContext(ctx,
		`SELECT id, inbound_email_id, file_name, content_type, size_bytes, blob_key, content, created_at
		 FROM inbound_email_attachments
		 WHERE id = $1`,
		attachmentID,
	).Scan(&a.ID, &a.InboundEmailID, &a.FileName, &a.ContentType, &a.SizeBytes, &a.BlobKey, &a.Content, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *InboundEmailStore) MarkInboundEmailRead(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE inbound_emails SET is_read = TRUE WHERE id = $1`, id)
	return err
}

func (s *InboundEmailStore) DeleteInboundEmail(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM inbound_emails WHERE id = $1`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanInboundEmail(scanner rowScanner) (*models.InboundEmail, error) {
	var email models.InboundEmail
	if err := scanner.Scan(
		&email.ID, &email.PublicID, &email.UserID, &email.DomainID, &email.DomainName,
		&email.Recipient, &email.Sender, &email.Subject, &email.TextBody, &email.HTMLBody,
		&email.MessageID, &email.IsRead, &email.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &email, nil
}

func itoa(n int) string {
	// tiny helper to avoid pulling fmt into query assembly.
	return strconv.Itoa(n)
}
