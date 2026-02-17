package mail

import (
	"errors"
	"fmt"
	"net/smtp"
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

	headers := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n",
		headerFrom, to, subject,
	)

	msg := []byte(headers + body)
	return smtpSendMail(addr, auth, envelopeFrom, []string{to}, msg)
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
