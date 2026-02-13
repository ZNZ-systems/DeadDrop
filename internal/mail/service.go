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

// SendReply sends a reply email from a mailbox. Implements conversation.Sender.
func (s *Service) SendReply(ctx context.Context, to, fromAddress, fromName, subject, body string) error {
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}
	return s.client.SendFrom(from, to, subject, body)
}

// NotifyNewConversation sends an email notification when a new conversation is started.
// Implements conversation.Notifier.
func (s *Service) NotifyNewConversation(ctx context.Context, mailbox *models.Mailbox, conv *models.Conversation, msg *models.ConversationMessage) error {
	user, err := s.users.GetUserByID(ctx, mailbox.UserID)
	if err != nil {
		return fmt.Errorf("mail: failed to look up mailbox owner: %w", err)
	}

	subject := fmt.Sprintf("New conversation in %s", mailbox.Name)
	body := NewConversationNotificationBody(mailbox.Name, msg.SenderName, msg.SenderAddress, msg.Body, conv.Subject)

	return s.client.Send(user.Email, subject, body)
}
