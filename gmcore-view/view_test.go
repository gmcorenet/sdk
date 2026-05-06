package gmcore_view

import (
	"context"
	"html/template"
	"testing"

	gmcore_i18n "github.com/gmcorenet/sdk/gmcore-i18n"
)

func TestNewFuncRegistry(t *testing.T) {
	r := NewFuncRegistry()
	if r == nil {
		t.Fatal("NewFuncRegistry returned nil")
	}
	if r.funcs == nil {
		t.Fatal("funcs map should be initialized")
	}
}

func TestFuncRegistry_Register(t *testing.T) {
	r := NewFuncRegistry()
	r.Register("upper", func(s string) string { return s })

	if _, ok := r.funcs["upper"]; !ok {
		t.Fatal("function should be registered")
	}
}

func TestFuncRegistry_MergeInto(t *testing.T) {
	r := NewFuncRegistry()
	r.Register("custom", func() string { return "custom" })

	base := make(template.FuncMap)
	base["existing"] = func() string { return "existing" }

	merged := r.MergeInto(base)

	if _, ok := merged["existing"]; !ok {
		t.Fatal("existing function should be preserved")
	}
	if _, ok := merged["custom"]; !ok {
		t.Fatal("custom function should be added")
	}
}

func TestNew(t *testing.T) {
	cfg := Config{
		AppRoot:    "/app",
		SystemRoot: "/system",
	}
	r := New(cfg)
	if r == nil {
		t.Fatal("New returned nil")
	}
	if r.funcsReg == nil {
		t.Fatal("funcsReg should be initialized")
	}
}

func TestRenderer_Funcs(t *testing.T) {
	cfg := Config{
		AppRoot:    "/app",
		SystemRoot: "/system",
	}
	r := New(cfg)
	fr := r.Funcs()
	if fr == nil {
		t.Fatal("Funcs should return non-nil")
	}
}

func TestRender_EmptyTemplate(t *testing.T) {
	cfg := Config{
		AppRoot:    "",
		SystemRoot: "",
	}
	r := New(cfg)

	_, err := r.Render("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestResolveTemplate(t *testing.T) {
	cfg := Config{
		AppRoot:    "",
		SystemRoot: "",
	}
	_, ok := ResolveTemplate(cfg, "nonexistent")
	if ok {
		t.Fatal("expected false for nonexistent template")
	}
}

func TestContextWithoutCancel(t *testing.T) {
	ctx := contextWithoutCancel(nil)
	if ctx == nil {
		t.Fatal("should return background context for nil")
	}

	parent := context.Background()
	ctx = contextWithoutCancel(parent)
	if ctx == nil {
		t.Fatal("should return non-nil context")
	}
}

func TestMergeTemplatePayload(t *testing.T) {
	tests := []struct {
		name  string
		base  map[string]interface{}
		extra []interface{}
		want  int
	}{
		{"nil extra", map[string]interface{}{"a": 1}, nil, 1},
		{"empty extra", map[string]interface{}{"a": 1}, []interface{}{}, 1},
		{"string map extra", map[string]interface{}{"a": 1}, []interface{}{map[string]string{"b": "2"}}, 2},
		{"interface map extra", map[string]interface{}{"a": 1}, []interface{}{map[string]interface{}{"b": 2}}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeTemplatePayload(tt.base, tt.extra...)
			if len(merged) != tt.want {
				t.Fatalf("expected %d keys, got %d: %v", tt.want, len(merged), merged)
			}
		})
	}
}

func TestMergeI18nParams(t *testing.T) {
	tests := []struct {
		name  string
		extra []interface{}
		want  int
	}{
		{"nil", nil, 0},
		{"empty", []interface{}{}, 0},
		{"native params", []interface{}{gmcore_i18n.Params{"key": "value"}}, 1},
		{"map string interface", []interface{}{map[string]interface{}{"key": "value"}}, 1},
		{"map string string", []interface{}{map[string]string{"key": "value"}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := mergeI18nParams(tt.extra...)
			if len(params) != tt.want {
				t.Fatalf("expected %d params, got %d", tt.want, len(params))
			}
		})
	}
}

func TestInterpolateTranslatedValue(t *testing.T) {
	result := interpolateTranslatedValue("Hello {name}", gmcore_i18n.Params{"name": "World"})
	if result != "Hello World" {
		t.Fatalf("expected 'Hello World', got %s", result)
	}

	result = interpolateTranslatedValue("No params", gmcore_i18n.Params{})
	if result != "No params" {
		t.Fatalf("expected 'No params', got %s", result)
	}
}

func TestSafeJSGlobalTarget(t *testing.T) {
	tests := []struct {
		name     string
		fallback string
		values   []string
		want     string
	}{
		{"no values", "window.default", nil, "window.default"},
		{"empty value", "window.default", []string{""}, "window.default"},
		{"valid", "window.default", []string{"window.myGlobal"}, "window.myGlobal"},
		{"invalid chars", "window.default", []string{"alert('xss')"}, "window.default"},
		{"valid dotted", "window.default", []string{"window.MyApp.global"}, "window.MyApp.global"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeJSGlobalTarget(tt.fallback, tt.values...)
			if result != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, result)
			}
		})
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := Config{
		AppRoot:    "/app",
		SystemRoot: "/system",
		BundleRoots: []string{"/bundle1", "/bundle2"},
	}
	if cfg.AppRoot != "/app" {
		t.Fatal("unexpected AppRoot")
	}
	if len(cfg.BundleRoots) != 2 {
		t.Fatalf("expected 2 bundle roots, got %d", len(cfg.BundleRoots))
	}
}

func TestViewResolvedFile(t *testing.T) {
	f := viewResolvedFile{
		Path:   "/path/to/template.html",
		Source: "template content",
	}
	if f.Path != "/path/to/template.html" {
		t.Fatal("unexpected path")
	}
	if f.Source != "template content" {
		t.Fatal("unexpected source")
	}
}

func TestFuncRegistry_ConcurrentAccess(t *testing.T) {
	r := NewFuncRegistry()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			r.Register("func"+string(rune('0'+id)), func() string { return "ok" })
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	base := template.FuncMap{
		"existing": func() string { return "existing" },
	}
	merged := r.MergeInto(base)
	if _, ok := merged["existing"]; !ok {
		t.Fatal("existing function should still be there")
	}
}

func TestWithAssetRegistry(t *testing.T) {
	ctx := context.Background()
	ctx = withAssetRegistry(ctx)

	registry := registryFromContext(ctx)
	if registry == nil {
		t.Fatal("registry should be created")
	}

	ctx2 := withAssetRegistry(ctx)
	registry2 := registryFromContext(ctx2)
	if registry != registry2 {
		t.Fatal("should reuse existing registry")
	}
}

func TestRegistryFromContext_Nil(t *testing.T) {
	r := registryFromContext(nil)
	if r != nil {
		t.Fatal("should return nil for nil context")
	}

	r = registryFromContext(context.Background())
	if r != nil {
		t.Fatal("should return nil for context without registry")
	}
}

func TestRegisterHeadAsset(t *testing.T) {
	ctx := withAssetRegistry(context.Background())

	RegisterHeadAsset(ctx, "css-reset", template.HTML("<link rel='stylesheet'>"))
	html := string(renderHeadAssets(ctx))
	if html == "" {
		t.Fatal("head assets should be rendered")
	}

	RegisterHeadAsset(ctx, "css-reset", template.HTML("<link rel='stylesheet'>"))
	html2 := string(renderHeadAssets(ctx))
	if html != html2 {
		t.Fatal("duplicate key should not add again")
	}
}

func TestRegisterHeadAsset_NoRegistry(t *testing.T) {
	ctx := context.Background()
	RegisterHeadAsset(ctx, "key", template.HTML("html"))

	html := renderHeadAssets(ctx)
	if html != template.HTML("") {
		t.Fatal("should return empty without registry")
	}
}

func TestRenderHeadAssets_Empty(t *testing.T) {
	ctx := withAssetRegistry(context.Background())
	html := renderHeadAssets(ctx)
	if html != template.HTML("") {
		t.Fatal("should return empty for empty registry")
	}
}

func TestHeadAssetsPlaceholder(t *testing.T) {
	if headAssetsPlaceholder != "<!-- GMCORE_HEAD_ASSETS -->" {
		t.Fatal("unexpected placeholder value")
	}
}
