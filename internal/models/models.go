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
