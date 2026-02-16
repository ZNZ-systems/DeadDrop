package inbound

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/znz-systems/deaddrop/internal/blob"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

var (
	ErrSenderRequired     = errors.New("sender is required")
	ErrRecipientsRequired = errors.New("at least one recipient is required")
)

type Message struct {
	Sender      string
	Recipients  []string
	Subject     string
	TextBody    string
	HTMLBody    string
	MessageID   string
	RawRFC822   string
	Attachments []Attachment
}

type Attachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

type IngestResult struct {
	Accepted int `json:"accepted"`
	Dropped  int `json:"dropped"`
}

type Service struct {
	domains store.DomainStore
	emails  store.InboundEmailStore
	configs store.InboundDomainConfigStore
	rules   store.InboundRecipientRuleStore
	blobs   blob.Store
}

func NewService(domains store.DomainStore, emails store.InboundEmailStore, configs store.InboundDomainConfigStore, rules store.InboundRecipientRuleStore, blobs blob.Store) *Service {
	return &Service{
		domains: domains,
		emails:  emails,
		configs: configs,
		rules:   rules,
		blobs:   blobs,
	}
}

func (s *Service) Ingest(ctx context.Context, msg Message) (IngestResult, error) {
	msg.Sender = strings.TrimSpace(msg.Sender)
	if msg.Sender == "" {
		return IngestResult{}, ErrSenderRequired
	}
	senderAddr, err := mail.ParseAddress(msg.Sender)
	if err != nil {
		return IngestResult{}, fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.Recipients) == 0 {
		return IngestResult{}, ErrRecipientsRequired
	}

	msg.Subject = strings.TrimSpace(msg.Subject)
	msg.TextBody = strings.TrimSpace(msg.TextBody)
	msg.HTMLBody = strings.TrimSpace(msg.HTMLBody)
	msg.MessageID = strings.TrimSpace(msg.MessageID)
	msg.RawRFC822 = strings.TrimSpace(msg.RawRFC822)

	res := IngestResult{}
	ruleCache := map[int64][]models.InboundRecipientRule{}
	for _, rawRecipient := range msg.Recipients {
		recipient := strings.TrimSpace(rawRecipient)
		if recipient == "" {
			res.Dropped++
			continue
		}

		addr, err := mail.ParseAddress(recipient)
		if err != nil {
			res.Dropped++
			continue
		}
		domainName, normalizedRecipient := normalizeRecipient(addr.Address)
		if domainName == "" {
			res.Dropped++
			continue
		}

		domain, err := s.domains.GetDomainByName(ctx, domainName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				res.Dropped++
				continue
			}
			return res, fmt.Errorf("get domain by name: %w", err)
		}
		if !domain.Verified {
			res.Dropped++
			continue
		}
		cfg, err := s.configs.GetInboundDomainConfigByDomainID(ctx, domain.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				res.Dropped++
				continue
			}
			return res, fmt.Errorf("get inbound domain config: %w", err)
		}
		if !cfg.MXVerified {
			res.Dropped++
			continue
		}

		localPart := strings.SplitN(normalizedRecipient, "@", 2)[0]
		rules, ok := ruleCache[domain.ID]
		if !ok {
			rules, err = s.rules.ListInboundRecipientRulesByDomainID(ctx, domain.ID)
			if err != nil {
				return res, fmt.Errorf("list inbound recipient rules: %w", err)
			}
			ruleCache[domain.ID] = rules
		}
		if !ruleAllowsRecipient(localPart, rules) {
			res.Dropped++
			continue
		}

		email, err := s.emails.CreateInboundEmail(ctx, models.InboundEmailCreateParams{
			UserID:    domain.UserID,
			DomainID:  domain.ID,
			Recipient: normalizedRecipient,
			Sender:    strings.ToLower(senderAddr.Address),
			Subject:   msg.Subject,
			TextBody:  msg.TextBody,
			HTMLBody:  msg.HTMLBody,
			MessageID: msg.MessageID,
		})
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				// Duplicate delivery, treat as dropped.
				res.Dropped++
				continue
			}
			return res, fmt.Errorf("create inbound email: %w", err)
		}

		if msg.RawRFC822 != "" {
			rawParams := models.InboundEmailRawCreateParams{
				InboundEmailID: email.ID,
				RawSource:      msg.RawRFC822,
			}
			if s.blobs != nil {
				rawKey := buildBlobKey("raw", domain.ID, email.ID, "eml")
				if err := s.blobs.Put(ctx, rawKey, "message/rfc822", []byte(msg.RawRFC822)); err != nil {
					return res, fmt.Errorf("store inbound raw blob: %w", err)
				}
				rawParams.RawSource = ""
				rawParams.BlobKey = rawKey
			}
			if err := s.emails.CreateInboundEmailRaw(ctx, rawParams); err != nil {
				return res, fmt.Errorf("store inbound raw email: %w", err)
			}
		}
		for _, attachment := range msg.Attachments {
			if len(attachment.Content) == 0 {
				continue
			}
			attachmentParams := models.InboundEmailAttachmentCreateParams{
				InboundEmailID: email.ID,
				FileName:       attachment.FileName,
				ContentType:    attachment.ContentType,
				SizeBytes:      int64(len(attachment.Content)),
				Content:        attachment.Content,
			}
			if s.blobs != nil {
				ext := blobExtFromName(attachment.FileName)
				blobKey := buildBlobKey("attachments", domain.ID, email.ID, ext)
				if err := s.blobs.Put(ctx, blobKey, attachment.ContentType, attachment.Content); err != nil {
					return res, fmt.Errorf("store inbound attachment blob: %w", err)
				}
				attachmentParams.BlobKey = blobKey
				attachmentParams.Content = nil
			}
			if _, err := s.emails.CreateInboundEmailAttachment(ctx, attachmentParams); err != nil {
				return res, fmt.Errorf("store inbound attachment: %w", err)
			}
		}
		res.Accepted++
	}

	return res, nil
}

func buildBlobKey(kind string, domainID, emailID int64, extension string) string {
	extension = strings.TrimSpace(strings.TrimPrefix(extension, "."))
	if extension == "" {
		extension = "bin"
	}
	return fmt.Sprintf("inbound/%s/%d/%d/%d-%s.%s",
		kind,
		domainID,
		emailID,
		time.Now().UTC().UnixNano(),
		uuid.NewString(),
		extension,
	)
}

func blobExtFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "bin"
	}
	lastDot := strings.LastIndex(name, ".")
	if lastDot == -1 || lastDot == len(name)-1 {
		return "bin"
	}
	ext := strings.TrimSpace(name[lastDot+1:])
	if ext == "" {
		return "bin"
	}
	return ext
}

func normalizeRecipient(email string) (domain string, normalized string) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "", ""
	}

	localPart := strings.TrimSpace(parts[0])
	domainPart := strings.TrimSpace(strings.ToLower(parts[1]))
	domainPart = strings.TrimSuffix(domainPart, ".")
	if localPart == "" || domainPart == "" {
		return "", ""
	}
	return domainPart, localPart + "@" + domainPart
}

func ruleAllowsRecipient(localPart string, rules []models.InboundRecipientRule) bool {
	localPart = strings.ToLower(strings.TrimSpace(localPart))
	if localPart == "" {
		return false
	}
	if len(rules) == 0 {
		return true
	}

	for _, r := range rules {
		if !r.IsActive {
			continue
		}
		if strings.ToLower(r.RuleType) == "exact" && strings.EqualFold(strings.TrimSpace(r.Pattern), localPart) {
			return strings.ToLower(r.Action) != "drop"
		}
	}
	for _, r := range rules {
		if !r.IsActive || strings.ToLower(r.RuleType) != "wildcard" {
			continue
		}
		pattern := strings.ToLower(strings.TrimSpace(r.Pattern))
		match, err := path.Match(pattern, localPart)
		if err != nil {
			continue
		}
		if match {
			return strings.ToLower(r.Action) != "drop"
		}
	}

	return false
}
