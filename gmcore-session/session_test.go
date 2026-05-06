package gmcore_session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryStore_New(t *testing.T) {
	store := NewMemoryStore()
	sess, err := store.New("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID() != "session-123" {
		t.Errorf("expected id session-123, got %s", sess.ID())
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	store.New("session-123")
	sess, err := store.Get("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()
	sess, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestMemoryStore_Save(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("key", "value")
	err := store.Save(sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	retrieved, _ := store.Get("session-123")
	if retrieved.Get("key") != "value" {
		t.Error("expected saved value")
	}
}

func TestMemoryStore_Save_Nil(t *testing.T) {
	store := NewMemoryStore()
	err := store.Save(nil)
	if err == nil {
		t.Error("expected error for nil session")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	store.New("session-123")
	err := store.Delete("session-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sess, _ := store.Get("session-123")
	if sess != nil {
		t.Error("expected nil after delete")
	}
}

func TestSession_GetSet(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	if sess.Get("name") != "alice" {
		t.Error("expected alice")
	}
}

func TestSession_Has(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	if !sess.Has("name") {
		t.Error("expected Has to return true")
	}
	if sess.Has("missing") {
		t.Error("expected Has to return false for missing")
	}
}

func TestSession_Remove(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	sess.Remove("name")
	if sess.Has("name") {
		t.Error("expected name to be removed")
	}
}

func TestSession_Keys(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	sess.Set("email", "alice@example.com")
	keys := sess.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestSession_Clear(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	sess.Clear()
	if sess.Has("name") {
		t.Error("expected name to be cleared")
	}
}

func TestSession_Flash(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Flash("hello")
	sess.Flash("world")
	flashes := sess.GetFlashes()
	if len(flashes) != 2 {
		t.Errorf("expected 2 flashes, got %d", len(flashes))
	}
	second := sess.GetFlashes()
	if len(second) != 0 {
		t.Error("flashes should be cleared after GetFlashes")
	}
}

func TestSession_Destroy(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	sess.Set("name", "alice")
	sess.Flash("message")
	sess.Destroy()
	if sess.Has("name") {
		t.Error("name should be cleared")
	}
}

func TestManager_Start_NewSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, "session_id", time.Hour)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	sess, err := manager.Start(w, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
	cookie := w.Result().Cookies()
	if len(cookie) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(cookie))
	}
}

func TestManager_Start_ExistingSession(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, "session_id", time.Hour)
	store.New("existing-session")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "session_id", Value: "existing-session"})
	sess, err := manager.Start(w, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.ID() != "existing-session" {
		t.Errorf("expected existing-session, got %s", sess.ID())
	}
}

func TestManager_Name(t *testing.T) {
	store := NewMemoryStore()
	manager := NewManager(store, "my_session", time.Hour)
	if manager.Name() != "my_session" {
		t.Errorf("expected my_session, got %s", manager.Name())
	}
}

func TestGenerateSid(t *testing.T) {
	sid1, err := generateSid()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sid1) != 64 {
		t.Errorf("expected 64-char hex string, got %d chars", len(sid1))
	}
	sid2, _ := generateSid()
	if sid1 == sid2 {
		t.Error("generated SIDs should be unique")
	}
}

func TestSaveToContext(t *testing.T) {
	store := NewMemoryStore()
	sess, _ := store.New("session-123")
	ctx := SaveToContext(context.Background(), sess)
	retrieved := FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected session from context")
	}
	if retrieved.ID() != "session-123" {
		t.Errorf("expected session-123, got %s", retrieved.ID())
	}
}

func TestFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	sess := FromContext(ctx)
	if sess != nil {
		t.Error("expected nil for empty context")
	}
}

func TestNewSession(t *testing.T) {
	s := newSession("test-id")
	if s.ID() != "test-id" {
		t.Errorf("expected test-id, got %s", s.ID())
	}
	if s.values == nil {
		t.Error("values should be initialized")
	}
	if s.flashes == nil {
		t.Error("flashes should be initialized")
	}
}
