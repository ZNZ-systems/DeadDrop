package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type UserStore interface {
	CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, token string, userID int64, expiresAt interface{}) (*models.Session, error)
	GetSessionByToken(ctx context.Context, token string) (*models.Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteExpiredSessions(ctx context.Context) error
}

type DomainStore interface {
	CreateDomain(ctx context.Context, userID int64, name, verificationToken string) (*models.Domain, error)
	GetDomainsByUserID(ctx context.Context, userID int64) ([]models.Domain, error)
	GetDomainByID(ctx context.Context, id int64) (*models.Domain, error)
	GetDomainByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Domain, error)
	GetDomainByName(ctx context.Context, name string) (*models.Domain, error)
	MarkDomainVerified(ctx context.Context, id int64) error
	DeleteDomain(ctx context.Context, id int64) error
}

type MessageStore interface {
	CreateMessage(ctx context.Context, domainID int64, senderName, senderEmail, body string) (*models.Message, error)
	GetMessageByID(ctx context.Context, id int64) (*models.Message, error)
	GetMessagesByDomainID(ctx context.Context, domainID int64, limit, offset int) ([]models.Message, error)
	MarkMessageRead(ctx context.Context, id int64) error
	DeleteMessage(ctx context.Context, id int64) error
	CountUnreadByDomainID(ctx context.Context, domainID int64) (int, error)
}

type MailboxStore interface {
	CreateMailbox(ctx context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error)
	GetMailboxesByUserID(ctx context.Context, userID int64) ([]models.Mailbox, error)
	GetMailboxByID(ctx context.Context, id int64) (*models.Mailbox, error)
	GetMailboxByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Mailbox, error)
	DeleteMailbox(ctx context.Context, id int64) error
}

type StreamStore interface {
	CreateStream(ctx context.Context, mailboxID int64, streamType string, address string, widgetID uuid.UUID) (*models.Stream, error)
	GetStreamsByMailboxID(ctx context.Context, mailboxID int64) ([]models.Stream, error)
	GetStreamByWidgetID(ctx context.Context, widgetID uuid.UUID) (*models.Stream, error)
	GetStreamByAddress(ctx context.Context, address string) (*models.Stream, error)
	DeleteStream(ctx context.Context, id int64) error
}

type ConversationStore interface {
	CreateConversation(ctx context.Context, mailboxID, streamID int64, subject string) (*models.Conversation, error)
	GetConversationByID(ctx context.Context, id int64) (*models.Conversation, error)
	GetConversationByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Conversation, error)
	GetConversationsByMailboxID(ctx context.Context, mailboxID int64, limit, offset int) ([]models.Conversation, error)
	UpdateConversationStatus(ctx context.Context, id int64, status string) error
	CountOpenByMailboxID(ctx context.Context, mailboxID int64) (int, error)
	CreateMessage(ctx context.Context, conversationID int64, direction, senderAddress, senderName, body string) (*models.ConversationMessage, error)
	GetMessagesByConversationID(ctx context.Context, conversationID int64) ([]models.ConversationMessage, error)
}
