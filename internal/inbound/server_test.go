package inbound

import (
	"testing"
)

func TestParseEmail(t *testing.T) {
	raw := []byte("Subject: Test Subject\r\nFrom: test@example.com\r\n\r\nHello world")
	subject, body := parseEmail(raw)
	if subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got '%s'", subject)
	}
	if body != "Hello world" {
		t.Errorf("expected body 'Hello world', got '%s'", body)
	}
}

func TestParseEmail_NoSubject(t *testing.T) {
	raw := []byte("From: test@example.com\r\n\r\nJust a body")
	subject, body := parseEmail(raw)
	if subject != "" {
		t.Errorf("expected empty subject, got '%s'", subject)
	}
	if body != "Just a body" {
		t.Errorf("expected body 'Just a body', got '%s'", body)
	}
}
