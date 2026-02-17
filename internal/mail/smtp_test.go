package mail

import (
	"net/smtp"
	"strings"
	"testing"
)

func withStubSendMail(t *testing.T, stub func(addr string, a smtp.Auth, from string, to []string, msg []byte) error) {
	t.Helper()
	orig := smtpSendMail
	smtpSendMail = stub
	t.Cleanup(func() {
		smtpSendMail = orig
	})
}

func TestSMTPClientSend_NoAuthWhenCredentialsBlank(t *testing.T) {
	client := NewSMTPClient("smtp.example.com", 25, "", "", "no-reply@example.com")

	withStubSendMail(t, func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		if addr != "smtp.example.com:25" {
			t.Fatalf("unexpected addr: %s", addr)
		}
		if a != nil {
			t.Fatal("expected nil auth when credentials are blank")
		}
		if from != "no-reply@example.com" {
			t.Fatalf("unexpected envelope from: %s", from)
		}
		if len(to) != 1 || to[0] != "user@example.com" {
			t.Fatalf("unexpected recipients: %v", to)
		}
		if !strings.Contains(string(msg), "From: no-reply@example.com\r\n") {
			t.Fatalf("expected From header in message, got %q", string(msg))
		}
		return nil
	})

	if err := client.Send("user@example.com", "Subject", "<p>Body</p>"); err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
}

func TestSMTPClientSendFrom_UsesEnvelopeAndHeaderSeparately(t *testing.T) {
	client := NewSMTPClient("smtp.example.com", 25, "", "", "fallback@example.com")

	withStubSendMail(t, func(_ string, _ smtp.Auth, from string, _ []string, msg []byte) error {
		if from != "support@example.com" {
			t.Fatalf("expected envelope from support@example.com, got %s", from)
		}
		if !strings.Contains(string(msg), "From: Support Team <support@example.com>\r\n") {
			t.Fatalf("expected display name in header From, got %q", string(msg))
		}
		return nil
	})

	if err := client.SendFrom("support@example.com", "Support Team <support@example.com>", "user@example.com", "Re: Help", "<p>Reply</p>"); err != nil {
		t.Fatalf("SendFrom returned error: %v", err)
	}
}

func TestSMTPClientSend_IncompleteCredentialsFail(t *testing.T) {
	client := NewSMTPClient("smtp.example.com", 587, "user-only", "", "no-reply@example.com")
	err := client.Send("user@example.com", "Subject", "<p>Body</p>")
	if err == nil {
		t.Fatal("expected error for incomplete SMTP credentials")
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete credentials error, got %v", err)
	}
}

