package conversation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

var (
	ErrStreamDisabled     = errors.New("stream is disabled")
	ErrConversationClosed = errors.New("conversation is closed")
)

// Notifier sends notifications when new conversations arrive.
type Notifier interface {
	NotifyNewConversation(ctx context.Context, mailbox *models.Mailbox, conv *models.Conversation, msg *models.ConversationMessage) error
}

type NoopNotifier struct{}

func (n *NoopNotifier) NotifyNewConversation(_ context.Context, _ *models.Mailbox, _ *models.Conversation, _ *models.ConversationMessage) error {
	return nil
}

// Sender sends outbound reply emails.
type Sender interface {
	SendReply(ctx context.Context, to, fromAddress, fromName, subject, body string) error
}

type NoopSender struct{}

func (n *NoopSender) SendReply(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

type Service struct {
	conversations store.ConversationStore
	mailboxes     store.MailboxStore
	notifier      Notifier
	sender        Sender
}

func NewService(
	conversations store.ConversationStore,
	mailboxes store.MailboxStore,
	notifier Notifier,
	sender Sender,
) *Service {
	return &Service{
		conversations: conversations,
		mailboxes:     mailboxes,
		notifier:      notifier,
		sender:        sender,
	}
}

// StartConversation creates a new conversation from an inbound message.
// The caller provides the stream directly (already looked up).
func (s *Service) StartConversation(ctx context.Context, stream *models.Stream, subject, senderAddress, senderName, body string) (*models.Conversation, error) {
	if !stream.Enabled {
		return nil, ErrStreamDisabled
	}

	conv, err := s.conversations.CreateConversation(ctx, stream.MailboxID, stream.ID, subject)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	msg, err := s.conversations.CreateMessage(ctx, conv.ID, string(models.MessageInbound), senderAddress, senderName, body)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Fire-and-forget notification
	go func() {
		mb, _ := s.mailboxes.GetMailboxByID(context.Background(), stream.MailboxID)
		if mb != nil {
			_ = s.notifier.NotifyNewConversation(context.Background(), mb, conv, msg)
		}
	}()

	return conv, nil
}

// Reply adds an outbound message to an existing conversation and sends the email.
func (s *Service) Reply(ctx context.Context, conversationID int64, body string) (*models.ConversationMessage, error) {
	conv, err := s.conversations.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if conv.Status == models.ConversationClosed {
		return nil, ErrConversationClosed
	}

	mb, err := s.mailboxes.GetMailboxByID(ctx, conv.MailboxID)
	if err != nil {
		return nil, fmt.Errorf("get mailbox: %w", err)
	}

	// Find the original sender to reply to
	msgs, err := s.conversations.GetMessagesByConversationID(ctx, conv.ID)
	if err != nil || len(msgs) == 0 {
		return nil, fmt.Errorf("no messages in conversation")
	}

	var replyTo string
	for _, m := range msgs {
		if m.Direction == models.MessageInbound && m.SenderAddress != "" {
			replyTo = m.SenderAddress
			break
		}
	}
	if replyTo == "" {
		return nil, errors.New("no inbound sender address to reply to")
	}

	// Send the email
	subject := conv.Subject
	if subject != "" {
		subject = "Re: " + subject
	}
	if err := s.sender.SendReply(ctx, replyTo, mb.FromAddress, mb.Name, subject, body); err != nil {
		return nil, fmt.Errorf("send reply: %w", err)
	}

	msg, err := s.conversations.CreateMessage(ctx, conv.ID, string(models.MessageOutbound), mb.FromAddress, mb.Name, body)
	if err != nil {
		return nil, fmt.Errorf("create outbound message: %w", err)
	}

	return msg, nil
}

// Close marks a conversation as closed.
func (s *Service) Close(ctx context.Context, conversationID int64) error {
	return s.conversations.UpdateConversationStatus(ctx, conversationID, string(models.ConversationClosed))
}

// List returns conversations for a mailbox with pagination.
func (s *Service) List(ctx context.Context, mailboxID int64, limit, offset int) ([]models.Conversation, error) {
	return s.conversations.GetConversationsByMailboxID(ctx, mailboxID, limit, offset)
}

// GetMessages returns all messages in a conversation.
func (s *Service) GetMessages(ctx context.Context, conversationID int64) ([]models.ConversationMessage, error) {
	return s.conversations.GetMessagesByConversationID(ctx, conversationID)
}

// GetByPublicID retrieves a conversation by public UUID.
func (s *Service) GetByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Conversation, error) {
	return s.conversations.GetConversationByPublicID(ctx, publicID)
}

// CountOpen returns the number of open conversations for a mailbox.
func (s *Service) CountOpen(ctx context.Context, mailboxID int64) (int, error) {
	return s.conversations.CountOpenByMailboxID(ctx, mailboxID)
}
