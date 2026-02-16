package inbound

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

type MXResolver interface {
	LookupMX(name string) ([]*net.MX, error)
}

type NetMXResolver struct{}

func (r *NetMXResolver) LookupMX(name string) ([]*net.MX, error) {
	return net.LookupMX(name)
}

type DomainService struct {
	configs  store.InboundDomainConfigStore
	resolver MXResolver
	mxTarget string
}

func NewDomainService(configs store.InboundDomainConfigStore, resolver MXResolver, mxTarget string) *DomainService {
	return &DomainService{
		configs:  configs,
		resolver: resolver,
		mxTarget: normalizeMXHost(mxTarget),
	}
}

func (s *DomainService) EnsureConfig(ctx context.Context, domainID int64) (*models.InboundDomainConfig, error) {
	return s.configs.UpsertInboundDomainConfig(ctx, domainID, s.mxTarget)
}

func (s *DomainService) VerifyMX(ctx context.Context, d *models.Domain) (bool, error) {
	if d == nil {
		return false, fmt.Errorf("domain is required")
	}

	cfg, err := s.configs.UpsertInboundDomainConfig(ctx, d.ID, s.mxTarget)
	if err != nil {
		return false, fmt.Errorf("upsert inbound config: %w", err)
	}

	records, err := s.resolver.LookupMX(d.Name)
	if err != nil {
		msg := fmt.Sprintf("mx lookup failed: %v", err)
		_ = s.configs.UpdateInboundDomainVerification(ctx, d.ID, false, msg)
		return false, fmt.Errorf("%s", msg)
	}

	target := normalizeMXHost(cfg.MXTarget)
	for _, mx := range records {
		if normalizeMXHost(mx.Host) == target {
			if err := s.configs.UpdateInboundDomainVerification(ctx, d.ID, true, ""); err != nil {
				return false, fmt.Errorf("update inbound verification: %w", err)
			}
			return true, nil
		}
	}

	msg := fmt.Sprintf("mx record not found; expected %s", target)
	if err := s.configs.UpdateInboundDomainVerification(ctx, d.ID, false, msg); err != nil {
		return false, fmt.Errorf("update inbound verification: %w", err)
	}
	return false, nil
}

func normalizeMXHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimSuffix(host, ".")
	return host
}
