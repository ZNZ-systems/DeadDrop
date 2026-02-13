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
