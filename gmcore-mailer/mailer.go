package gmcore_mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Email struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	Html        bool
	Attachments []Attachment
}

type Attachment struct {
	Name    string
	Content []byte
}

type Mailer interface {
	Send(email *Email) error
}

type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	tls      *tls.Config
}

func NewSMTPMailer(host string, port int, username, password string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password}
}

func (m *SMTPMailer) WithTLS(tlsCfg *tls.Config) *SMTPMailer {
	m.tls = tlsCfg
	return m
}

func sanitizeHeaderValue(value string) (string, error) {
	if strings.ContainsAny(value, "\r\n") {
		return "", errors.New("header value contains invalid characters (CR/LF)")
	}
	return value, nil
}

func (m *SMTPMailer) Send(email *Email) error {
	if len(email.To) == 0 {
		return errors.New("email must have at least one recipient")
	}

	if _, err := sanitizeHeaderValue(email.From); err != nil {
		return fmt.Errorf("invalid From header: %w", err)
	}
	for _, to := range email.To {
		if _, err := sanitizeHeaderValue(to); err != nil {
			return fmt.Errorf("invalid To header: %w", err)
		}
	}
	for _, cc := range email.Cc {
		if _, err := sanitizeHeaderValue(cc); err != nil {
			return fmt.Errorf("invalid Cc header: %w", err)
		}
	}
	for _, bcc := range email.Bcc {
		if _, err := sanitizeHeaderValue(bcc); err != nil {
			return fmt.Errorf("invalid Bcc header: %w", err)
		}
	}
	if _, err := sanitizeHeaderValue(email.Subject); err != nil {
		return fmt.Errorf("invalid Subject header: %w", err)
	}

	var msg bytes.Buffer

	if len(email.Attachments) > 0 {
		if err := m.buildMultipartMessage(&msg, email); err != nil {
			return err
		}
	} else {
		if err := m.buildSimpleMessage(&msg, email); err != nil {
			return err
		}
	}

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	allRecipients := make([]string, 0, len(email.To)+len(email.Cc)+len(email.Bcc))
	allRecipients = append(allRecipients, email.To...)
	allRecipients = append(allRecipients, email.Cc...)
	allRecipients = append(allRecipients, email.Bcc...)

	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	if m.port == 465 && m.tls != nil {
		return m.sendWithImplicitTLS(addr, auth, email.From, allRecipients, msg.Bytes())
	}

	if m.port == 587 || m.tls != nil {
		return m.sendWithSTARTTLS(addr, auth, email.From, allRecipients, msg.Bytes())
	}

	return smtp.SendMail(addr, auth, email.From, allRecipients, msg.Bytes())
}

func (m *SMTPMailer) sendWithImplicitTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsCfg := m.tls
	if tlsCfg == nil {
		tlsCfg = &tls.Config{ServerName: m.host}
	}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	return m.sendWithClient(client, auth, from, to, msg)
}

func (m *SMTPMailer) sendWithSTARTTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	tlsCfg := m.tls
	if tlsCfg == nil {
		tlsCfg = &tls.Config{ServerName: m.host}
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	} else if m.tls != nil {
		return errors.New("server does not support STARTTLS")
	}

	return m.sendWithClient(client, auth, from, to, msg)
}

func (m *SMTPMailer) sendWithClient(client *smtp.Client, auth smtp.Auth, from string, to []string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

func (m *SMTPMailer) buildSimpleMessage(msg *bytes.Buffer, email *Email) error {
	headers := make(map[string]string)
	headers["From"] = email.From
	headers["To"] = strings.Join(email.To, ", ")
	if len(email.Cc) > 0 {
		headers["Cc"] = strings.Join(email.Cc, ", ")
	}
	headers["Subject"] = email.Subject

	if email.Html {
		headers["Content-Type"] = "text/html; charset=utf-8"
	} else {
		headers["Content-Type"] = "text/plain; charset=utf-8"
	}

	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(email.Body)

	return nil
}

func (m *SMTPMailer) buildMultipartMessage(msg *bytes.Buffer, email *Email) error {
	boundary := mime.BEncoding.Encode("b", fmt.Sprintf("%d", time.Now().UnixNano()))

	headers := make(map[string]string)
	headers["From"] = email.From
	headers["To"] = strings.Join(email.To, ", ")
	if len(email.Cc) > 0 {
		headers["Cc"] = strings.Join(email.Cc, ", ")
	}
	headers["Subject"] = email.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=\"%s\"", boundary)

	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")

	writer := multipart.NewWriter(msg)
	writer.SetBoundary(boundary)

	contentType := "text/plain"
	if email.Html {
		contentType = "text/html"
	}
	part, err := writer.CreatePart(map[string][]string{
		"Content-Type": {contentType + "; charset=utf-8"},
	})
	if err != nil {
		return err
	}
	io.WriteString(part, email.Body)

	for _, attachment := range email.Attachments {
		attContentType := mime.TypeByExtension(attachment.Name)
		if attContentType == "" {
			attContentType = "application/octet-stream"
		}
		attPart, err := writer.CreatePart(map[string][]string{
			"Content-Type":              {attContentType},
			"Content-Disposition":      {fmt.Sprintf(`attachment; filename="%s"`, attachment.Name)},
			"Content-Transfer-Encoding": {"base64"},
		})
		if err != nil {
			return err
		}

		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Content)))
		base64.StdEncoding.Encode(encoded, attachment.Content)
		attPart.Write(encoded)
	}

	writer.Close()
	return nil
}

type MemoryMailer struct {
	emails []*Email
}

func NewMemoryMailer() *MemoryMailer {
	return &MemoryMailer{emails: make([]*Email, 0)}
}

func (m *MemoryMailer) Send(email *Email) error {
	if len(email.To) == 0 {
		return errors.New("email must have at least one recipient")
	}
	m.emails = append(m.emails, email)
	return nil
}

func (m *MemoryMailer) GetEmails() []*Email {
	result := make([]*Email, len(m.emails))
	copy(result, m.emails)
	return result
}

func (m *MemoryMailer) Clear() {
	m.emails = make([]*Email, 0)
}

func NewEmail(from, to, subject, body string) *Email {
	return &Email{
		From: from,
		To:   []string{to},
		Subject: subject,
		Body: body,
	}
}

func (e *Email) AddTo(to string)              { e.To = append(e.To, to) }
func (e *Email) AddCc(cc string)             { e.Cc = append(e.Cc, cc) }
func (e *Email) AddBcc(bcc string)           { e.Bcc = append(e.Bcc, bcc) }
func (e *Email) SetHtml(html bool)           { e.Html = html }
func (e *Email) AddAttachment(name string, content []byte) {
	e.Attachments = append(e.Attachments, Attachment{Name: name, Content: content})
}
