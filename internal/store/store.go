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

type InboundEmailStore interface {
	CreateInboundEmail(ctx context.Context, params models.InboundEmailCreateParams) (*models.InboundEmail, error)
	ListInboundEmailsByUserID(ctx context.Context, userID int64, limit, offset int) ([]models.InboundEmail, error)
	SearchInboundEmailsByUserID(ctx context.Context, userID int64, query models.InboundEmailQuery) ([]models.InboundEmail, error)
	GetInboundEmailByID(ctx context.Context, id int64) (*models.InboundEmail, error)
	CreateInboundEmailRaw(ctx context.Context, inboundEmailID int64, rawSource string) error
	CreateInboundEmailAttachment(ctx context.Context, params models.InboundEmailAttachmentCreateParams) (*models.InboundEmailAttachment, error)
	ListInboundEmailAttachmentsByEmailID(ctx context.Context, inboundEmailID int64) ([]models.InboundEmailAttachment, error)
	GetInboundEmailAttachmentByID(ctx context.Context, attachmentID int64) (*models.InboundEmailAttachment, error)
	MarkInboundEmailRead(ctx context.Context, id int64) error
	DeleteInboundEmail(ctx context.Context, id int64) error
}

type InboundDomainConfigStore interface {
	UpsertInboundDomainConfig(ctx context.Context, domainID int64, mxTarget string) (*models.InboundDomainConfig, error)
	GetInboundDomainConfigByDomainID(ctx context.Context, domainID int64) (*models.InboundDomainConfig, error)
	UpdateInboundDomainVerification(ctx context.Context, domainID int64, verified bool, lastError string) error
}

type InboundRecipientRuleStore interface {
	CreateInboundRecipientRule(ctx context.Context, domainID int64, ruleType, pattern, action string) (*models.InboundRecipientRule, error)
	ListInboundRecipientRulesByDomainID(ctx context.Context, domainID int64) ([]models.InboundRecipientRule, error)
	DeleteInboundRecipientRule(ctx context.Context, domainID, ruleID int64) error
}
