package gmcore_router

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type testHandlerRegistry struct {
	handlers map[string]http.HandlerFunc
}

func newTestHandlerRegistry() *testHandlerRegistry {
	return &testHandlerRegistry{handlers: map[string]http.HandlerFunc{}}
}

func (r *testHandlerRegistry) Register(name string, handler http.HandlerFunc) {
	r.handlers[name] = handler
}

func (r *testHandlerRegistry) Get(name string) HandlerInfo {
	return HandlerInfo{Controller: name, Method: "Handle"}
}

func (r *testHandlerRegistry) CreateHandler(name string) http.HandlerFunc {
	if h, ok := r.handlers[name]; ok {
		return h
	}
	return func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "handler not found", http.StatusInternalServerError)
	}
}

func TestConfigApplyTo_UsesHandlerRegistryInterface(t *testing.T) {
	r := New()
	reg := newTestHandlerRegistry()

	var called int32
	reg.Register("home", func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	})

	cfg := &Config{Routes: []RouteConfig{{
		Name:    "home",
		Path:    "/",
		Handler: "home",
		Methods: []string{"GET"},
	}}}

	if err := cfg.ApplyTo(r, reg); err != nil {
		t.Fatalf("ApplyTo failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected handler to be called once, got %d", called)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestConfigApplyTo_MultipleMethods(t *testing.T) {
	r := New()
	reg := newTestHandlerRegistry()

	var called int32
	reg.Register("item", func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusNoContent)
	})

	cfg := &Config{Routes: []RouteConfig{{
		Name:    "item",
		Path:    "/items",
		Handler: "item",
		Methods: []string{"GET", "post", "GET"},
	}}}

	if err := cfg.ApplyTo(r, reg); err != nil {
		t.Fatalf("ApplyTo failed: %v", err)
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/items", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected status 204 for %s, got %d", method, w.Code)
		}
	}

	if atomic.LoadInt32(&called) != 2 {
		t.Fatalf("expected handler to be called twice, got %d", called)
	}
}

func TestConfigApplyTo_NilInputs(t *testing.T) {
	cfg := &Config{}
	if err := cfg.ApplyTo(nil, newTestHandlerRegistry()); err == nil {
		t.Fatal("expected error for nil router")
	}
	if err := cfg.ApplyTo(New(), nil); err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestNormalizeMethods(t *testing.T) {
	out := normalizeMethods([]string{" get ", "POST", "", "post", "GET"})
	if len(out) != 2 {
		t.Fatalf("expected 2 normalized methods, got %d", len(out))
	}
	if out[0] != http.MethodGet || out[1] != http.MethodPost {
		t.Fatalf("unexpected normalized methods: %v", out)
	}
}
