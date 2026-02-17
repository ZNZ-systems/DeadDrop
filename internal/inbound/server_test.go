package inbound

import (
	"strings"
	"testing"
)

func TestParseEmail(t *testing.T) {
	raw := []byte("Subject: Test Subject\r\nFrom: Alice <alice@example.com>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello world")
	email := parseEmail(raw, "envelope@example.com")
	if email.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got '%s'", email.Subject)
	}
	if email.Body != "Hello world" {
		t.Errorf("expected body 'Hello world', got '%s'", email.Body)
	}
	if email.SenderAddress != "alice@example.com" {
		t.Errorf("expected sender address alice@example.com, got '%s'", email.SenderAddress)
	}
	if email.SenderName != "Alice" {
		t.Errorf("expected sender name Alice, got '%s'", email.SenderName)
	}
}

func TestParseEmail_NoSubject(t *testing.T) {
	raw := []byte("From: test@example.com\r\n\r\nJust a body")
	email := parseEmail(raw, "bounce@example.com")
	if email.Subject != "" {
		t.Errorf("expected empty subject, got '%s'", email.Subject)
	}
	if email.Body != "Just a body" {
		t.Errorf("expected body 'Just a body', got '%s'", email.Body)
	}
	if email.SenderAddress != "test@example.com" {
		t.Errorf("expected sender address test@example.com, got '%s'", email.SenderAddress)
	}
}

func TestParseEmail_MultipartAlternativePrefersText(t *testing.T) {
	raw := strings.Join([]string{
		"From: Test Sender <sender@example.com>",
		"Subject: =?UTF-8?Q?Test_=E2=9C=93?=",
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=\"alt\"",
		"",
		"--alt",
		"Content-Type: text/plain; charset=\"utf-8\"",
		"Content-Transfer-Encoding: 7bit",
		"",
		"Plain body",
		"",
		"--alt",
		"Content-Type: text/html; charset=\"utf-8\"",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"<div>HTML body</div>",
		"",
		"--alt--",
		"",
	}, "\r\n")

	email := parseEmail([]byte(raw), "envelope@example.com")
	if email.Subject != "Test ✓" {
		t.Errorf("expected decoded subject 'Test ✓', got '%s'", email.Subject)
	}
	if email.Body != "Plain body" {
		t.Errorf("expected plain body, got '%s'", email.Body)
	}
}

func TestParseEmail_HTMLOnlyBody(t *testing.T) {
	raw := strings.Join([]string{
		"From: html@example.com",
		"Content-Type: text/html; charset=\"utf-8\"",
		"Content-Transfer-Encoding: quoted-printable",
		"",
		"<html><body><div><span style=3D\"font-size:14px\">test</span></div></body></html>",
		"",
	}, "\r\n")

	email := parseEmail([]byte(raw), "envelope@example.com")
	if email.Body != "test" {
		t.Errorf("expected html body converted to text 'test', got '%s'", email.Body)
	}
}
