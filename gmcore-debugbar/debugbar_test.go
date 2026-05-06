package gmcore_debugbar

import (
	"fmt"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	db := New()
	if db == nil {
		t.Fatal("New returned nil")
	}
	if !db.enabled {
		t.Fatal("debugbar should be enabled by default")
	}
	if len(db.panels) != 0 {
		t.Fatalf("expected 0 panels, got %d", len(db.panels))
	}
}

func TestDebugBar_EnableDisable(t *testing.T) {
	db := New()
	db.Disable()
	if db.IsEnabled() {
		t.Fatal("should be disabled")
	}
	db.Enable()
	if !db.IsEnabled() {
		t.Fatal("should be enabled")
	}
}

func TestDebugBar_Render_Disabled(t *testing.T) {
	db := New()
	db.Disable()
	html := db.Render()
	if html != "" {
		t.Fatal("Render should return empty string when disabled")
	}
}

func TestDebugBar_Render_Enabled(t *testing.T) {
	db := New()
	html := db.Render()
	if html == "" {
		t.Fatal("Render should return HTML when enabled")
	}
	if !contains(html, "debug-bar") {
		t.Fatal("HTML should contain debug-bar id")
	}
}

func TestDebugBar_AddPanel(t *testing.T) {
	db := New()
	p := NewRequestsPanel()
	db.AddPanel(p)
	if len(db.panels) != 1 {
		t.Fatalf("expected 1 panel, got %d", len(db.panels))
	}
}

func TestRequestsPanel(t *testing.T) {
	p := NewRequestsPanel()
	if p == nil {
		t.Fatal("NewRequestsPanel returned nil")
	}
	if p.GetName() != "requests" {
		t.Fatalf("expected name 'requests', got %s", p.GetName())
	}
	html := p.GetHTML()
	if html == "" {
		t.Fatal("GetHTML should return a string")
	}
	if !contains(html, "Requests:") {
		t.Fatal("HTML should contain 'Requests:'")
	}
}

func TestTimelinePanel(t *testing.T) {
	p := &TimelinePanel{events: []TimelineEvent{
		{Name: "event1"},
		{Name: "event2"},
	}}
	if p.GetName() != "timeline" {
		t.Fatalf("expected name 'timeline', got %s", p.GetName())
	}
	html := p.GetHTML()
	if !contains(html, "Events: 2") {
		t.Fatalf("expected 'Events: 2', got %s", html)
	}
}

func TestMemoryPanel(t *testing.T) {
	p := NewMemoryPanel()
	if p.GetName() != "memory" {
		t.Fatalf("expected name 'memory', got %s", p.GetName())
	}
	html := p.GetHTML()
	if !contains(html, "Memory:") {
		t.Fatalf("expected 'Memory:', got %s", html)
	}
}

func TestDatabasePanel(t *testing.T) {
	p := NewDatabasePanel()
	if p.GetName() != "database" {
		t.Fatalf("expected name 'database', got %s", p.GetName())
	}
	p.AddQuery("SELECT * FROM users")
	p.AddQuery("SELECT * FROM posts")
	html := p.GetHTML()
	if !contains(html, "Queries: 2") {
		t.Fatalf("expected 'Queries: 2', got %s", html)
	}
}

func TestCachePanel(t *testing.T) {
	p := NewCachePanel()
	html := p.GetHTML()
	if !contains(html, "Cache: N/A") {
		t.Fatalf("expected 'Cache: N/A', got %s", html)
	}

	p.hits = 75
	p.misses = 25
	html = p.GetHTML()
		if !contains(html, "Cache: 75%") {
		t.Fatalf("expected 'Cache: 75%%', got %s", html)
	}
}

func TestLogsPanel(t *testing.T) {
	p := NewLogsPanel()
	if p.GetName() != "logs" {
		t.Fatalf("expected name 'logs', got %s", p.GetName())
	}
	p.AddLog("info: server started")
	p.AddLog("error: db connection failed")
	html := p.GetHTML()
	if !contains(html, "Logs: 2") {
		t.Fatalf("expected 'Logs: 2', got %s", html)
	}
}

func TestDebugBar_Render_WithPanels(t *testing.T) {
	db := New()
	db.AddPanel(NewRequestsPanel())
	db.AddPanel(NewMemoryPanel())
	db.AddPanel(NewCachePanel())

	html := db.Render()
	if !contains(html, "Requests:") {
		t.Fatal("HTML should include requests panel")
	}
	if !contains(html, "Memory:") {
		t.Fatal("HTML should include memory panel")
	}
	if !contains(html, "Cache:") {
		t.Fatal("HTML should include cache panel")
	}
}

func TestDebugBar_Middleware_Disabled(t *testing.T) {
	db := New()
	db.Disable()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("OK"))
	})

	handler := db.Middleware(next)

	mockWriter := &mockResponseWriter{header: make(http.Header)}
	req, _ := http.NewRequest("GET", "/", nil)
	handler.ServeHTTP(mockWriter, req)

	if !called {
		t.Fatal("next handler should be called")
	}
}

func TestDebugBar_Middleware_Enabled(t *testing.T) {
	db := New()
	db.AddPanel(NewRequestsPanel())

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("<html><head></head><body></body></html>"))
	})

	handler := db.Middleware(next)

	mockWriter := &mockResponseWriter{header: make(http.Header)}
	req, _ := http.NewRequest("GET", "/", nil)
	handler.ServeHTTP(mockWriter, req)

	if !called {
		t.Fatal("next handler should be called")
	}
	if !mockWriter.written {
		t.Fatal("response should be written")
	}
}

func TestResponseWriter_Push(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: &mockResponseWriter{header: make(http.Header)},
		statusCode:     http.StatusOK,
	}

	err := rw.Push("/static/app.js", nil)
	if err == nil {
		t.Fatal("Push on non-pusher should return error")
	}
}

type mockResponseWriter struct {
	header     http.Header
	body       []byte
	statusCode int
	written    bool
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body = append(m.body, b...)
	m.written = true
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
	m.written = true
}

func BenchmarkDebugBar_Render(b *testing.B) {
	db := New()
	db.AddPanel(NewRequestsPanel())
	db.AddPanel(NewMemoryPanel())
	db.AddPanel(NewDatabasePanel())
	db.AddPanel(NewCachePanel())
	db.AddPanel(NewLogsPanel())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = db.Render()
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDebugBar_RenderContainsClosingDiv(t *testing.T) {
	db := New()
	html := db.Render()
	if !contains(html, "</div>") {
		t.Fatal("HTML should contain closing div tag")
	}
}

func TestDebugBar_RenderContainsStyle(t *testing.T) {
	db := New()
	html := db.Render()
	if !contains(html, "debug-bar") {
		t.Fatal("HTML should contain debug-bar element")
	}
}

func TestDebugBar_MultiplePanelsAreRendered(t *testing.T) {
	db := New()
	for i := 0; i < 5; i++ {
		db.AddPanel(NewRequestsPanel())
	}
	html := db.Render()
	count := 0
	idx := 0
	for {
		i := indexOf(html, "Requests:", idx)
		if i < 0 {
			break
		}
		count++
		idx = i + 1
	}
	if count != 5 {
		t.Fatalf("expected 5 occurrences of 'Requests:', got %d", count)
	}
}

func indexOf(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	_ = fmt.Sprintf("") // ensure fmt import is used
}
