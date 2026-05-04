package gmcore_router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter_Add(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})

	if len(r.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(r.routes))
	}
	if r.routes[0].Method != "GET" {
		t.Errorf("expected method GET, got %s", r.routes[0].Method)
	}
	if r.routes[0].Path != "/test" {
		t.Errorf("expected path /test, got %s", r.routes[0].Path)
	}
	if r.routes[0].Name != "test" {
		t.Errorf("expected name test, got %s", r.routes[0].Name)
	}
}

func TestRouter_Add_TrimsMethod(t *testing.T) {
	r := New()
	r.Add("  get  ", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})

	if r.routes[0].Method != "GET" {
		t.Errorf("expected method GET, got %s", r.routes[0].Method)
	}
}

func TestRouter_Group(t *testing.T) {
	r := New()
	g := r.Group("/api/v1", "api_v1_")
	g.Add("GET", "/users", "users", func(w http.ResponseWriter, r *http.Request) {})

	if len(r.routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(r.routes))
	}
	if r.routes[0].Path != "/api/v1/users" {
		t.Errorf("expected path /api/v1/users, got %s", r.routes[0].Path)
	}
	if r.routes[0].Name != "api_v1_users" {
		t.Errorf("expected name api_v1_users, got %s", r.routes[0].Name)
	}
}

func TestRouter_Group_Nested(t *testing.T) {
	r := New()
	g1 := r.Group("/api", "api_")
	g2 := r.Group("/v1", "v1_") // Groups don't nest, but prefixes accumulate
	g1.Add("GET", "/users", "list", func(w http.ResponseWriter, r *http.Request) {})
	g2.Add("GET", "/users", "v1_list", func(w http.ResponseWriter, r *http.Request) {})

	if r.routes[0].Path != "/api/users" {
		t.Errorf("expected path /api/users, got %s", r.routes[0].Path)
	}
	if r.routes[0].Name != "api_list" {
		t.Errorf("expected name api_list, got %s", r.routes[0].Name)
	}
	if r.routes[1].Path != "/v1/users" {
		t.Errorf("expected path /v1/users, got %s", r.routes[1].Path)
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	var called bool
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRouter_ServeHTTP_NotFound(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestRouter_ServeHTTP_MethodNotAllowed(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestRouter_ServeHTTP_HeadToGet(t *testing.T) {
	var called bool
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("HEAD", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("HEAD should fall back to GET")
	}
}

func TestRouter_ServeHTTP_WithParams(t *testing.T) {
	var capturedParam string
	r := New()
	r.Add("GET", "/users/{id}", "user", func(w http.ResponseWriter, r *http.Request) {
		capturedParam = Param(r, "id")
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedParam != "123" {
		t.Errorf("expected param 123, got %s", capturedParam)
	}
}

func TestRouter_ServeHTTP_CustomNotFound(t *testing.T) {
	var called bool
	r := New()
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("custom not found was not called")
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", w.Code)
	}
}

func TestRouter_URL(t *testing.T) {
	r := New()
	r.Add("GET", "/users/{id}", "user_show", nil)
	r.Add("GET", "/users/{id}/posts/{post}", "user_post", nil)

	tests := []struct {
		name     string
		route    string
		params   map[string]string
		expected string
	}{
		{"simple param", "user_show", map[string]string{"id": "123"}, "/users/123"},
		{"multiple params", "user_post", map[string]string{"id": "1", "post": "5"}, "/users/1/posts/5"},
		{"url encoded", "user_show", map[string]string{"id": "a b"}, "/users/a%20b"},
		{"not found", "nonexistent", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.URL(tt.route, tt.params)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRouter_Routes(t *testing.T) {
	r := New()
	r.Add("GET", "/one", "one", nil)
	r.Add("POST", "/two", "two", nil)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

func TestRouter_NamedRoutes(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "test", nil)
	r.Add("POST", "/other", "", nil) // unnamed

	named := r.NamedRoutes()
	if len(named) != 1 {
		t.Errorf("expected 1 named route, got %d", len(named))
	}
	if _, ok := named["test"]; !ok {
		t.Error("expected test route to be named")
	}
}

func TestRouter_NamedRoutesSorted(t *testing.T) {
	r := New()
	r.Add("GET", "/z", "z", nil)
	r.Add("GET", "/a", "a", nil)
	r.Add("GET", "/m", "m", nil)

	sorted := r.NamedRoutesSorted()
	if len(sorted) != 3 {
		t.Errorf("expected 3 routes, got %d", len(sorted))
	}
	if sorted[0].Name != "a" {
		t.Errorf("expected first route to be 'a', got %s", sorted[0].Name)
	}
	if sorted[1].Name != "m" {
		t.Errorf("expected second route to be 'm', got %s", sorted[1].Name)
	}
	if sorted[2].Name != "z" {
		t.Errorf("expected third route to be 'z', got %s", sorted[2].Name)
	}
}

func TestParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := req.Context()
	params := map[string]string{"id": "42"}
	ctx = context.WithValue(ctx, paramsKey, params)
	req = req.WithContext(ctx)

	if Param(req, "id") != "42" {
		t.Error("expected param id=42")
	}
	if Param(req, "missing") != "" {
		t.Error("expected empty string for missing param")
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		params  map[string]string
		matched bool
	}{
		{"/", "/", nil, true},
		{"/users", "/users", nil, true},
		{"/users", "/posts", nil, false},
		{"/users/{id}", "/users/123", map[string]string{"id": "123"}, true},
		{"/users/{id}/posts/{post}", "/users/1/posts/5", map[string]string{"id": "1", "post": "5"}, true},
		{"/users/{id}", "/users/", nil, false},
		{"/users/{id}", "/users", nil, false},
		{"/prefix/{id}/suffix", "/prefix/123/suffix", map[string]string{"id": "123"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			params, matched := matchPath(tt.pattern, tt.path)
			if matched != tt.matched {
				t.Errorf("expected matched=%v, got %v", tt.matched, matched)
			}
			if tt.matched {
				for k, v := range tt.params {
					if params[k] != v {
						t.Errorf("expected param %s=%s, got %s", k, v, params[k])
					}
				}
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"/", nil},
		{"/users", []string{"users"}},
		{"/users/", []string{"users"}},
		{"users", []string{"users"}},
		{"/users/posts", []string{"users", "posts"}},
		{"/users/posts/", []string{"users", "posts"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := split(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"/users", "/users"},
		{"users", "/users"},
		{"/users/", "/users"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePrefix(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		prefix   string
		path     string
		expected string
	}{
		{"", "", "/"},
		{"", "/", "/"},
		{"/api", "users", "/api/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api/v1", "users/{id}", "/api/v1/users/{id}"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_"+tt.path, func(t *testing.T) {
			result := joinPath(tt.prefix, tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGroup_EmptyPrefix(t *testing.T) {
	r := New()
	g := r.Group("", "")
	g.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})

	if r.routes[0].Path != "/test" {
		t.Errorf("expected /test, got %s", r.routes[0].Path)
	}
}

func TestRouter_Routes_ReturnsCopy(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "test", nil)

	routes1 := r.Routes()
	routes2 := r.Routes()

	routes1[0].Path = "/modified"

	if r.routes[0].Path == "/modified" {
		t.Error("Routes() should return a copy, not the original")
	}
	if routes2[0].Path == "/modified" {
		t.Error("subsequent calls should also return copies")
	}
}

func TestContext_IsNotLost(t *testing.T) {
	var capturedParams map[string]string
	r := New()
	r.Add("GET", "/users/{id}", "user", func(w http.ResponseWriter, r *http.Request) {
		capturedParams = r.Context().Value(paramsKey).(map[string]string)
	})

	req := httptest.NewRequest("GET", "/users/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedParams["id"] != "abc" {
		t.Errorf("expected id=abc, got %v", capturedParams)
	}
}

func TestPathWithQueryAndFragment(t *testing.T) {
	var called bool
	r := New()
	r.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest("GET", "/test?foo=bar#section", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !called {
		t.Error("route with query string should match")
	}
}

func TestEmptyRouteName(t *testing.T) {
	r := New()
	r.Add("GET", "/test", "", func(w http.ResponseWriter, r *http.Request) {})

	named := r.NamedRoutes()
	if len(named) != 0 {
		t.Error("empty route name should not appear in named routes")
	}
}

func TestGroup_TrimsNamePrefix(t *testing.T) {
	r := New()
	g := r.Group("/api", "  api_")
	g.Add("GET", "/test", "test", func(w http.ResponseWriter, r *http.Request) {})

	if !strings.HasPrefix(r.routes[0].Name, "api_") {
		t.Errorf("name should be prefixed, got %s", r.routes[0].Name)
	}
}

func TestRouter_PathParamsWithSpecialChars(t *testing.T) {
	var id string
	r := New()
	r.Add("GET", "/users/{id}", "user", func(w http.ResponseWriter, r *http.Request) {
		id = Param(r, "id")
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/123", "123"},
		{"/users/abc", "abc"},
		{"/users/abc-def", "abc-def"},
		{"/users/abc_def", "abc_def"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			id = ""
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if id != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, id)
			}
		})
	}
}
