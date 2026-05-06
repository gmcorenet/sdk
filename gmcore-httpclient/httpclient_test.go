package gmcore_httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.client == nil {
		t.Fatal("http.Client should be initialized")
	}
	if c.headers == nil {
		t.Fatal("headers map should be initialized")
	}
}

func TestClient_SetBaseURL(t *testing.T) {
	c := NewClient()
	result := c.SetBaseURL("https://api.example.com")
	if result != c {
		t.Fatal("SetBaseURL should return the same client")
	}
	if c.baseURL != "https://api.example.com" {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
}

func TestClient_SetTimeout(t *testing.T) {
	c := NewClient()
	c.SetTimeout(5 * time.Second)
	if c.timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %v", c.timeout)
	}
}

func TestClient_SetHeader(t *testing.T) {
	c := NewClient()
	c.SetHeader("Authorization", "Bearer token123")
	if c.headers["Authorization"] != "Bearer token123" {
		t.Fatalf("unexpected header: %s", c.headers["Authorization"])
	}
}

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		rawURL  string
		want    string
		wantErr bool
	}{
		{"absolute url", "", "https://example.com/api", "https://example.com/api", false},
		{"relative with base", "https://api.example.com", "/users", "https://api.example.com/users", false},
		{"empty url", "", "", "", true},
		{"no base no absolute", "", "/relative", "", true},
		{"absolute with base still absolute", "https://api.example.com", "https://other.com/api", "https://other.com/api", false},
		{"absolute without host", "", "http://", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient()
			if tt.baseURL != "" {
				c.SetBaseURL(tt.baseURL)
			}
			got, err := c.buildURL(tt.rawURL)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Get(context.Background(), server.URL+"/api/test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	if !resp.Ok() {
		t.Fatal("Ok should return true for 2xx")
	}
}

func TestClient_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Post(context.Background(), server.URL+"/api/users", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode())
	}
}

func TestClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Put(context.Background(), server.URL+"/api/users/1", map[string]string{"name": "updated"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if !resp.Ok() {
		t.Fatal("Ok should be true")
	}
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Delete(context.Background(), server.URL+"/api/users/1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if resp.StatusCode() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode())
	}
}

func TestClient_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Patch(context.Background(), server.URL+"/api/users/1", map[string]string{"name": "patched"})
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}
	if !resp.Ok() {
		t.Fatal("Ok should be true")
	}
}

func TestClient_GET_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Get(context.Background(), server.URL+"/missing")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if resp.Ok() {
		t.Fatal("Ok should be false for 404")
	}
	if resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode())
	}
}

func TestResponse_BodyString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	c := NewClient()
	resp, _ := c.Get(context.Background(), server.URL+"/echo")
	if resp.BodyString() != "hello world" {
		t.Fatalf("expected 'hello world', got %s", resp.BodyString())
	}
}

func TestResponse_Json(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"test","age":30}`))
	}))
	defer server.Close()

	c := NewClient()
	resp, _ := c.Get(context.Background(), server.URL+"/json")
	var data struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	err := resp.Json(&data)
	if err != nil {
		t.Fatalf("Json unmarshal failed: %v", err)
	}
	if data.Name != "test" || data.Age != 30 {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestResponse_Headers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient()
	resp, _ := c.Get(context.Background(), server.URL+"/headers")
	headers := resp.Headers()
	if headers["X-Custom-Header"] != "custom-value" {
		t.Fatalf("expected custom header, got %s", headers["X-Custom-Header"])
	}
}

func TestResponse_Body(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f})
	}))
	defer server.Close()

	c := NewClient()
	resp, _ := c.Get(context.Background(), server.URL+"/bytes")
	body := resp.Body()
	if len(body) != 5 {
		t.Fatalf("expected 5 bytes, got %d", len(body))
	}
}

func TestClient_InvalidURL(t *testing.T) {
	c := NewClient()
	_, err := c.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestClient_PostStringBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Post(context.Background(), server.URL+"/echo", "raw string body")
	if err != nil {
		t.Fatalf("Post with string body failed: %v", err)
	}
	if !resp.Ok() {
		t.Fatal("expected 200")
	}
}

func TestClient_PostByteBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient()
	resp, err := c.Post(context.Background(), server.URL+"/echo", []byte("byte body"))
	if err != nil {
		t.Fatalf("Post with byte body failed: %v", err)
	}
	if !resp.Ok() {
		t.Fatal("expected 200")
	}
}

func TestBuildURL_InvalidBase(t *testing.T) {
	c := NewClient()
	c.SetBaseURL("://invalid")
	_, err := c.buildURL("/test")
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestBuildURL_InvalidRelative(t *testing.T) {
	c := NewClient()
	c.SetBaseURL("https://example.com")
	_, err := c.buildURL("://invalid-relative")
	if err == nil {
		t.Fatal("expected error for invalid relative URL")
	}
}

func init() {
	_ = json.Marshal
	_ = fmt.Sprintf("")
	_ = strings.TrimSpace
}
