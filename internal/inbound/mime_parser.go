package inbound

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"path/filepath"
	"strings"
)

const defaultMaxAttachmentBytes int64 = 5 * 1024 * 1024

func ParseRFC822(raw string, maxAttachmentBytes int64) (Message, error) {
	if strings.TrimSpace(raw) == "" {
		return Message{}, fmt.Errorf("raw RFC822 payload is empty")
	}
	if maxAttachmentBytes <= 0 {
		maxAttachmentBytes = defaultMaxAttachmentBytes
	}

	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return Message{}, fmt.Errorf("parse message: %w", err)
	}

	result := Message{
		Sender:    strings.TrimSpace(msg.Header.Get("From")),
		Subject:   strings.TrimSpace(msg.Header.Get("Subject")),
		MessageID: strings.TrimSpace(firstNonEmpty(msg.Header.Get("Message-ID"), msg.Header.Get("Message-Id"))),
		RawRFC822: raw,
	}
	result.Recipients = parseRecipientsFromHeaders(msg.Header)

	contentType := strings.TrimSpace(msg.Header.Get("Content-Type"))
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = ""
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") && params["boundary"] != "" {
		reader := multipart.NewReader(msg.Body, params["boundary"])
		if err := parseMultipart(reader, &result, maxAttachmentBytes); err != nil {
			return Message{}, err
		}
		return result, nil
	}

	decoded, err := decodeBody(msg.Header.Get("Content-Transfer-Encoding"), msg.Body)
	if err != nil {
		return Message{}, fmt.Errorf("decode body: %w", err)
	}
	if strings.Contains(strings.ToLower(mediaType), "text/html") {
		result.HTMLBody = strings.TrimSpace(string(decoded))
	} else {
		result.TextBody = strings.TrimSpace(string(decoded))
	}
	return result, nil
}

func parseMultipart(reader *multipart.Reader, msg *Message, maxAttachmentBytes int64) error {
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read multipart part: %w", err)
		}

		contentType := strings.TrimSpace(part.Header.Get("Content-Type"))
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			mediaType = ""
			params = map[string]string{}
		}

		contentDisposition, dispParams, err := mime.ParseMediaType(strings.TrimSpace(part.Header.Get("Content-Disposition")))
		if err != nil {
			contentDisposition = ""
			dispParams = map[string]string{}
		}

		if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") && params["boundary"] != "" {
			nested := multipart.NewReader(part, params["boundary"])
			if err := parseMultipart(nested, msg, maxAttachmentBytes); err != nil {
				return err
			}
			continue
		}

		payload, err := decodeBody(part.Header.Get("Content-Transfer-Encoding"), io.LimitReader(part, maxAttachmentBytes+1))
		if err != nil {
			return fmt.Errorf("decode multipart part: %w", err)
		}
		if int64(len(payload)) > maxAttachmentBytes {
			continue
		}

		rawFilename := firstNonEmpty(dispParams["filename"], params["name"], part.FileName())
		if isAttachmentPart(contentDisposition, rawFilename) {
			filename := sanitizeFilename(rawFilename)
			msg.Attachments = append(msg.Attachments, Attachment{
				FileName:    filename,
				ContentType: normalizeContentType(mediaType),
				Content:     payload,
			})
			continue
		}

		switch strings.ToLower(mediaType) {
		case "text/plain":
			text := strings.TrimSpace(string(payload))
			if text != "" {
				if msg.TextBody != "" {
					msg.TextBody += "\n\n"
				}
				msg.TextBody += text
			}
		case "text/html":
			html := strings.TrimSpace(string(payload))
			if html != "" {
				if msg.HTMLBody != "" {
					msg.HTMLBody += "\n"
				}
				msg.HTMLBody += html
			}
		}
	}
}

func decodeBody(encoding string, r io.Reader) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "base64":
		return io.ReadAll(base64.NewDecoder(base64.StdEncoding, r))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(r))
	default:
		return io.ReadAll(r)
	}
}

func parseRecipientsFromHeaders(header mail.Header) []string {
	uniq := map[string]struct{}{}
	ordered := make([]string, 0, 8)
	for _, key := range []string{"To", "Cc", "Bcc"} {
		raw := strings.TrimSpace(header.Get(key))
		if raw == "" {
			continue
		}
		list, err := mail.ParseAddressList(raw)
		if err != nil {
			continue
		}
		for _, addr := range list {
			email := strings.ToLower(strings.TrimSpace(addr.Address))
			if email == "" {
				continue
			}
			if _, exists := uniq[email]; exists {
				continue
			}
			uniq[email] = struct{}{}
			ordered = append(ordered, email)
		}
	}
	return ordered
}

func isAttachmentPart(disposition, filename string) bool {
	if strings.EqualFold(strings.TrimSpace(disposition), "attachment") {
		return true
	}
	return strings.TrimSpace(filename) != ""
}

func normalizeContentType(mediaType string) string {
	mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	if mediaType == "" {
		return "application/octet-stream"
	}
	return mediaType
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "attachment.bin"
	}
	base := filepath.Base(name)
	base = strings.TrimSpace(base)
	if base == "." || base == "/" || base == "" {
		return "attachment.bin"
	}
	return base
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func encodeRFC822ForJSON(raw string) string {
	// retained for potential debugging hooks; kept private utility.
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeRFC822FromJSON(encoded string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(b)), nil
}
