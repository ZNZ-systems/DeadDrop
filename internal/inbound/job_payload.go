package inbound

import "strings"

type IngestJobPayload struct {
	Sender     string   `json:"sender"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	TextBody   string   `json:"text_body"`
	HTMLBody   string   `json:"html_body"`
	MessageID  string   `json:"message_id"`
	RawRFC822  string   `json:"raw_rfc822"`
}

func (p *IngestJobPayload) Normalize() {
	p.Sender = strings.TrimSpace(p.Sender)
	p.Subject = strings.TrimSpace(p.Subject)
	p.TextBody = strings.TrimSpace(p.TextBody)
	p.HTMLBody = strings.TrimSpace(p.HTMLBody)
	p.MessageID = strings.TrimSpace(p.MessageID)
	p.RawRFC822 = strings.TrimSpace(p.RawRFC822)
	for i := range p.Recipients {
		p.Recipients[i] = strings.TrimSpace(p.Recipients[i])
	}
}

func (p IngestJobPayload) IsUsable() bool {
	if p.RawRFC822 != "" {
		return true
	}
	if p.Sender == "" {
		return false
	}
	for _, rcpt := range p.Recipients {
		if strings.TrimSpace(rcpt) != "" {
			return true
		}
	}
	return false
}

func (p IngestJobPayload) ToMessage() Message {
	recipients := make([]string, 0, len(p.Recipients))
	for _, rcpt := range p.Recipients {
		rcpt = strings.TrimSpace(rcpt)
		if rcpt == "" {
			continue
		}
		recipients = append(recipients, rcpt)
	}

	return Message{
		Sender:     p.Sender,
		Recipients: recipients,
		Subject:    p.Subject,
		TextBody:   p.TextBody,
		HTMLBody:   p.HTMLBody,
		MessageID:  p.MessageID,
		RawRFC822:  p.RawRFC822,
	}
}
