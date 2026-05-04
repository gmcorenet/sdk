package gmcore_webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

type Webhook struct {
	URL     string
	Secret  string
	Events  []string
	Headers map[string]string
	Retry   RetryPolicy
}

type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
	Backoff     BackoffStrategy
}

type BackoffStrategy func(attempt int) time.Duration

func LinearBackoff(attempt int) time.Duration {
	return time.Duration(attempt) * time.Second
}

func ExponentialBackoff(attempt int) time.Duration {
	return time.Duration(1<<attempt) * time.Second
}

type WebhookManager interface {
	Send(webhook *Webhook, payload interface{}) (*Result, error)
	Register(webhook *Webhook) error
	Unregister(url string) error
}

type Result struct {
	Success    bool
	StatusCode int
	Response   []byte
	Error      error
	Attempts   int
}

type manager struct {
	webhooks map[string]*Webhook
	client   *http.Client
}

func NewManager() *manager {
	return &manager{
		webhooks: make(map[string]*Webhook),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *manager) Register(webhook *Webhook) error {
	m.webhooks[webhook.URL] = webhook
	return nil
}

func (m *manager) Unregister(url string) error {
	delete(m.webhooks, url)
	return nil
}

func (m *manager) Send(webhook *Webhook, payload interface{}) (*Result, error) {
	if webhook.Retry.MaxAttempts <= 0 {
		webhook.Retry.MaxAttempts = 1
	}
	if webhook.Retry.Backoff == nil {
		webhook.Retry.Backoff = LinearBackoff
	}

	var lastErr error
	for attempt := 0; attempt < webhook.Retry.MaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(webhook.Retry.Backoff(attempt))
		}

		result, err := m.doSend(webhook, payload)
		if err == nil {
			result.Attempts = attempt + 1
			return result, nil
		}
		lastErr = err
	}

	return &Result{Success: false, Error: lastErr, Attempts: webhook.Retry.MaxAttempts}, lastErr
}

func (m *manager) doSend(webhook *Webhook, payload interface{}) (*Result, error) {
	body, err := marshalPayload(payload)
	if err != nil {
		return &Result{Success: false, Error: err}, err
	}

	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(body))
	if err != nil {
		return &Result{Success: false, Error: err}, err
	}
	req.Header.Set("Content-Type", "application/json")

	for k, v := range webhook.Headers {
		req.Header.Set(k, v)
	}

	if webhook.Secret != "" {
		signature := computeHMAC(body, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return &Result{Success: false, Error: err}, err
	}
	defer resp.Body.Close()

	return &Result{Success: resp.StatusCode >= 200 && resp.StatusCode < 300, StatusCode: resp.StatusCode}, nil
}

func computeHMAC(body []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func marshalPayload(payload interface{}) ([]byte, error) {
	switch v := payload.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

func NewWebhook(url, secret string, events ...string) *Webhook {
	return &Webhook{
		URL:    url,
		Secret: secret,
		Events: events,
		Headers: make(map[string]string),
		Retry: RetryPolicy{
			MaxAttempts: 3,
			Delay:       time.Second,
			Backoff:     ExponentialBackoff,
		},
	}
}

func (w *Webhook) AddHeader(key, value string) {
	w.Headers[key] = value
}
