package inbound

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"html"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"regexp"
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
	email := parseEmail(body, s.from)

	_, err = s.server.conversations.StartConversation(
		context.Background(),
		s.stream,
		email.Subject,
		email.SenderAddress,
		email.SenderName,
		email.Body,
	)
	if err != nil {
		slog.Error("failed to create conversation from inbound email",
			"from", s.from, "to", s.to, "error", err)
		return err
	}

	slog.Info("inbound email processed",
		"from", email.SenderAddress,
		"to", s.to,
		"subject", email.Subject,
	)
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

type parsedEmail struct {
	Subject       string
	SenderAddress string
	SenderName    string
	Body          string
}

var (
	scriptStyleTagRe = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	htmlTagRe        = regexp.MustCompile(`(?s)<[^>]+>`)
)

// parseEmail extracts sender, subject, and a readable text body from raw MIME email bytes.
func parseEmail(raw []byte, envelopeFrom string) parsedEmail {
	email := parsedEmail{
		SenderAddress: strings.TrimSpace(envelopeFrom),
	}

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		email.Body = fallbackBody(raw)
		return email
	}

	email.Subject = decodeHeaderValue(msg.Header.Get("Subject"))
	addr, name := parseFrom(msg.Header.Get("From"))
	if addr != "" {
		email.SenderAddress = addr
	}
	email.SenderName = name

	body, err := extractBodyFromHeader(textproto.MIMEHeader(msg.Header), msg.Body)
	if err != nil {
		email.Body = fallbackBody(raw)
	} else {
		email.Body = strings.TrimSpace(body)
	}

	if email.Body == "" {
		email.Body = fallbackBody(raw)
	}

	return email
}

func parseFrom(rawFrom string) (address, name string) {
	if strings.TrimSpace(rawFrom) == "" {
		return "", ""
	}
	addr, err := mail.ParseAddress(rawFrom)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(addr.Address), decodeHeaderValue(addr.Name)
}

func decodeHeaderValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	decoded, err := new(mime.WordDecoder).DecodeHeader(v)
	if err != nil {
		return v
	}

	return strings.TrimSpace(decoded)
}

func extractBodyFromHeader(header textproto.MIMEHeader, body io.Reader) (string, error) {
	contentType := header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "text/plain; charset=utf-8"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	encoding := strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding")))
	decodedReader := decodeTransferEncoding(body, encoding)

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			b, readErr := io.ReadAll(decodedReader)
			if readErr != nil {
				return "", readErr
			}
			return strings.TrimSpace(string(b)), nil
		}

		mr := multipart.NewReader(decodedReader, boundary)
		var plainBody string
		var htmlBody string

		for {
			part, partErr := mr.NextPart()
			if errors.Is(partErr, io.EOF) {
				break
			}
			if partErr != nil {
				return "", partErr
			}

			partBody, extractErr := extractBodyFromHeader(part.Header, part)
			_ = part.Close()
			if extractErr != nil {
				continue
			}

			partContentType := strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Type")))
			if strings.Contains(partContentType, "text/plain") {
				if strings.TrimSpace(partBody) != "" && plainBody == "" {
					plainBody = partBody
				}
				continue
			}

			if strings.Contains(partContentType, "text/html") {
				if strings.TrimSpace(partBody) != "" && htmlBody == "" {
					htmlBody = partBody
				}
				continue
			}

			if strings.TrimSpace(partBody) != "" && plainBody == "" {
				plainBody = partBody
			}
		}

		if strings.TrimSpace(plainBody) != "" {
			return strings.TrimSpace(plainBody), nil
		}
		if strings.TrimSpace(htmlBody) != "" {
			return strings.TrimSpace(htmlBody), nil
		}
		return "", nil
	}

	b, err := io.ReadAll(decodedReader)
	if err != nil {
		return "", err
	}
	text := string(b)

	switch mediaType {
	case "text/html":
		return strings.TrimSpace(htmlToText(text)), nil
	case "message/rfc822":
		nested, nestedErr := mail.ReadMessage(bytes.NewReader(b))
		if nestedErr != nil {
			return strings.TrimSpace(text), nil
		}
		return extractBodyFromHeader(textproto.MIMEHeader(nested.Header), nested.Body)
	default:
		return strings.TrimSpace(text), nil
	}
}

func decodeTransferEncoding(body io.Reader, encoding string) io.Reader {
	switch encoding {
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, body)
	case "quoted-printable":
		return quotedprintable.NewReader(body)
	default:
		return body
	}
}

func htmlToText(s string) string {
	s = scriptStyleTagRe.ReplaceAllString(s, "")
	lineBreakReplacer := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"</p>", "\n",
		"</div>", "\n",
		"</li>", "\n",
		"<li>", "\n- ",
		"</tr>", "\n",
		"</h1>", "\n",
		"</h2>", "\n",
		"</h3>", "\n",
		"</h4>", "\n",
		"</h5>", "\n",
		"</h6>", "\n",
	)
	s = lineBreakReplacer.Replace(s)
	s = htmlTagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(out) > 0 && out[len(out)-1] != "" {
				out = append(out, "")
			}
			continue
		}
		out = append(out, trimmed)
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}

func fallbackBody(raw []byte) string {
	parts := strings.SplitN(string(raw), "\r\n\r\n", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(string(raw))
}
