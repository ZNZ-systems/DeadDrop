# Mailbox Flows Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform DeadDrop from a contact-form collector into a self-hosted mailbox system with inbound SMTP, conversation threading, and dashboard-based replies.

**Architecture:** New Mailbox, Stream, and Conversation entities sit alongside the existing Domain model. A bundled Go SMTP server (go-smtp) handles inbound email. Outbound replies use the existing SMTP client with Resend-style from-address based on verified domains.

**Tech Stack:** Go 1.25, Chi router, Postgres, HTMX, `github.com/emersion/go-smtp`, `github.com/emersion/go-message`, `database/sql` with `lib/pq`.

---

### Task 1: Add New Models

**Files:**
- Modify: `internal/models/models.go`

**Step 1: Add Mailbox, Stream, Conversation, and Message models**

Add these structs to `internal/models/models.go` after the existing `Message` struct. Note: the existing `Message` struct stays for now (used by existing code). The new `ConversationMessage` replaces it going forward.

```go
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
	Address   string    // email address for email streams
	WidgetID  uuid.UUID // public widget ID for form streams
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
```

**Step 2: Verify the file compiles**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/models/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add Mailbox, Stream, Conversation, ConversationMessage models"
```

---

### Task 2: Create Database Migrations

**Files:**
- Create: `migrations/005_create_mailboxes.up.sql`
- Create: `migrations/005_create_mailboxes.down.sql`
- Create: `migrations/006_create_streams.up.sql`
- Create: `migrations/006_create_streams.down.sql`
- Create: `migrations/007_create_conversations.up.sql`
- Create: `migrations/007_create_conversations.down.sql`

**Step 1: Write mailboxes migration**

`migrations/005_create_mailboxes.up.sql`:
```sql
CREATE TABLE mailboxes (
    id           BIGSERIAL PRIMARY KEY,
    public_id    UUID NOT NULL UNIQUE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id    BIGINT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    from_address TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX idx_mailboxes_domain_id ON mailboxes(domain_id);
CREATE INDEX idx_mailboxes_public_id ON mailboxes(public_id);
```

`migrations/005_create_mailboxes.down.sql`:
```sql
DROP TABLE IF EXISTS mailboxes;
```

**Step 2: Write streams migration**

`migrations/006_create_streams.up.sql`:
```sql
CREATE TABLE streams (
    id         BIGSERIAL PRIMARY KEY,
    public_id  UUID NOT NULL UNIQUE,
    mailbox_id BIGINT NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('form', 'email')),
    address    TEXT NOT NULL DEFAULT '',
    widget_id  UUID UNIQUE,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_streams_mailbox_id ON streams(mailbox_id);
CREATE INDEX idx_streams_widget_id ON streams(widget_id);
CREATE INDEX idx_streams_address ON streams(address) WHERE address != '';
```

`migrations/006_create_streams.down.sql`:
```sql
DROP TABLE IF EXISTS streams;
```

**Step 3: Write conversations + conversation_messages migration**

`migrations/007_create_conversations.up.sql`:
```sql
CREATE TABLE conversations (
    id         BIGSERIAL PRIMARY KEY,
    public_id  UUID NOT NULL UNIQUE,
    mailbox_id BIGINT NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    stream_id  BIGINT NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    subject    TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_mailbox_id ON conversations(mailbox_id, created_at DESC);
CREATE INDEX idx_conversations_status ON conversations(mailbox_id, status);

CREATE TABLE conversation_messages (
    id              BIGSERIAL PRIMARY KEY,
    public_id       UUID NOT NULL UNIQUE,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    direction       TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    sender_address  TEXT NOT NULL DEFAULT '',
    sender_name     TEXT NOT NULL DEFAULT '',
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conv_messages_conversation ON conversation_messages(conversation_id, created_at ASC);
```

`migrations/007_create_conversations.down.sql`:
```sql
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations;
```

**Step 4: Verify migrations compile (Go embed)**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./migrations/`
Expected: No errors

**Step 5: Commit**

```bash
git add migrations/005_create_mailboxes.up.sql migrations/005_create_mailboxes.down.sql \
       migrations/006_create_streams.up.sql migrations/006_create_streams.down.sql \
       migrations/007_create_conversations.up.sql migrations/007_create_conversations.down.sql
git commit -m "feat: add migrations for mailboxes, streams, conversations"
```

---

### Task 3: Add Store Interfaces

**Files:**
- Modify: `internal/store/store.go`

**Step 1: Write the failing test** — N/A (interfaces only, no logic)

**Step 2: Add MailboxStore, StreamStore, ConversationStore interfaces**

Append to `internal/store/store.go`:

```go
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
```

**Step 3: Verify compilation**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/store/`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/store/store.go
git commit -m "feat: add MailboxStore, StreamStore, ConversationStore interfaces"
```

---

### Task 4: Implement Postgres Stores

**Files:**
- Create: `internal/store/postgres/mailboxes.go`
- Create: `internal/store/postgres/streams.go`
- Create: `internal/store/postgres/conversations.go`

**Step 1: Implement MailboxStore**

Create `internal/store/postgres/mailboxes.go`:

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type MailboxStore struct {
	db *sql.DB
}

func NewMailboxStore(db *sql.DB) *MailboxStore {
	return &MailboxStore{db: db}
}

func (s *MailboxStore) CreateMailbox(ctx context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error) {
	m := &models.Mailbox{
		PublicID:    uuid.New(),
		UserID:      userID,
		DomainID:    domainID,
		Name:        name,
		FromAddress: fromAddress,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO mailboxes (public_id, user_id, domain_id, name, from_address)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		m.PublicID, m.UserID, m.DomainID, m.Name, m.FromAddress,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) GetMailboxesByUserID(ctx context.Context, userID int64) ([]models.Mailbox, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mailboxes []models.Mailbox
	for rows.Next() {
		var m models.Mailbox
		if err := rows.Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		mailboxes = append(mailboxes, m)
	}
	return mailboxes, rows.Err()
}

func (s *MailboxStore) GetMailboxByID(ctx context.Context, id int64) (*models.Mailbox, error) {
	m := &models.Mailbox{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE id = $1`, id,
	).Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) GetMailboxByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Mailbox, error) {
	m := &models.Mailbox{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, user_id, domain_id, name, from_address, created_at, updated_at
		 FROM mailboxes WHERE public_id = $1`, publicID,
	).Scan(&m.ID, &m.PublicID, &m.UserID, &m.DomainID, &m.Name, &m.FromAddress, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *MailboxStore) DeleteMailbox(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mailboxes WHERE id = $1`, id)
	return err
}
```

**Step 2: Implement StreamStore**

Create `internal/store/postgres/streams.go`:

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type StreamStore struct {
	db *sql.DB
}

func NewStreamStore(db *sql.DB) *StreamStore {
	return &StreamStore{db: db}
}

func (s *StreamStore) CreateStream(ctx context.Context, mailboxID int64, streamType string, address string, widgetID uuid.UUID) (*models.Stream, error) {
	st := &models.Stream{
		PublicID:  uuid.New(),
		MailboxID: mailboxID,
		Type:      models.StreamType(streamType),
		Address:   address,
		WidgetID:  widgetID,
		Enabled:   true,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO streams (public_id, mailbox_id, type, address, widget_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, enabled, created_at, updated_at`,
		st.PublicID, st.MailboxID, string(st.Type), st.Address, st.WidgetID,
	).Scan(&st.ID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) GetStreamsByMailboxID(ctx context.Context, mailboxID int64) ([]models.Stream, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE mailbox_id = $1 ORDER BY created_at`, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []models.Stream
	for rows.Next() {
		var st models.Stream
		if err := rows.Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		streams = append(streams, st)
	}
	return streams, rows.Err()
}

func (s *StreamStore) GetStreamByWidgetID(ctx context.Context, widgetID uuid.UUID) (*models.Stream, error) {
	st := &models.Stream{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE widget_id = $1`, widgetID,
	).Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) GetStreamByAddress(ctx context.Context, address string) (*models.Stream, error) {
	st := &models.Stream{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, type, address, widget_id, enabled, created_at, updated_at
		 FROM streams WHERE address = $1 AND type = 'email'`, address,
	).Scan(&st.ID, &st.PublicID, &st.MailboxID, &st.Type, &st.Address, &st.WidgetID, &st.Enabled, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StreamStore) DeleteStream(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM streams WHERE id = $1`, id)
	return err
}
```

**Step 3: Implement ConversationStore**

Create `internal/store/postgres/conversations.go`:

```go
package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

type ConversationStore struct {
	db *sql.DB
}

func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) CreateConversation(ctx context.Context, mailboxID, streamID int64, subject string) (*models.Conversation, error) {
	c := &models.Conversation{
		PublicID:  uuid.New(),
		MailboxID: mailboxID,
		StreamID:  streamID,
		Subject:   subject,
		Status:    models.ConversationOpen,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO conversations (public_id, mailbox_id, stream_id, subject)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, status, created_at, updated_at`,
		c.PublicID, c.MailboxID, c.StreamID, c.Subject,
	).Scan(&c.ID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationByID(ctx context.Context, id int64) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE id = $1`, id,
	).Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Conversation, error) {
	c := &models.Conversation{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE public_id = $1`, publicID,
	).Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConversationStore) GetConversationsByMailboxID(ctx context.Context, mailboxID int64, limit, offset int) ([]models.Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, mailbox_id, stream_id, subject, status, created_at, updated_at
		 FROM conversations WHERE mailbox_id = $1
		 ORDER BY updated_at DESC LIMIT $2 OFFSET $3`,
		mailboxID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convos []models.Conversation
	for rows.Next() {
		var c models.Conversation
		if err := rows.Scan(&c.ID, &c.PublicID, &c.MailboxID, &c.StreamID, &c.Subject, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convos = append(convos, c)
	}
	return convos, rows.Err()
}

func (s *ConversationStore) UpdateConversationStatus(ctx context.Context, id int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE conversations SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id)
	return err
}

func (s *ConversationStore) CountOpenByMailboxID(ctx context.Context, mailboxID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM conversations WHERE mailbox_id = $1 AND status = 'open'`,
		mailboxID).Scan(&count)
	return count, err
}

func (s *ConversationStore) CreateMessage(ctx context.Context, conversationID int64, direction, senderAddress, senderName, body string) (*models.ConversationMessage, error) {
	m := &models.ConversationMessage{
		PublicID:       uuid.New(),
		ConversationID: conversationID,
		Direction:      models.MessageDirection(direction),
		SenderAddress:  senderAddress,
		SenderName:     senderName,
		Body:           body,
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO conversation_messages (public_id, conversation_id, direction, sender_address, sender_name, body)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		m.PublicID, m.ConversationID, string(m.Direction), m.SenderAddress, m.SenderName, m.Body,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Touch the conversation's updated_at
	_, _ = s.db.ExecContext(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`, conversationID)

	return m, nil
}

func (s *ConversationStore) GetMessagesByConversationID(ctx context.Context, conversationID int64) ([]models.ConversationMessage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, public_id, conversation_id, direction, sender_address, sender_name, body, created_at
		 FROM conversation_messages WHERE conversation_id = $1
		 ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.ConversationMessage
	for rows.Next() {
		var m models.ConversationMessage
		if err := rows.Scan(&m.ID, &m.PublicID, &m.ConversationID, &m.Direction, &m.SenderAddress, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
```

**Step 4: Verify compilation**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/store/postgres/`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/store/postgres/mailboxes.go internal/store/postgres/streams.go internal/store/postgres/conversations.go
git commit -m "feat: implement Postgres stores for mailboxes, streams, conversations"
```

---

### Task 5: Implement Mailbox Service with Tests

**Files:**
- Create: `internal/mailbox/service.go`
- Create: `internal/mailbox/service_test.go`

**Step 1: Write the failing tests**

Create `internal/mailbox/service_test.go`:

```go
package mailbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
)

// --- Mock stores ---

type mockMailboxStore struct {
	mailboxes  map[int64]*models.Mailbox
	byPublicID map[uuid.UUID]*models.Mailbox
	byUserID   map[int64][]models.Mailbox
	nextID     int64
}

func newMockMailboxStore() *mockMailboxStore {
	return &mockMailboxStore{
		mailboxes:  make(map[int64]*models.Mailbox),
		byPublicID: make(map[uuid.UUID]*models.Mailbox),
		byUserID:   make(map[int64][]models.Mailbox),
		nextID:     1,
	}
}

func (m *mockMailboxStore) CreateMailbox(_ context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error) {
	mb := &models.Mailbox{
		ID:          m.nextID,
		PublicID:    uuid.New(),
		UserID:      userID,
		DomainID:    domainID,
		Name:        name,
		FromAddress: fromAddress,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.nextID++
	m.mailboxes[mb.ID] = mb
	m.byPublicID[mb.PublicID] = mb
	m.byUserID[userID] = append(m.byUserID[userID], *mb)
	return mb, nil
}

func (m *mockMailboxStore) GetMailboxesByUserID(_ context.Context, userID int64) ([]models.Mailbox, error) {
	return m.byUserID[userID], nil
}

func (m *mockMailboxStore) GetMailboxByID(_ context.Context, id int64) (*models.Mailbox, error) {
	mb, ok := m.mailboxes[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStore) GetMailboxByPublicID(_ context.Context, publicID uuid.UUID) (*models.Mailbox, error) {
	mb, ok := m.byPublicID[publicID]
	if !ok {
		return nil, errors.New("not found")
	}
	return mb, nil
}

func (m *mockMailboxStore) DeleteMailbox(_ context.Context, id int64) error {
	delete(m.mailboxes, id)
	return nil
}

type mockDomainStoreForMailbox struct {
	domains map[int64]*models.Domain
}

func newMockDomainStoreForMailbox() *mockDomainStoreForMailbox {
	return &mockDomainStoreForMailbox{
		domains: make(map[int64]*models.Domain),
	}
}

func (m *mockDomainStoreForMailbox) addDomain(d *models.Domain) {
	m.domains[d.ID] = d
}

func (m *mockDomainStoreForMailbox) GetDomainByID(_ context.Context, id int64) (*models.Domain, error) {
	d, ok := m.domains[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	mb, err := svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mb.Name != "Support" {
		t.Errorf("expected name Support, got %s", mb.Name)
	}
	if mb.FromAddress != "support@example.com" {
		t.Errorf("expected from_address support@example.com, got %s", mb.FromAddress)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "", "support@example.com")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreate_DomainNotVerified(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: false, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	if err == nil {
		t.Fatal("expected error for unverified domain")
	}
}

func TestCreate_FromAddressDomainMismatch(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, err := svc.Create(context.Background(), 1, 1, "Support", "support@other.com")
	if err == nil {
		t.Fatal("expected error for from_address domain mismatch")
	}
}

func TestList_ReturnsMailboxes(t *testing.T) {
	ms := newMockMailboxStore()
	ds := newMockDomainStoreForMailbox()
	ds.addDomain(&models.Domain{ID: 1, Verified: true, Name: "example.com"})
	svc := NewService(ms, ds)

	_, _ = svc.Create(context.Background(), 1, 1, "Support", "support@example.com")
	_, _ = svc.Create(context.Background(), 1, 1, "Sales", "sales@example.com")

	mailboxes, err := svc.List(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(mailboxes) != 2 {
		t.Errorf("expected 2 mailboxes, got %d", len(mailboxes))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/mailbox/ -v`
Expected: Compilation error (package doesn't exist yet)

**Step 3: Implement the service**

Create `internal/mailbox/service.go`:

```go
package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

// DomainLookup is the subset of DomainStore needed by the mailbox service.
type DomainLookup interface {
	GetDomainByID(ctx context.Context, id int64) (*models.Domain, error)
}

type Service struct {
	mailboxes store.MailboxStore
	domains   DomainLookup
}

func NewService(mailboxes store.MailboxStore, domains DomainLookup) *Service {
	return &Service{
		mailboxes: mailboxes,
		domains:   domains,
	}
}

func (s *Service) Create(ctx context.Context, userID, domainID int64, name, fromAddress string) (*models.Mailbox, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("mailbox name must not be empty")
	}

	fromAddress = strings.TrimSpace(fromAddress)
	if fromAddress == "" {
		return nil, errors.New("from address must not be empty")
	}

	domain, err := s.domains.GetDomainByID(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}

	if !domain.Verified {
		return nil, errors.New("domain must be verified before creating a mailbox")
	}

	// Validate that from_address belongs to the domain
	parts := strings.SplitN(fromAddress, "@", 2)
	if len(parts) != 2 || parts[1] != domain.Name {
		return nil, fmt.Errorf("from address must be on domain %s", domain.Name)
	}

	mb, err := s.mailboxes.CreateMailbox(ctx, userID, domainID, name, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("create mailbox: %w", err)
	}
	return mb, nil
}

func (s *Service) List(ctx context.Context, userID int64) ([]models.Mailbox, error) {
	return s.mailboxes.GetMailboxesByUserID(ctx, userID)
}

func (s *Service) GetByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Mailbox, error) {
	return s.mailboxes.GetMailboxByPublicID(ctx, publicID)
}

func (s *Service) Delete(ctx context.Context, mailboxID int64) error {
	return s.mailboxes.DeleteMailbox(ctx, mailboxID)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/mailbox/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/mailbox/service.go internal/mailbox/service_test.go
git commit -m "feat: add mailbox service with validation and tests"
```

---

### Task 6: Implement Conversation Service with Tests

**Files:**
- Create: `internal/conversation/service.go`
- Create: `internal/conversation/service_test.go`

**Step 1: Write the failing tests**

Create `internal/conversation/service_test.go` with mock stores and tests for:
- `TestStartConversation_Success` — creates conversation + first inbound message
- `TestStartConversation_StreamDisabled` — rejects if stream is not enabled
- `TestReply_Success` — adds outbound message to existing conversation
- `TestReply_ConversationClosed` — rejects reply to closed conversation
- `TestClose_Success` — marks conversation as closed
- `TestListConversations` — returns conversations for a mailbox

(Test file follows the same mock pattern as `internal/message/service_test.go` — in-memory maps implementing the store interfaces.)

**Step 2: Run tests to verify they fail**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/conversation/ -v`
Expected: Compilation error

**Step 3: Implement the service**

Create `internal/conversation/service.go`:

```go
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
	ErrStreamDisabled      = errors.New("stream is disabled")
	ErrConversationClosed  = errors.New("conversation is closed")
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

type Service struct {
	conversations store.ConversationStore
	mailboxes     store.MailboxStore
	streams       store.StreamStore
	notifier      Notifier
	sender        Sender
}

func NewService(
	conversations store.ConversationStore,
	mailboxes store.MailboxStore,
	streams store.StreamStore,
	notifier Notifier,
	sender Sender,
) *Service {
	return &Service{
		conversations: conversations,
		mailboxes:     mailboxes,
		streams:       streams,
		notifier:      notifier,
		sender:        sender,
	}
}

// StartConversation creates a new conversation from an inbound message.
func (s *Service) StartConversation(ctx context.Context, streamID int64, subject, senderAddress, senderName, body string) (*models.Conversation, error) {
	stream, err := s.getEnabledStream(ctx, streamID)
	if err != nil {
		return nil, err
	}

	conv, err := s.conversations.CreateConversation(ctx, stream.MailboxID, streamID, subject)
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

func (s *Service) getEnabledStream(ctx context.Context, streamID int64) (*models.Stream, error) {
	// StreamStore doesn't have GetByID, so we need to add it or work around.
	// For now, we'll store the stream in the service call context.
	// Actually, we should add GetStreamByID to the store interface.
	// This will be addressed in a follow-up to Task 3 — add GetStreamByID.
	return nil, errors.New("not implemented — see Task 6 note")
}
```

**IMPORTANT NOTE:** We need to add `GetStreamByID` to the `StreamStore` interface and its Postgres implementation. Update Task 3's `StreamStore` interface to include:

```go
GetStreamByID(ctx context.Context, id int64) (*models.Stream, error)
```

And the `getEnabledStream` helper becomes:

```go
func (s *Service) getEnabledStream(ctx context.Context, streamID int64) (*models.Stream, error) {
	// We need a StreamByID lookup. For the service, we'll accept the stream directly
	// or look it up. Since StartConversation is called by the inbound handler which
	// already has the stream, we'll refactor to accept *models.Stream directly.
	return nil, nil
}
```

**Revised approach:** Change `StartConversation` to accept `*models.Stream` directly (the caller already looked it up). Remove `getEnabledStream`:

```go
func (s *Service) StartConversation(ctx context.Context, stream *models.Stream, subject, senderAddress, senderName, body string) (*models.Conversation, error) {
	if !stream.Enabled {
		return nil, ErrStreamDisabled
	}
	// ... rest of implementation
}
```

**Step 4: Run tests**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/conversation/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/conversation/service.go internal/conversation/service_test.go
git commit -m "feat: add conversation service with threading, replies, and tests"
```

---

### Task 7: Add Inbound SMTP Config

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add inbound SMTP fields to Config**

Add to the `Config` struct:

```go
InboundSMTPAddr   string
InboundSMTPDomain string
InboundSMTPEnabled bool
```

Add to `Load()`:

```go
inboundAddr := getEnv("INBOUND_SMTP_ADDR", "")
inboundDomain := getEnv("INBOUND_SMTP_DOMAIN", "localhost")
```

And in the return:

```go
InboundSMTPAddr:    inboundAddr,
InboundSMTPDomain:  inboundDomain,
InboundSMTPEnabled: inboundAddr != "",
```

**Step 2: Verify compilation**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/config/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add inbound SMTP configuration"
```

---

### Task 8: Implement Inbound SMTP Server

**Files:**
- Create: `internal/inbound/server.go`
- Create: `internal/inbound/server_test.go`

This task adds the Go SMTP server using `github.com/emersion/go-smtp`.

**Step 1: Add dependency**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go get github.com/emersion/go-smtp github.com/emersion/go-message`

**Step 2: Write the failing test**

Create `internal/inbound/server_test.go` that tests:
- `TestInboundEmail_CreatesConversation` — send a test email, verify conversation created
- `TestInboundEmail_UnknownRecipient` — email to unknown address is rejected

Tests use mock stores and a test SMTP client to connect to the server.

**Step 3: Implement the inbound SMTP server**

Create `internal/inbound/server.go`:

```go
package inbound

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

type Server struct {
	smtpServer    *smtp.Server
	streams       store.StreamStore
	conversations *conversation.Service
}

func NewServer(addr, domain string, streams store.StreamStore, conversations *conversation.Service) *Server {
	s := &Server{
		streams:       streams,
		conversations: conversations,
	}

	smtpSrv := smtp.NewServer(s)
	smtpSrv.Addr = addr
	smtpSrv.Domain = domain
	smtpSrv.ReadTimeout = 30 * time.Second
	smtpSrv.WriteTimeout = 30 * time.Second
	smtpSrv.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
	smtpSrv.MaxRecipients = 1
	smtpSrv.AllowInsecureAuth = true

	s.smtpServer = smtpSrv
	return s
}

func (s *Server) Start() error {
	slog.Info("inbound SMTP server starting", "addr", s.smtpServer.Addr)
	return s.smtpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.smtpServer.Close()
}

// NewSession implements smtp.Backend
func (s *Server) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &session{server: s}, nil
}

type session struct {
	server *Server
	from   string
	to     string
	stream *models.Stream
}

func (s *session) AuthPlain(username, password string) error {
	return nil // No auth required for inbound
}

func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// Look up the stream by recipient address
	addr := strings.ToLower(strings.TrimSpace(to))
	stream, err := s.server.streams.GetStreamByAddress(context.Background(), addr)
	if err != nil {
		slog.Warn("inbound email to unknown address", "to", addr)
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "no such recipient",
		}
	}

	if !stream.Enabled {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "recipient disabled",
		}
	}

	s.to = addr
	s.stream = stream
	return nil
}

func (s *session) Data(r io.Reader) error {
	if s.stream == nil {
		return errors.New("no valid recipient")
	}

	body, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return err
	}

	// Parse the email to extract subject and plain text body.
	subject, plainBody := parseEmail(body)

	_, err = s.server.conversations.StartConversation(
		context.Background(),
		s.stream,
		subject,
		s.from,
		"", // sender name extracted from email headers if available
		plainBody,
	)
	if err != nil {
		slog.Error("failed to create conversation from inbound email",
			"from", s.from, "to", s.to, "error", err)
		return err
	}

	slog.Info("inbound email processed", "from", s.from, "to", s.to, "subject", subject)
	return nil
}

func (s *session) Reset() {
	s.from = ""
	s.to = ""
	s.stream = nil
}

func (s *session) Logout() error {
	return nil
}

// parseEmail extracts subject and plain text body from raw email bytes.
// Uses go-message for MIME parsing.
func parseEmail(raw []byte) (subject, body string) {
	// Minimal parsing: look for Subject header and body
	// Full MIME parsing with go-message can be added later
	lines := strings.SplitN(string(raw), "\r\n\r\n", 2)
	if len(lines) == 2 {
		body = lines[1]
	}
	for _, line := range strings.Split(lines[0], "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			subject = strings.TrimSpace(line[8:])
		}
	}
	return subject, body
}
```

**Step 4: Run tests**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/inbound/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/inbound/
git commit -m "feat: add inbound SMTP server with stream routing"
```

---

### Task 9: Implement Outbound Reply Sender

**Files:**
- Modify: `internal/mail/smtp.go` — add `SendFrom` method that allows specifying a custom from address
- Modify: `internal/mail/service.go` — implement `conversation.Sender` interface

**Step 1: Add SendFrom to SMTPClient**

In `internal/mail/smtp.go`, add:

```go
// SendFrom delivers an email using a custom from address.
// Used for mailbox replies where the from address is the user's domain.
func (c *SMTPClient) SendFrom(from, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth := smtp.PlainAuth("", c.user, c.pass, c.host)

	headers := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n",
		from, to, subject,
	)

	msg := []byte(headers + body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
```

**Step 2: Add SendReply to mail.Service**

In `internal/mail/service.go`, add:

```go
// SendReply sends a reply email from a mailbox. Implements conversation.Sender.
func (s *Service) SendReply(ctx context.Context, to, fromAddress, fromName, subject, body string) error {
	from := fromAddress
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromAddress)
	}
	return s.client.SendFrom(from, to, subject, body)
}
```

**Step 3: Verify compilation**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/mail/`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/mail/smtp.go internal/mail/service.go
git commit -m "feat: add SendFrom and SendReply for mailbox outbound replies"
```

---

### Task 10: Implement Web Handlers for Mailboxes

**Files:**
- Create: `internal/web/handlers/mailboxes.go`

**Step 1: Implement MailboxHandler**

```go
package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/domain"
	"github.com/znz-systems/deaddrop/internal/mailbox"
	"github.com/znz-systems/deaddrop/internal/store"
	"github.com/znz-systems/deaddrop/internal/web/middleware"
	"github.com/znz-systems/deaddrop/internal/web/render"
)

type MailboxHandler struct {
	mailboxes     *mailbox.Service
	conversations *conversation.Service
	domains       *domain.Service
	streams       store.StreamStore
	convStore     store.ConversationStore
	render        *render.Renderer
	secureCookies bool
}

func NewMailboxHandler(
	mailboxes *mailbox.Service,
	conversations *conversation.Service,
	domains *domain.Service,
	streams store.StreamStore,
	convStore store.ConversationStore,
	r *render.Renderer,
	secureCookies bool,
) *MailboxHandler {
	return &MailboxHandler{
		mailboxes:     mailboxes,
		conversations: conversations,
		domains:       domains,
		streams:       streams,
		convStore:     convStore,
		render:        r,
		secureCookies: secureCookies,
	}
}

func (h *MailboxHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mailboxes, err := h.mailboxes.List(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to list mailboxes", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	type mailboxWithCount struct {
		Mailbox   interface{}
		OpenCount int
	}

	items := make([]mailboxWithCount, 0, len(mailboxes))
	for _, mb := range mailboxes {
		count, err := h.conversations.CountOpen(r.Context(), mb.ID)
		if err != nil {
			count = 0
		}
		items = append(items, mailboxWithCount{Mailbox: mb, OpenCount: count})
	}

	h.render.Render(w, r, "mailbox_dashboard.html", map[string]interface{}{
		"User":      user,
		"Mailboxes": items,
	})
}

func (h *MailboxHandler) ShowNewMailbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	domains, _ := h.domains.List(r.Context(), user.ID)
	h.render.Render(w, r, "mailbox_new.html", map[string]interface{}{
		"User":    user,
		"Domains": domains,
	})
}

func (h *MailboxHandler) HandleCreateMailbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	fromAddress := r.FormValue("from_address")
	domainIDStr := r.FormValue("domain_id")

	domainID, err := strconv.ParseInt(domainIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.Create(r.Context(), user.ID, domainID, name, fromAddress)
	if err != nil {
		slog.Error("failed to create mailbox", "error", err)
		domains, _ := h.domains.List(r.Context(), user.ID)
		h.render.Render(w, r, "mailbox_new.html", map[string]interface{}{
			"User":    user,
			"Domains": domains,
			"Error":   err.Error(),
			"Name":    name,
		})
		return
	}

	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) ShowMailboxDetail(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid mailbox id", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convos, _ := h.conversations.List(r.Context(), mb.ID, 50, 0)
	streams, _ := h.streams.GetStreamsByMailboxID(r.Context(), mb.ID)

	h.render.Render(w, r, "mailbox_detail.html", map[string]interface{}{
		"User":          user,
		"Mailbox":       mb,
		"Conversations": convos,
		"Streams":       streams,
	})
}

func (h *MailboxHandler) ShowConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid mailbox id", http.StatusBadRequest)
		return
	}

	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, err := uuid.Parse(chi.URLParam(r, "cid"))
	if err != nil {
		http.Error(w, "invalid conversation id", http.StatusBadRequest)
		return
	}

	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	messages, _ := h.conversations.GetMessages(r.Context(), conv.ID)

	h.render.Render(w, r, "conversation_detail.html", map[string]interface{}{
		"User":         user,
		"Mailbox":      mb,
		"Conversation": conv,
		"Messages":     messages,
	})
}

func (h *MailboxHandler) HandleReply(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, _ := uuid.Parse(chi.URLParam(r, "cid"))
	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	body := r.FormValue("body")
	if body == "" {
		setFlash(w, "Reply body cannot be empty", h.secureCookies)
		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	if _, err := h.conversations.Reply(r.Context(), conv.ID, body); err != nil {
		slog.Error("failed to send reply", "error", err)
		setFlash(w, "Failed to send reply: "+err.Error(), h.secureCookies)
	} else {
		setFlash(w, "Reply sent!", h.secureCookies)
	}

	http.Redirect(w, r, fmt.Sprintf("/mailboxes/%s/conversations/%s", mb.PublicID, conv.PublicID), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleCloseConversation(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mbPublicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), mbPublicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	convPublicID, _ := uuid.Parse(chi.URLParam(r, "cid"))
	conv, err := h.conversations.GetByPublicID(r.Context(), convPublicID)
	if err != nil || conv.MailboxID != mb.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	_ = h.conversations.Close(r.Context(), conv.ID)
	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleDeleteMailbox(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	_ = h.mailboxes.Delete(r.Context(), mb.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *MailboxHandler) HandleAddStream(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	streamType := r.FormValue("type")
	address := r.FormValue("address")

	var widgetID uuid.UUID
	if streamType == "form" {
		widgetID = uuid.New()
	}

	if _, err := h.streams.CreateStream(r.Context(), mb.ID, streamType, address, widgetID); err != nil {
		slog.Error("failed to create stream", "error", err)
		setFlash(w, "Failed to create stream: "+err.Error(), h.secureCookies)
	}

	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}

func (h *MailboxHandler) HandleDeleteStream(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	publicID, _ := uuid.Parse(chi.URLParam(r, "id"))
	mb, err := h.mailboxes.GetByPublicID(r.Context(), publicID)
	if err != nil || mb.UserID != user.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	sidStr := chi.URLParam(r, "sid")
	sid, err := strconv.ParseInt(sidStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	_ = h.streams.DeleteStream(r.Context(), sid)
	http.Redirect(w, r, "/mailboxes/"+mb.PublicID.String(), http.StatusSeeOther)
}
```

**Step 2: Verify compilation**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./internal/web/handlers/`
Expected: No errors (may need `fmt` import added)

**Step 3: Commit**

```bash
git add internal/web/handlers/mailboxes.go
git commit -m "feat: add mailbox web handlers for CRUD, conversations, streams"
```

---

### Task 11: Add Templates

**Files:**
- Create: `templates/mailbox_dashboard.html`
- Create: `templates/mailbox_new.html`
- Create: `templates/mailbox_detail.html`
- Create: `templates/conversation_detail.html`
- Create: `templates/partials/mailbox_row.html`
- Create: `templates/partials/conversation_row.html`

Templates follow the existing design system (Space Grotesk/Mono, brutalist style with --red, --black, --bg variables). Use the same CSS classes from `base.html`.

Each template defines `title` and `content` blocks matching the existing `base.html` structure. The `Renderer` automatically picks up new templates from the `templates/` directory.

**USER CONTRIBUTION OPPORTUNITY:** The template HTML/CSS is where your design preferences matter most. The structure and data bindings are determined by the handlers, but the visual layout is up to you. I'll create the templates with functional markup using the existing design system.

**Step 1: Create all template files** (functional, matching existing design patterns)

**Step 2: Verify the Go embed picks them up**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add templates/
git commit -m "feat: add mailbox, conversation, and stream templates"
```

---

### Task 12: Update Router and Main

**Files:**
- Modify: `internal/web/router.go` — add MailboxHandler routes, update RouterDeps
- Modify: `cmd/deaddrop/main.go` — wire up new stores, services, handlers, SMTP server

**Step 1: Update RouterDeps and routes**

Add `MailboxHandler *handlers.MailboxHandler` to `RouterDeps`.

Add mailbox routes in the authenticated group:

```go
// Mailbox routes
r.Get("/", deps.MailboxHandler.ShowDashboard) // Replace domain dashboard
r.Get("/mailboxes/new", deps.MailboxHandler.ShowNewMailbox)
r.Post("/mailboxes", deps.MailboxHandler.HandleCreateMailbox)
r.Get("/mailboxes/{id}", deps.MailboxHandler.ShowMailboxDetail)
r.Post("/mailboxes/{id}/delete", deps.MailboxHandler.HandleDeleteMailbox)
r.Post("/mailboxes/{id}/streams", deps.MailboxHandler.HandleAddStream)
r.Post("/mailboxes/{id}/streams/{sid}/delete", deps.MailboxHandler.HandleDeleteStream)
r.Get("/mailboxes/{id}/conversations/{cid}", deps.MailboxHandler.ShowConversation)
r.Post("/mailboxes/{id}/conversations/{cid}/reply", deps.MailboxHandler.HandleReply)
r.Post("/mailboxes/{id}/conversations/{cid}/close", deps.MailboxHandler.HandleCloseConversation)
```

Keep the existing domain routes for domain management (verification, etc.).

**Step 2: Update main.go**

Add new store initialization, service wiring, and SMTP server startup:

```go
// New stores
mailboxStore := postgres.NewMailboxStore(db)
streamStore := postgres.NewStreamStore(db)
conversationStore := postgres.NewConversationStore(db)

// New services
mailboxService := mailbox.NewService(mailboxStore, domainStore)

var sender conversation.Sender
if cfg.SMTPEnabled {
	sender = mail.NewService(smtpClient, userStore)
} else {
	sender = &conversation.NoopSender{} // need to add this
}

conversationService := conversation.NewService(conversationStore, mailboxStore, streamStore, convNotifier, sender)

// Inbound SMTP server
if cfg.InboundSMTPEnabled {
	smtpSrv := inbound.NewServer(cfg.InboundSMTPAddr, cfg.InboundSMTPDomain, streamStore, conversationService)
	go func() {
		if err := smtpSrv.Start(); err != nil {
			slog.Error("inbound SMTP server error", "error", err)
		}
	}()
}
```

**Step 3: Verify full build**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/web/router.go cmd/deaddrop/main.go
git commit -m "feat: wire mailbox system into router and main"
```

---

### Task 13: Update Docker Compose

**Files:**
- Modify: `docker/docker-compose.yml`

**Step 1: Add SMTP port and env vars**

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: docker/Dockerfile
    ports:
      - "8080:8080"
      - "25:2525"
    depends_on:
      - db
    env_file:
      - .env
    environment:
      - INBOUND_SMTP_ADDR=:2525
      - INBOUND_SMTP_DOMAIN=${SMTP_DOMAIN:-localhost}
    restart: unless-stopped
```

**Step 2: Commit**

```bash
git add docker/docker-compose.yml
git commit -m "feat: expose inbound SMTP port in Docker Compose"
```

---

### Task 14: Update Public API for Form Submissions

**Files:**
- Modify: `internal/web/handlers/api.go`

**Step 1: Update HandleSubmitMessage to create conversations**

The existing widget API (`POST /api/v1/messages`) needs to route through the new stream/conversation system. The `domain_id` form field now maps to a stream's `widget_id`.

Update `APIHandler` to accept `StreamStore` and `*conversation.Service` instead of `*message.Service`. Look up the stream by widget ID, then call `conversations.StartConversation`.

**Step 2: Run existing API tests**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./internal/web/handlers/ -v`
Expected: Tests may need updating for new dependencies

**Step 3: Update tests**

Update `internal/web/handlers/api_test.go` to use the new conversation-based flow.

**Step 4: Commit**

```bash
git add internal/web/handlers/api.go internal/web/handlers/api_test.go
git commit -m "feat: update public API to create conversations via streams"
```

---

### Task 15: Run Full Test Suite and Fix Issues

**Step 1: Run all tests**

Run: `cd /Volumes/BigStorage/CodeWorld/DeadDrop && go test ./... -v`

**Step 2: Fix any compilation or test failures**

Address issues discovered during the full build.

**Step 3: Commit fixes**

```bash
git add -A
git commit -m "fix: resolve test failures from mailbox integration"
```

---

### Task 16: Notification Service for Conversations

**Files:**
- Modify: `internal/mail/service.go` — add `NotifyNewConversation` method
- Modify: `internal/mail/templates.go` — add conversation notification template

**Step 1: Add notification template**

Add `NewConversationNotificationBody` to `internal/mail/templates.go` that formats a notification email with the conversation subject, sender, and a link to view it in the dashboard.

**Step 2: Implement NotifyNewConversation**

Add to `internal/mail/service.go`:

```go
func (s *Service) NotifyNewConversation(ctx context.Context, mailbox *models.Mailbox, conv *models.Conversation, msg *models.ConversationMessage) error {
	user, err := s.users.GetUserByID(ctx, mailbox.UserID)
	if err != nil {
		return fmt.Errorf("mail: failed to look up mailbox owner: %w", err)
	}

	subject := fmt.Sprintf("New conversation in %s", mailbox.Name)
	body := NewConversationNotificationBody(mailbox.Name, msg.SenderName, msg.SenderAddress, msg.Body, conv.Subject)

	return s.client.Send(user.Email, subject, body)
}
```

**Step 3: Commit**

```bash
git add internal/mail/service.go internal/mail/templates.go
git commit -m "feat: add conversation notification emails"
```

---

### Summary of Tasks

| # | Task | Key Files |
|---|------|-----------|
| 1 | Add new models | `internal/models/models.go` |
| 2 | Create migrations | `migrations/005-007` |
| 3 | Add store interfaces | `internal/store/store.go` |
| 4 | Implement Postgres stores | `internal/store/postgres/{mailboxes,streams,conversations}.go` |
| 5 | Mailbox service + tests | `internal/mailbox/` |
| 6 | Conversation service + tests | `internal/conversation/` |
| 7 | Inbound SMTP config | `internal/config/config.go` |
| 8 | Inbound SMTP server | `internal/inbound/` |
| 9 | Outbound reply sender | `internal/mail/{smtp,service}.go` |
| 10 | Web handlers | `internal/web/handlers/mailboxes.go` |
| 11 | Templates | `templates/mailbox_*.html`, `templates/conversation_*.html` |
| 12 | Router + main wiring | `internal/web/router.go`, `cmd/deaddrop/main.go` |
| 13 | Docker Compose | `docker/docker-compose.yml` |
| 14 | Update public API | `internal/web/handlers/api.go` |
| 15 | Full test suite pass | All files |
| 16 | Notification service | `internal/mail/{service,templates}.go` |
