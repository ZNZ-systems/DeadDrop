package mail

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

var smtpSendMail = smtp.SendMail

// SMTPClient wraps net/smtp to provide a simple interface for sending emails.
type SMTPClient struct {
	host string
	port int
	user string
	pass string
	from string
}

// NewSMTPClient creates a new SMTPClient with the given SMTP server configuration.
func NewSMTPClient(host string, port int, user, pass, from string) *SMTPClient {
	return &SMTPClient{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

func (c *SMTPClient) auth() (smtp.Auth, error) {
	if c.user == "" && c.pass == "" {
		return nil, nil
	}
	if c.user == "" || c.pass == "" {
		return nil, errors.New("smtp credentials are incomplete")
	}
	return smtp.PlainAuth("", c.user, c.pass, c.host), nil
}

func (c *SMTPClient) send(envelopeFrom, headerFrom, to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth, err := c.auth()
	if err != nil {
		return err
	}

	if envelopeFrom == "" {
		envelopeFrom = c.from
	}
	if headerFrom == "" {
		headerFrom = envelopeFrom
	}

	if envelopeFrom == "" {
		return errors.New("from address is required")
	}

	messageID := buildMessageID(envelopeFrom, headerFrom, c.from)
	dateHeader := time.Now().UTC().Format(time.RFC1123Z)

	headers := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Date: %s\r\n"+
			"Message-ID: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n",
		headerFrom, to, subject, dateHeader, messageID,
	)

	msg := []byte(headers + body)
	return smtpSendMail(addr, auth, envelopeFrom, []string{to}, msg)
}

func buildMessageID(addresses ...string) string {
	domain := "localhost"
	for _, raw := range addresses {
		d := extractDomain(raw)
		if d != "" {
			domain = d
			break
		}
	}

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Sprintf("<%d@%s>", time.Now().UTC().UnixNano(), domain)
	}
	return fmt.Sprintf("<%d.%x@%s>", time.Now().UTC().UnixNano(), suffix, domain)
}

func extractDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	addr := raw
	if parsed, err := mail.ParseAddress(raw); err == nil {
		addr = parsed.Address
	}

	at := strings.LastIndex(addr, "@")
	if at < 1 || at+1 >= len(addr) {
		return ""
	}

	domain := strings.Trim(strings.TrimSpace(addr[at+1:]), " >")
	if domain == "" {
		return ""
	}
	return strings.ToLower(domain)
}

// Send delivers an HTML email to the specified recipient using SMTP.
func (c *SMTPClient) Send(to, subject, body string) error {
	return c.send(c.from, c.from, to, subject, body)
}

// SendFrom delivers an email using a custom envelope sender and header sender.
// Used for mailbox replies where the From header should include the mailbox name.
func (c *SMTPClient) SendFrom(envelopeFrom, headerFrom, to, subject, body string) error {
	return c.send(envelopeFrom, headerFrom, to, subject, body)
}
