package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           int64
	PublicID     uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        int64
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Domain struct {
	ID                int64
	PublicID          uuid.UUID
	UserID            int64
	Name              string
	VerificationToken string
	Verified          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Message struct {
	ID          int64
	PublicID    uuid.UUID
	DomainID   int64
	SenderName  string
	SenderEmail string
	Body        string
	IsRead      bool
	CreatedAt   time.Time
}

type Mailbox struct {
	ID          int64
	PublicID    uuid.UUID
	UserID      int64
	DomainID    int64
	Name        string
	FromAddress string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StreamType string

const (
	StreamTypeForm  StreamType = "form"
	StreamTypeEmail StreamType = "email"
)

type Stream struct {
	ID        int64
	PublicID  uuid.UUID
	MailboxID int64
	Type      StreamType
	Address   string
	WidgetID  uuid.UUID
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConversationStatus string

const (
	ConversationOpen   ConversationStatus = "open"
	ConversationClosed ConversationStatus = "closed"
)

type Conversation struct {
	ID        int64
	PublicID  uuid.UUID
	MailboxID int64
	StreamID  int64
	Subject   string
	Status    ConversationStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MessageDirection string

const (
	MessageInbound  MessageDirection = "inbound"
	MessageOutbound MessageDirection = "outbound"
)

type ConversationMessage struct {
	ID             int64
	PublicID       uuid.UUID
	ConversationID int64
	Direction      MessageDirection
	SenderAddress  string
	SenderName     string
	Body           string
	CreatedAt      time.Time
}
