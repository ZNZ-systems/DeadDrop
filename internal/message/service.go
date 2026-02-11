package message

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

// Sentinel errors returned by Service methods.
var (
	ErrDomainNotFound    = errors.New("domain not found")
	ErrDomainNotVerified = errors.New("domain not verified")
)

// Notifier sends notifications when new messages arrive.
type Notifier interface {
	NotifyNewMessage(ctx context.Context, domain *models.Domain, msg *models.Message) error
}

// NoopNotifier is a Notifier that does nothing.
type NoopNotifier struct{}

func (n *NoopNotifier) NotifyNewMessage(_ context.Context, _ *models.Domain, _ *models.Message) error {
	return nil
}

// Service provides message business logic.
type Service struct {
	messages store.MessageStore
	domains  store.DomainStore
	notifier Notifier
}

// NewService creates a new message Service.
func NewService(messages store.MessageStore, domains store.DomainStore, notifier Notifier) *Service {
	return &Service{
		messages: messages,
		domains:  domains,
		notifier: notifier,
	}
}

// Submit creates a new message for the domain identified by its public ID.
// It verifies that the domain exists and is verified before accepting the message.
// Notification is fire-and-forget: errors are logged but not returned.
func (s *Service) Submit(ctx context.Context, domainPublicID uuid.UUID, senderName, senderEmail, body string) error {
	domain, err := s.domains.GetDomainByPublicID(ctx, domainPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDomainNotFound
		}
		return fmt.Errorf("looking up domain: %w", err)
	}

	if !domain.Verified {
		return ErrDomainNotVerified
	}

	msg, err := s.messages.CreateMessage(ctx, domain.ID, senderName, senderEmail, body)
	if err != nil {
		return fmt.Errorf("creating message: %w", err)
	}

	// Fire-and-forget notification; log any error.
	go func() {
		if notifyErr := s.notifier.NotifyNewMessage(ctx, domain, msg); notifyErr != nil {
			slog.Error("failed to send new-message notification",
				"domain_id", domain.ID,
				"message_id", msg.ID,
				"error", notifyErr,
			)
		}
	}()

	return nil
}

// List returns messages for a domain with pagination.
func (s *Service) List(ctx context.Context, domainID int64, limit, offset int) ([]models.Message, error) {
	msgs, err := s.messages.GetMessagesByDomainID(ctx, domainID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	return msgs, nil
}

// MarkRead marks a single message as read.
func (s *Service) MarkRead(ctx context.Context, messageID int64) error {
	if err := s.messages.MarkMessageRead(ctx, messageID); err != nil {
		return fmt.Errorf("marking message read: %w", err)
	}
	return nil
}

// Delete removes a message.
func (s *Service) Delete(ctx context.Context, messageID int64) error {
	if err := s.messages.DeleteMessage(ctx, messageID); err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}
	return nil
}

// CountUnread returns the number of unread messages for a domain.
func (s *Service) CountUnread(ctx context.Context, domainID int64) (int, error) {
	count, err := s.messages.CountUnreadByDomainID(ctx, domainID)
	if err != nil {
		return 0, fmt.Errorf("counting unread messages: %w", err)
	}
	return count, nil
}
