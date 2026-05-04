package gmcore_mailer

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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
}

func NewSMTPMailer(host string, port int, username, password string) *SMTPMailer {
	return &SMTPMailer{host: host, port: port, username: username, password: password}
}

func (m *SMTPMailer) Send(email *Email) error {
	if len(email.To) == 0 {
		return errors.New("email must have at least one recipient")
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

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", m.host, m.port),
		auth,
		email.From,
		allRecipients,
		msg.Bytes(),
	)
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
