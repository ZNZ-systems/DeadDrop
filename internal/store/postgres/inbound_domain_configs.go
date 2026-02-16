package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/znz-systems/deaddrop/internal/models"
)

type InboundDomainConfigStore struct {
	db *sql.DB
}

func NewInboundDomainConfigStore(db *sql.DB) *InboundDomainConfigStore {
	return &InboundDomainConfigStore{db: db}
}

func (s *InboundDomainConfigStore) UpsertInboundDomainConfig(ctx context.Context, domainID int64, mxTarget string) (*models.InboundDomainConfig, error) {
	mxTarget = strings.TrimSpace(mxTarget)
	var cfg models.InboundDomainConfig
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO inbound_domain_configs (domain_id, mx_target)
		 VALUES ($1, $2)
		 ON CONFLICT (domain_id) DO UPDATE
		 SET mx_target = EXCLUDED.mx_target, updated_at = NOW()
		 RETURNING domain_id, mx_target, mx_verified, last_error, checked_at, created_at, updated_at`,
		domainID, mxTarget,
	).Scan(
		&cfg.DomainID, &cfg.MXTarget, &cfg.MXVerified, &cfg.LastError, &cfg.CheckedAt, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *InboundDomainConfigStore) GetInboundDomainConfigByDomainID(ctx context.Context, domainID int64) (*models.InboundDomainConfig, error) {
	var cfg models.InboundDomainConfig
	err := s.db.QueryRowContext(ctx,
		`SELECT domain_id, mx_target, mx_verified, last_error, checked_at, created_at, updated_at
		 FROM inbound_domain_configs
		 WHERE domain_id = $1`,
		domainID,
	).Scan(&cfg.DomainID, &cfg.MXTarget, &cfg.MXVerified, &cfg.LastError, &cfg.CheckedAt, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *InboundDomainConfigStore) UpdateInboundDomainVerification(ctx context.Context, domainID int64, verified bool, lastError string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE inbound_domain_configs
		 SET mx_verified = $2, last_error = $3, checked_at = NOW(), updated_at = NOW()
		 WHERE domain_id = $1`,
		domainID, verified, strings.TrimSpace(lastError),
	)
	return err
}
