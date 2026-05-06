package gmcore_notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrSlackTokenMissing = errors.New("slack token is missing")
	ErrSlackSendFailed   = errors.New("failed to send slack message")
)

type Notification struct {
	Subject   string
	Content   string
	Importance Importance
	Channels  []string
}

type Importance int

const (
	ImportanceLow    Importance = 0
	ImportanceMedium Importance = 1
	ImportanceHigh   Importance = 2
)

type Channel interface {
	Send(notification *Notification) error
}

type Notifier interface {
	Send(notification *Notification, channels ...string) []SentMessage
}

type SentMessage struct {
	Channel  string
	Sent     bool
	Error    error
}

type notifier struct {
	channels map[string]Channel
}

func NewNotifier() *notifier {
	return &notifier{channels: make(map[string]Channel)}
}

func (n *notifier) AddChannel(name string, channel Channel) {
	n.channels[name] = channel
}

func (n *notifier) Send(notification *Notification, channels ...string) []SentMessage {
	results := make([]SentMessage, 0)

	if len(channels) == 0 {
		for name, ch := range n.channels {
			err := ch.Send(notification)
			results = append(results, SentMessage{Channel: name, Sent: err == nil, Error: err})
		}
	} else {
		for _, name := range channels {
			if ch, ok := n.channels[name]; ok {
				err := ch.Send(notification)
				results = append(results, SentMessage{Channel: name, Sent: err == nil, Error: err})
			}
		}
	}

	return results
}

type EmailChannel struct {
	mailer interface{ Send(email *struct{ To, Subject, Body string }) error }
}

func NewEmailChannel(mailer interface{ Send(email *struct{ To, Subject, Body string }) error }) *EmailChannel {
	return &EmailChannel{mailer: mailer}
}

func (c *EmailChannel) Send(notification *Notification) error {
	if len(notification.Channels) == 0 {
		return errors.New("no email channels specified in notification")
	}
	recipients := strings.Join(notification.Channels, ", ")
	return c.mailer.Send(&struct{ To, Subject, Body string }{To: recipients, Subject: notification.Subject, Body: notification.Content})
}

type SlackChannel struct {
	token   string
	channel string
	client  *http.Client
}

func NewSlackChannel(token, channel string) *SlackChannel {
	return &SlackChannel{
		token:   token,
		channel: channel,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type slackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func (c *SlackChannel) Send(notification *Notification) error {
	if c.token == "" {
		return ErrSlackTokenMissing
	}

	msg := slackMessage{
		Channel: c.channel,
		Text:    fmt.Sprintf("*%s*\n%s", notification.Subject, notification.Content),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrSlackSendFailed, resp.StatusCode)
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode slack response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("%w: %s", ErrSlackSendFailed, result.Error)
	}

	return nil
}

func NewNotification(subject, content string) *Notification {
	return &Notification{Subject: subject, Content: content, Importance: ImportanceMedium}
}

func (n *Notification) SetImportance(i Importance) { n.Importance = i }
func (n *Notification) AddChannel(ch string)        { n.Channels = append(n.Channels, ch) }
