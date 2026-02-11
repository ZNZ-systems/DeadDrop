package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

// Service contains the business logic for domain management.
type Service struct {
	domains  store.DomainStore
	resolver DNSResolver
}

// NewService creates a new domain Service.
func NewService(domains store.DomainStore, resolver DNSResolver) *Service {
	return &Service{
		domains:  domains,
		resolver: resolver,
	}
}

// Create validates the domain name, generates a verification token, and
// persists the new domain.
func (s *Service) Create(ctx context.Context, userID int64, name string) (*models.Domain, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("domain name must not be empty")
	}

	token := uuid.New().String()

	d, err := s.domains.CreateDomain(ctx, userID, name, token)
	if err != nil {
		return nil, fmt.Errorf("create domain: %w", err)
	}

	return d, nil
}

// List returns all domains belonging to the given user.
func (s *Service) List(ctx context.Context, userID int64) ([]models.Domain, error) {
	domains, err := s.domains.GetDomainsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	return domains, nil
}

// GetByPublicID retrieves a single domain by its public UUID.
func (s *Service) GetByPublicID(ctx context.Context, publicID uuid.UUID) (*models.Domain, error) {
	d, err := s.domains.GetDomainByPublicID(ctx, publicID)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}
	return d, nil
}

// Verify performs a DNS TXT lookup on the domain name and checks for a record
// matching "deaddrop-verify=<token>". If found the domain is marked as verified
// in the store.
func (s *Service) Verify(ctx context.Context, d *models.Domain) error {
	records, err := s.resolver.LookupTXT(d.Name)
	if err != nil {
		return fmt.Errorf("dns lookup failed for %s: %w", d.Name, err)
	}

	expected := "deaddrop-verify=" + d.VerificationToken

	for _, record := range records {
		if strings.TrimSpace(record) == expected {
			if err := s.domains.MarkDomainVerified(ctx, d.ID); err != nil {
				return fmt.Errorf("mark domain verified: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("verification TXT record not found for %s", d.Name)
}

// Delete removes a domain by its internal ID.
func (s *Service) Delete(ctx context.Context, domainID int64) error {
	if err := s.domains.DeleteDomain(ctx, domainID); err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	return nil
}
