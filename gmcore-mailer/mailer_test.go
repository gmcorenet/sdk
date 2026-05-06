package gmcore_mailer

import (
	"bytes"
	"crypto/tls"
	"strings"
	"testing"
)

func TestSanitizeHeaderValue_RejectsCRLF(t *testing.T) {
	tests := []string{
		"value\r\n",
		"value\r",
		"value\n",
		"before\r\nafter",
		"line1\nline2",
		"line1\rline2",
	}
	for _, input := range tests {
		_, err := sanitizeHeaderValue(input)
		if err == nil || !strings.Contains(err.Error(), "CR/LF") {
			t.Errorf("sanitizeHeaderValue(%q) should reject CRLF, got err=%v", input, err)
		}
	}
}

func TestSanitizeHeaderValue_AcceptsValid(t *testing.T) {
	tests := []string{
		"valid value",
		"Subject: Hello",
		"simple text",
		"",
		"with spaces and numbers 123",
	}
	for _, input := range tests {
		val, err := sanitizeHeaderValue(input)
		if err != nil {
			t.Errorf("sanitizeHeaderValue(%q) should accept, got err=%v", input, err)
		}
		if val != input {
			t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", input, val, input)
		}
	}
}

func TestSMTPMailer_Send_NoRecipients(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for no recipients")
	}
}

func TestSMTPMailer_Send_InvalidFrom(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com\r\nBcc: victim@evil.com",
		To:      []string{"to@example.com"},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for invalid From with CRLF")
	}
}

func TestSMTPMailer_Send_InvalidTo(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{"to@example.com\r\nBcc: victim@evil.com"},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for invalid To with CRLF")
	}
}

func TestSMTPMailer_Send_InvalidSubject(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Subject: "Subject\r\nBcc: victim@evil.com",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for invalid Subject with CRLF")
	}
}

func TestMemoryMailer_Send_NoRecipients(t *testing.T) {
	m := NewMemoryMailer()
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for no recipients")
	}
}

func TestMemoryMailer_Send_Valid(t *testing.T) {
	m := NewMemoryMailer()
	email := NewEmail("from@example.com", "to@example.com", "subject", "body")
	err := m.Send(email)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	emails := m.GetEmails()
	if len(emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(emails))
	}
}

func TestMemoryMailer_Clear(t *testing.T) {
	m := NewMemoryMailer()
	m.Send(NewEmail("from@example.com", "to@example.com", "s", "b"))
	m.Clear()
	if len(m.GetEmails()) != 0 {
		t.Error("expected 0 emails after clear")
	}
}

func TestNewEmail(t *testing.T) {
	email := NewEmail("a@b.com", "c@d.com", "subject", "body")
	if email.From != "a@b.com" {
		t.Errorf("expected From=a@b.com, got %s", email.From)
	}
	if len(email.To) != 1 || email.To[0] != "c@d.com" {
		t.Errorf("expected To=[c@d.com], got %v", email.To)
	}
	if email.Subject != "subject" {
		t.Errorf("expected Subject=subject, got %s", email.Subject)
	}
	if email.Body != "body" {
		t.Errorf("expected Body=body, got %s", email.Body)
	}
}

func TestEmail_AddTo(t *testing.T) {
	e := NewEmail("a@b.com", "c@d.com", "s", "b")
	e.AddTo("e@f.com")
	if len(e.To) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(e.To))
	}
}

func TestEmail_AddCc(t *testing.T) {
	e := NewEmail("a@b.com", "c@d.com", "s", "b")
	e.AddCc("cc@example.com")
	if len(e.Cc) != 1 {
		t.Errorf("expected 1 Cc, got %d", len(e.Cc))
	}
}

func TestEmail_AddBcc(t *testing.T) {
	e := NewEmail("a@b.com", "c@d.com", "s", "b")
	e.AddBcc("bcc@example.com")
	if len(e.Bcc) != 1 {
		t.Errorf("expected 1 Bcc, got %d", len(e.Bcc))
	}
}

func TestEmail_SetHtml(t *testing.T) {
	e := NewEmail("a@b.com", "c@d.com", "s", "b")
	e.SetHtml(true)
	if !e.Html {
		t.Error("expected Html=true")
	}
}

func TestEmail_AddAttachment(t *testing.T) {
	e := NewEmail("a@b.com", "c@d.com", "s", "b")
	e.AddAttachment("file.txt", []byte("content"))
	if len(e.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(e.Attachments))
	}
	if e.Attachments[0].Name != "file.txt" {
		t.Errorf("expected name=file.txt, got %s", e.Attachments[0].Name)
	}
}

func TestBuildSimpleMessage(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	var buf bytes.Buffer
	email := &Email{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "Test Subject",
		Body:    "Test Body",
		Html:    false,
	}
	err := m.buildSimpleMessage(&buf, email)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "From: from@example.com") {
		t.Error("expected From header")
	}
	if !strings.Contains(output, "To: to@example.com") {
		t.Error("expected To header")
	}
	if !strings.Contains(output, "Cc: cc@example.com") {
		t.Error("expected Cc header")
	}
	if !strings.Contains(output, "Subject: Test Subject") {
		t.Error("expected Subject header")
	}
	if !strings.Contains(output, "Test Body") {
		t.Error("expected body content")
	}
}

func TestBuildSimpleMessage_Html(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	var buf bytes.Buffer
	email := &Email{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Subject: "HTML Email",
		Body:    "<html><body>Hello</body></html>",
		Html:    true,
	}
	err := m.buildSimpleMessage(&buf, email)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "text/html") {
		t.Error("expected text/html content type for Html=true")
	}
}

func TestBuildMultipartMessage(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	var buf bytes.Buffer
	email := &Email{
		From:        "from@example.com",
		To:          []string{"to@example.com"},
		Subject:     "With Attachment",
		Body:        "Hello",
		Attachments: []Attachment{{Name: "file.txt", Content: []byte("hello world")}},
	}
	err := m.buildMultipartMessage(&buf, email)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "multipart/mixed") {
		t.Error("expected multipart/mixed content type")
	}
}

func TestSMTPMailer_Send_CRLFInjectionInBcc(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Bcc:     []string{"bcc\r\nBcc: victim@evil.com"},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for CRLF in Bcc header")
	}
}

func TestSMTPMailer_Send_CRLFInjectionInCc(t *testing.T) {
	m := NewSMTPMailer("localhost", 1025, "", "")
	err := m.Send(&Email{
		From:    "from@example.com",
		To:      []string{"to@example.com"},
		Cc:      []string{"cc\r\nBcc: victim@evil.com"},
		Subject: "test",
		Body:    "body",
	})
	if err == nil {
		t.Error("expected error for CRLF in Cc header")
	}
}

func TestSMTPMailer_WithTLS(t *testing.T) {
	m := NewSMTPMailer("localhost", 465, "user", "pass")
	tlsCfg := &tls.Config{ServerName: "localhost"}
	m2 := m.WithTLS(tlsCfg)
	if m2 != m {
		t.Error("WithTLS should return same mailer")
	}
	if m.tls == nil {
		t.Error("tls should be set")
	}
}
