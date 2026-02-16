package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/znz-systems/deaddrop/internal/models"
)

type InboundRecipientRuleStore struct {
	db *sql.DB
}

func NewInboundRecipientRuleStore(db *sql.DB) *InboundRecipientRuleStore {
	return &InboundRecipientRuleStore{db: db}
}

func (s *InboundRecipientRuleStore) CreateInboundRecipientRule(ctx context.Context, domainID int64, ruleType, pattern, action string) (*models.InboundRecipientRule, error) {
	ruleType = strings.TrimSpace(strings.ToLower(ruleType))
	pattern = strings.TrimSpace(strings.ToLower(pattern))
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "" {
		action = "inbox"
	}

	var rule models.InboundRecipientRule
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO inbound_recipient_rules (domain_id, rule_type, pattern, action)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, domain_id, rule_type, pattern, action, is_active, created_at, updated_at`,
		domainID, ruleType, pattern, action,
	).Scan(&rule.ID, &rule.DomainID, &rule.RuleType, &rule.Pattern, &rule.Action, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *InboundRecipientRuleStore) ListInboundRecipientRulesByDomainID(ctx context.Context, domainID int64) ([]models.InboundRecipientRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, domain_id, rule_type, pattern, action, is_active, created_at, updated_at
		 FROM inbound_recipient_rules
		 WHERE domain_id = $1 AND is_active = TRUE
		 ORDER BY
		   CASE WHEN rule_type = 'exact' THEN 0 ELSE 1 END ASC,
		   created_at ASC`,
		domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]models.InboundRecipientRule, 0, 16)
	for rows.Next() {
		var r models.InboundRecipientRule
		if err := rows.Scan(&r.ID, &r.DomainID, &r.RuleType, &r.Pattern, &r.Action, &r.IsActive, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *InboundRecipientRuleStore) DeleteInboundRecipientRule(ctx context.Context, domainID, ruleID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM inbound_recipient_rules WHERE domain_id = $1 AND id = $2`, domainID, ruleID)
	return err
}
