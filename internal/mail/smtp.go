package mail

import (
	"fmt"
	"net/smtp"
)

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

// Send delivers an HTML email to the specified recipient using SMTP with PlainAuth.
func (c *SMTPClient) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	auth := smtp.PlainAuth("", c.user, c.pass, c.host)

	headers := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n",
		c.from, to, subject,
	)

	msg := []byte(headers + body)

	return smtp.SendMail(addr, auth, c.from, []string{to}, msg)
}
