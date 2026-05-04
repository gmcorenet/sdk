package gmcore_templating

import (
	"context"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTwigInheritanceAndIncludeWithPayload(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "partials"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "base.html"), `<html><body>{% block content %}{% endblock %}</body></html>`)
	writeTemplate(t, filepath.Join(templatesDir, "partials", "card.html"), `<section>{{.Title}}</section>`)
	writeTemplate(t, filepath.Join(templatesDir, "page.html"), `{% extends "base.html" %}{% block content %}{% include "partials/card.html" with (dict "Title" "Status") %}<p>{{trans_choice "messages.items" 2}}</p>{% endblock %}`)

	engine := New(Config{
		AppRoot: appRoot,
		Funcs: template.FuncMap{
			"trans_choice": func(key string, count int) string { return "2 items" },
		},
	})
	rendered, err := engine.RenderContext(context.Background(), "page.html", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "<section>Status</section>") {
		t.Fatalf("expected include output, got %q", rendered)
	}
	if !strings.Contains(rendered, "<p>2 items</p>") {
		t.Fatalf("expected translation output, got %q", rendered)
	}
}

func TestBaseFuncsHelpers(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "helpers.html"), `{{json (dict "a" 1)}}|{{first (list "x" "y")}}|{{last (list "x" "y")}}|{{coalesce "" nil "ok"}}|{{date "2026-04-06" "2006"}}`)
	engine := New(Config{AppRoot: appRoot})
	rendered, err := engine.RenderContext(context.Background(), "helpers.html", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `&#34;a&#34;:1`) || !strings.Contains(rendered, `|x|y|ok|2026`) {
		t.Fatalf("unexpected helpers output: %q", rendered)
	}
}

func TestTwigMacrosRenderReusableMarkup(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "page.html"), `
{% macro badge(label, tone) %}
<span class="badge badge-{{ tone }}">{{ label|upper }}</span>
{% endmacro %}
<div>{{ badge("draft", "warning") }}</div>
`)

	engine := New(Config{AppRoot: appRoot})
	rendered, err := engine.RenderContext(context.Background(), "page.html", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `<span class="badge badge-warning">DRAFT</span>`) {
		t.Fatalf("expected macro output, got %q", rendered)
	}
}

func TestTwigControlFlowAndSet(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "flow.html"), `{% set title = providedTitle %}{% if title %}<h1>{{ title|upper }}</h1>{% endif %}<ul>{% for item in items %}<li>{{ name }}</li>{% endfor %}</ul>`)
	engine := New(Config{AppRoot: appRoot})
	rendered, err := engine.RenderContext(context.Background(), "flow.html", map[string]interface{}{
		"providedTitle": "Control",
		"items": []map[string]interface{}{
			{"name": "one"},
			{"name": "two"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "<h1>CONTROL</h1>") || !strings.Contains(rendered, "<li>one</li>") || !strings.Contains(rendered, "<li>two</li>") {
		t.Fatalf("unexpected control flow output: %q", rendered)
	}
}

func TestTwigImportMacrosAndWithBlock(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "macros"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "macros", "ui.html"), `
{% macro badge(label, tone) %}
<span class="badge badge-{{ tone }}">{{ label }}</span>
{% endmacro %}
`)
	writeTemplate(t, filepath.Join(templatesDir, "page.html"), `
{% import "macros/ui.html" as ui %}
{% with (dict "label" "Ready" "tone" "success") %}
<div>{{ ui.badge(label, tone) }}</div>
{% endwith %}
`)
	engine := New(Config{AppRoot: appRoot, Mode: ModeProd})
	rendered, err := engine.RenderContext(context.Background(), "page.html", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `<span class="badge badge-success">Ready</span>`) {
		t.Fatalf("unexpected imported macro output: %q", rendered)
	}
}

func TestTwigEmbedApplySpacelessAndFromImport(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "macros"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "card.html"), `<article>{% block body %}<p>base</p>{% endblock %}</article>`)
	writeTemplate(t, filepath.Join(templatesDir, "macros", "text.html"), `{% macro shout(value) %}{{ value|upper }}{% endmacro %}`)
	writeTemplate(t, filepath.Join(templatesDir, "page.html"), `
{% from "macros/text.html" import shout as shout %}
{% apply trim|upper %}
  hello world
{% endapply %}
{% spaceless %}
  <div>
    <span>{{ shout("ok") }}</span>
  </div>
{% endspaceless %}
{% embed "card.html" with (dict "message" "embedded") %}
  {% block body %}
    <strong>{{ message }}</strong>
  {% endblock %}
{% endembed %}
`)
	engine := New(Config{AppRoot: appRoot, Mode: ModeProd})
	rendered, err := engine.RenderContext(context.Background(), "page.html", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "HELLO WORLD") {
		t.Fatalf("expected apply output, got %q", rendered)
	}
	if !strings.Contains(rendered, `<div> <span>OK</span> </div>`) {
		t.Fatalf("expected spaceless output, got %q", rendered)
	}
	if !strings.Contains(strings.Join(strings.Fields(rendered), " "), `<article> <strong>embedded</strong> </article>`) {
		t.Fatalf("expected embed override output, got %q", rendered)
	}
}

func TestTwigAdvancedExpressionsIncludeOnlyAndSetCapture(t *testing.T) {
	appRoot := t.TempDir()
	templatesDir := filepath.Join(appRoot, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTemplate(t, filepath.Join(templatesDir, "partial.html"), `<em>{{ title ?? "none" }}</em>`)
	writeTemplate(t, filepath.Join(templatesDir, "page.html"), `
{{ "Hola" ~ " " ~ name }}
{% include "partial.html" with (dict "title" "Local") only %}
{% include "missing.html" ignore missing %}
{% if 4 is even and "ol" in "hola" %}ok{% endif %}
`)
	engine := New(Config{AppRoot: appRoot, Mode: ModeProd})
	rendered, err := engine.RenderContext(context.Background(), "page.html", map[string]interface{}{"name": "Ada", "title": "Global"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "Hola Ada") {
		t.Fatalf("expected concat output, got %q", rendered)
	}
	if !strings.Contains(rendered, "<em>Local</em>") || strings.Contains(rendered, "<em>Global</em>") {
		t.Fatalf("expected include only semantics, got %q", rendered)
	}
	if !strings.Contains(rendered, "ok") {
		t.Fatalf("expected advanced tests/operators output, got %q", rendered)
	}
}

func writeTemplate(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
