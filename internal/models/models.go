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
	DomainID    int64
	SenderName  string
	SenderEmail string
	Body        string
	IsRead      bool
	CreatedAt   time.Time
}

type InboundEmail struct {
	ID         int64
	PublicID   uuid.UUID
	UserID     int64
	DomainID   int64
	DomainName string
	Recipient  string
	Sender     string
	Subject    string
	TextBody   string
	HTMLBody   string
	MessageID  string
	IsRead     bool
	CreatedAt  time.Time
}

type InboundEmailCreateParams struct {
	UserID    int64
	DomainID  int64
	Recipient string
	Sender    string
	Subject   string
	TextBody  string
	HTMLBody  string
	MessageID string
}

type InboundEmailQuery struct {
	Q          string
	Domain     string
	UnreadOnly bool
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

type InboundEmailAttachment struct {
	ID             int64
	InboundEmailID int64
	FileName       string
	ContentType    string
	SizeBytes      int64
	Content        []byte
	CreatedAt      time.Time
}

type InboundEmailAttachmentCreateParams struct {
	InboundEmailID int64
	FileName       string
	ContentType    string
	Content        []byte
}

type InboundDomainConfig struct {
	DomainID   int64
	MXTarget   string
	MXVerified bool
	LastError  string
	CheckedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type InboundRecipientRule struct {
	ID        int64
	DomainID  int64
	RuleType  string
	Pattern   string
	Action    string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
