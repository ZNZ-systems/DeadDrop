package inbound

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/znz-systems/deaddrop/internal/conversation"
	"github.com/znz-systems/deaddrop/internal/models"
	"github.com/znz-systems/deaddrop/internal/store"
)

type Server struct {
	smtpServer    *smtp.Server
	streams       store.StreamStore
	conversations *conversation.Service
}

func NewServer(addr, domain string, streams store.StreamStore, conversations *conversation.Service) *Server {
	s := &Server{
		streams:       streams,
		conversations: conversations,
	}

	smtpSrv := smtp.NewServer(s)
	smtpSrv.Addr = addr
	smtpSrv.Domain = domain
	smtpSrv.ReadTimeout = 30 * time.Second
	smtpSrv.WriteTimeout = 30 * time.Second
	smtpSrv.MaxMessageBytes = 10 * 1024 * 1024 // 10MB
	smtpSrv.MaxRecipients = 1
	smtpSrv.AllowInsecureAuth = true

	s.smtpServer = smtpSrv
	return s
}

func (s *Server) Start() error {
	slog.Info("inbound SMTP server starting", "addr", s.smtpServer.Addr)
	return s.smtpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.smtpServer.Close()
}

// NewSession implements smtp.Backend.
func (s *Server) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &session{server: s}, nil
}

type session struct {
	server *Server
	from   string
	to     string
	stream *models.Stream
}

func (s *session) Mail(from string, _ *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	// Look up the stream by recipient address.
	addr := strings.ToLower(strings.TrimSpace(to))
	stream, err := s.server.streams.GetStreamByAddress(context.Background(), addr)
	if err != nil {
		slog.Warn("inbound email to unknown address", "to", addr)
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "no such recipient",
		}
	}

	if !stream.Enabled {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 1, 1},
			Message:      "recipient disabled",
		}
	}

	s.to = addr
	s.stream = stream
	return nil
}

func (s *session) Data(r io.Reader) error {
	if s.stream == nil {
		return errors.New("no valid recipient")
	}

	body, err := io.ReadAll(io.LimitReader(r, 10*1024*1024))
	if err != nil {
		return err
	}

	// Parse the email to extract subject and plain text body.
	subject, plainBody := parseEmail(body)

	_, err = s.server.conversations.StartConversation(
		context.Background(),
		s.stream,
		subject,
		s.from,
		"", // sender name extracted from email headers if available
		plainBody,
	)
	if err != nil {
		slog.Error("failed to create conversation from inbound email",
			"from", s.from, "to", s.to, "error", err)
		return err
	}

	slog.Info("inbound email processed", "from", s.from, "to", s.to, "subject", subject)
	return nil
}

func (s *session) Reset() {
	s.from = ""
	s.to = ""
	s.stream = nil
}

func (s *session) Logout() error {
	return nil
}

// parseEmail extracts subject and plain text body from raw email bytes.
func parseEmail(raw []byte) (subject, body string) {
	lines := strings.SplitN(string(raw), "\r\n\r\n", 2)
	if len(lines) == 2 {
		body = lines[1]
	}
	for _, line := range strings.Split(lines[0], "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			subject = strings.TrimSpace(line[8:])
		}
	}
	return subject, body
}
