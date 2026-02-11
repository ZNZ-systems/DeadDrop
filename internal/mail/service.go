package mail

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

// Service implements the message.Notifier interface by sending email
// notifications to domain owners when new messages are received.
type Service struct {
	client *SMTPClient
	users  store.UserStore
}

// NewService creates a new mail Service that sends notifications via SMTP.
func NewService(client *SMTPClient, users store.UserStore) *Service {
	return &Service{
		client: client,
		users:  users,
	}
}

// NotifyNewMessage looks up the domain owner and sends them an email notification
// about the new message. This method satisfies the message.Notifier interface.
func (s *Service) NotifyNewMessage(ctx context.Context, domain *models.Domain, msg *models.Message) error {
	user, err := s.users.GetUserByID(ctx, domain.UserID)
	if err != nil {
		return fmt.Errorf("mail: failed to look up domain owner (userID=%d): %w", domain.UserID, err)
	}

	subject := fmt.Sprintf("New message on %s", domain.Name)
	body := NewMessageNotificationBody(domain.Name, msg.SenderName, msg.SenderEmail, msg.Body)

	if err := s.client.Send(user.Email, subject, body); err != nil {
		return fmt.Errorf("mail: failed to send notification to %s: %w", user.Email, err)
	}

	slog.InfoContext(ctx, "sent new-message notification",
		"domain", domain.Name,
		"recipient", user.Email,
		"message_id", msg.PublicID,
	)

	return nil
}
