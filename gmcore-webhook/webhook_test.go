package gmcore_webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewWebhook(t *testing.T) {
	w := NewWebhook("https://example.com/webhook", "my-secret", "user.created", "user.deleted")
	if w == nil {
		t.Fatal("NewWebhook returned nil")
	}
	if w.URL != "https://example.com/webhook" {
		t.Fatalf("expected URL 'https://example.com/webhook', got %s", w.URL)
	}
	if w.Secret != "my-secret" {
		t.Fatalf("expected Secret 'my-secret', got %s", w.Secret)
	}
	if len(w.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(w.Events))
	}
	if w.Events[0] != "user.created" {
		t.Fatalf("expected 'user.created', got %s", w.Events[0])
	}
	if w.Events[1] != "user.deleted" {
		t.Fatalf("expected 'user.deleted', got %s", w.Events[1])
	}
	if w.Headers == nil {
		t.Fatal("Headers should be initialized")
	}
	if w.Retry.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts 3, got %d", w.Retry.MaxAttempts)
	}
}

func TestNewWebhook_NoEvents(t *testing.T) {
	w := NewWebhook("https://example.com/webhook", "secret")
	if len(w.Events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(w.Events))
	}
}

func TestWebhook_AddHeader(t *testing.T) {
	w := NewWebhook("https://example.com/webhook", "secret")
	w.AddHeader("X-Custom-Header", "custom-value")
	w.AddHeader("Authorization", "Bearer token")

	if w.Headers["X-Custom-Header"] != "custom-value" {
		t.Fatalf("unexpected header value: %s", w.Headers["X-Custom-Header"])
	}
	if w.Headers["Authorization"] != "Bearer token" {
		t.Fatalf("unexpected header value: %s", w.Headers["Authorization"])
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.webhooks == nil {
		t.Fatal("webhooks map should be initialized")
	}
	if m.client == nil {
		t.Fatal("http client should be initialized")
	}
}

func TestManager_Register(t *testing.T) {
	m := NewManager()
	w := NewWebhook("https://example.com/webhook", "secret")
	err := m.Register(w)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if len(m.webhooks) != 1 {
		t.Fatalf("expected 1 webhook, got %d", len(m.webhooks))
	}
}

func TestManager_Unregister(t *testing.T) {
	m := NewManager()
	w := NewWebhook("https://example.com/webhook", "secret")
	m.Register(w)

	err := m.Unregister("https://example.com/webhook")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}
	if len(m.webhooks) != 0 {
		t.Fatalf("expected 0 webhooks, got %d", len(m.webhooks))
	}

	err = m.Unregister("nonexistent")
	if err != nil {
		t.Fatalf("Unregister of nonexistent should not error: %v", err)
	}
}

func TestManager_Send_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if sig := r.Header.Get("X-Webhook-Signature"); sig == "" {
			t.Fatal("expected X-Webhook-Signature header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager()
	w := NewWebhook(server.URL, "secret")
	w.Retry.MaxAttempts = 1

	result, err := m.Send(w, "test payload")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", result.StatusCode)
	}
	if result.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestManager_Send_NoSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Webhook-Signature")
		if sig != "" {
			t.Fatal("X-Webhook-Signature should not be set without secret")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager()
	w := NewWebhook(server.URL, "")
	w.Retry.MaxAttempts = 1

	result, err := m.Send(w, "payload")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestManager_Send_SuccessAfterOneAttempt(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager()
	w := NewWebhook(server.URL, "secret")
	w.Retry.MaxAttempts = 3
	w.Retry.Backoff = LinearBackoff

	result, err := m.Send(w, "payload")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", result.Attempts)
	}
}

func TestManager_Send_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Fatalf("expected X-Custom header, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m := NewManager()
	w := NewWebhook(server.URL, "secret")
	w.AddHeader("X-Custom", "custom-value")
	w.Retry.MaxAttempts = 1

	result, err := m.Send(w, "payload")
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
}

func TestManager_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := NewManager()
	w := NewWebhook(server.URL, "secret")
	w.Retry.MaxAttempts = 1

	result, _ := m.Send(w, "payload")
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", result.StatusCode)
	}
}

func TestManager_Send_UnreachableServer(t *testing.T) {
	m := NewManager()
	w := NewWebhook("http://localhost:1", "secret")
	w.Retry.MaxAttempts = 1

	result, err := m.Send(w, "payload")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if result.Success {
		t.Fatal("expected failure")
	}
}

func TestManager_Send_InvalidURL(t *testing.T) {
	m := NewManager()
	w := NewWebhook("://invalid-url", "secret")
	w.Retry.MaxAttempts = 1

	_, err := m.Send(w, "payload")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestLinearBackoff(t *testing.T) {
	if LinearBackoff(1) != time.Second {
		t.Fatalf("LinearBackoff(1) should be 1s, got %v", LinearBackoff(1))
	}
	if LinearBackoff(3) != 3*time.Second {
		t.Fatalf("LinearBackoff(3) should be 3s, got %v", LinearBackoff(3))
	}
}

func TestExponentialBackoff(t *testing.T) {
	if ExponentialBackoff(0) != time.Second {
		t.Fatalf("ExponentialBackoff(0) should be 1s, got %v", ExponentialBackoff(0))
	}
	if ExponentialBackoff(1) != 2*time.Second {
		t.Fatalf("ExponentialBackoff(1) should be 2s, got %v", ExponentialBackoff(1))
	}
	if ExponentialBackoff(2) != 4*time.Second {
		t.Fatalf("ExponentialBackoff(2) should be 4s, got %v", ExponentialBackoff(2))
	}
}

func TestComputeHMAC(t *testing.T) {
	result := computeHMAC([]byte("hello"), "secret")
	if result == "" {
		t.Fatal("HMAC should not be empty")
	}
	if len(result) != 64 {
		t.Fatalf("expected 64 hex chars (sha256), got %d", len(result))
	}

	result2 := computeHMAC([]byte("hello"), "secret")
	if result != result2 {
		t.Fatal("HMAC should be deterministic")
	}

	result3 := computeHMAC([]byte("hello"), "different")
	if result == result3 {
		t.Fatal("HMAC should differ with different secret")
	}
}

func TestMarshalPayload(t *testing.T) {
	b, err := marshalPayload([]byte("bytes"))
	if err != nil {
		t.Fatalf("marshalPayload failed: %v", err)
	}
	if string(b) != "bytes" {
		t.Fatalf("unexpected bytes: %s", string(b))
	}

	b, err = marshalPayload("string")
	if err != nil {
		t.Fatalf("marshalPayload failed: %v", err)
	}
	if string(b) != "string" {
		t.Fatalf("unexpected string: %s", string(b))
	}

	b, err = marshalPayload(42)
	if err != nil {
		t.Fatalf("marshalPayload failed: %v", err)
	}
	if string(b) != "42" {
		t.Fatalf("unexpected: %s", string(b))
	}

	b, err = marshalPayload(map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("marshalPayload failed: %v", err)
	}
	if string(b) != "map[a:b]" {
		t.Fatalf("unexpected: %s", string(b))
	}
}

func TestManager_Send_DefaultsMaxAttempts(t *testing.T) {
	m := NewManager()
	w := NewWebhook("https://example.com/webhook", "secret")
	w.Retry.MaxAttempts = 0

	if w.Retry.MaxAttempts <= 0 {
		_, _ = m.Send(w, "payload")
		if w.Retry.MaxAttempts != 1 {
			t.Fatalf("MaxAttempts should default to 1, got %d", w.Retry.MaxAttempts)
		}
	}
}

func TestManager_Send_DefaultsBackoff(t *testing.T) {
	m := NewManager()
	w := NewWebhook("https://example.com/webhook", "secret")
	w.Retry.Backoff = nil

	if w.Retry.Backoff == nil {
		_, _ = m.Send(w, "payload")
		if w.Retry.Backoff == nil {
			t.Fatal("Backoff should be set to default")
		}
	}
}

func TestResult(t *testing.T) {
	r := Result{
		Success:    true,
		StatusCode: 200,
		Response:   []byte("OK"),
		Error:      nil,
		Attempts:   1,
	}
	if !r.Success {
		t.Fatal("expected true")
	}
	if r.StatusCode != 200 {
		t.Fatal("expected 200")
	}
	if r.Attempts != 1 {
		t.Fatal("expected 1")
	}
}
