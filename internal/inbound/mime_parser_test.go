package inbound

import "testing"

func TestParseRFC822_TextEmail(t *testing.T) {
	raw := "From: Sender <sender@outside.com>\r\nTo: ideas@example.com\r\nSubject: Hello\r\nMessage-ID: <abc@test>\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello world"
	msg, err := ParseRFC822(raw, 1024*1024)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msg.Sender == "" {
		t.Fatalf("expected sender")
	}
	if len(msg.Recipients) != 1 || msg.Recipients[0] != "ideas@example.com" {
		t.Fatalf("unexpected recipients: %+v", msg.Recipients)
	}
	if msg.TextBody != "Hello world" {
		t.Fatalf("unexpected body: %q", msg.TextBody)
	}
	if msg.Subject != "Hello" {
		t.Fatalf("unexpected subject: %q", msg.Subject)
	}
}

func TestParseRFC822_Attachment(t *testing.T) {
	raw := "From: Sender <sender@outside.com>\r\n" +
		"To: ideas@example.com\r\n" +
		"Subject: Multipart\r\n" +
		"Message-ID: <abc2@test>\r\n" +
		"Content-Type: multipart/mixed; boundary=abc123\r\n\r\n" +
		"--abc123\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		"Body text\r\n" +
		"--abc123\r\n" +
		"Content-Type: text/plain; name=\"note.txt\"\r\n" +
		"Content-Disposition: attachment; filename=\"note.txt\"\r\n\r\n" +
		"Attachment data\r\n" +
		"--abc123--\r\n"

	msg, err := ParseRFC822(raw, 1024*1024)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msg.TextBody != "Body text" {
		t.Fatalf("unexpected text body: %q", msg.TextBody)
	}
	if len(msg.Attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(msg.Attachments))
	}
	if msg.Attachments[0].FileName != "note.txt" {
		t.Fatalf("unexpected filename: %q", msg.Attachments[0].FileName)
	}
}
