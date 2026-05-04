package gmcore_i18n

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTranslatorInterpolationAndPluralization(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "en.yaml"), []byte(`
messages:
  welcome: "Hello %name%"
  items:
    zero: "No items"
    one: "One item"
    other: "%count% items"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.T("en", "messages.welcome", Params{"name": "Ada"}); got != "Hello Ada" {
		t.Fatalf("unexpected translation: %q", got)
	}
	if got := translator.TC("en", "messages.items", 0); got != "No items" {
		t.Fatalf("unexpected zero plural: %q", got)
	}
	if got := translator.TC("en", "messages.items", 1); got != "One item" {
		t.Fatalf("unexpected singular plural: %q", got)
	}
	if got := translator.TC("en", "messages.items", 4); got != "4 items" {
		t.Fatalf("unexpected plural: %q", got)
	}
}

func TestTranslatorLocaleFallbackAndRequestParsing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "es.yaml"), []byte(`
messages:
  title: "Hola"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "es")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.T("es-mx", "messages.title"); got != "Hola" {
		t.Fatalf("expected locale fallback, got %q", got)
	}
	req := httptest.NewRequest("GET", "/?_locale=es-MX", nil)
	if got := LocaleFromRequest(req, translator); got != "es" {
		t.Fatalf("expected locale resolution, got %q", got)
	}
}

func TestTranslatorDomainsMissingTranslationsAndFrontendPayload(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "messages.en.yaml"), []byte(`
messages:
  title: "Hello"
  nav:
    home: "Home"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "messages.es.yaml"), []byte(`
messages:
  title: "Hola"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "admin.en.yaml"), []byte(`
dashboard:
  title: "Admin Dashboard"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "admin.es.yaml"), []byte(`
dashboard:
  title: "Panel"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	translator, err := LoadDir(root, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.T("es", "admin:dashboard.title"); got != "Panel" {
		t.Fatalf("unexpected domain translation: %q", got)
	}
	namespaces := translator.Namespaces("es")
	if len(namespaces) != 2 || namespaces[0] != "admin" || namespaces[1] != "messages" {
		t.Fatalf("unexpected namespaces: %#v", namespaces)
	}
	missing := translator.MissingTranslations("es", "domain=messages")
	if len(missing) != 1 || missing[0] != "messages.nav.home" {
		t.Fatalf("unexpected missing translations: %#v", missing)
	}
	payload := translator.FrontendPayload("es", "domain=messages", "prefix=messages.nav")
	if payload.Locale != "es" || payload.Domain != "messages" || payload.Hash == "" || payload.Version == "" || len(payload.Messages) != 1 || payload.Messages["messages.nav.home"] == "" {
		t.Fatalf("unexpected frontend payload: %#v", payload)
	}
	json := translator.FrontendJSON("es", "domain=messages", "prefix=messages.nav")
	if !strings.Contains(json, `"domain":"messages"`) || !strings.Contains(json, `"messages.nav.home"`) {
		t.Fatalf("unexpected frontend json: %s", json)
	}
	audit := translator.AuditLocale("es", "domain=messages")
	if len(audit.Missing) != 1 || audit.Coverage >= 1 {
		t.Fatalf("unexpected audit report: %#v", audit)
	}
}

func TestExtractionHelpersAndNamespaceFromPath(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "ExampleController.go")
	content := `
package sample
func test() {
  _ = trans("messages:title")
  _ = T("admin:dashboard.title")
  _ = TC("messages:items", 2)
}
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	keys := ExtractKeysFromContent(content)
	if len(keys) != 3 {
		t.Fatalf("unexpected extracted keys: %#v", keys)
	}
	report, err := ExtractKeysFromFiles(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Keys) != 3 || len(report.ByFile[filePath]) != 3 {
		t.Fatalf("unexpected extraction report: %#v", report)
	}
	namespace := NamespaceFromPath("/bundle/security/templates/auth/login.html")
	if namespace != "security.auth.login" {
		t.Fatalf("unexpected namespace: %s", namespace)
	}
}

func TestLocaleAwarePluralization(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ru.yaml"), []byte(`
messages:
  cars:
    one: "%count% машина"
    few: "%count% машины"
    many: "%count% машин"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "ru")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.TC("ru", "messages.cars", 2); got != "2 машины" {
		t.Fatalf("unexpected russian plural form: %q", got)
	}
	if got := translator.TC("ru", "messages.cars", 5); got != "5 машин" {
		t.Fatalf("unexpected russian plural form: %q", got)
	}
}

func TestTranslatorDomainsAndFrontendExport(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "messages.en.yaml"), []byte(`
messages:
  title: "Hello"
  dashboard:
    subtitle: "Runtime"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "admin.en.yaml"), []byte(`
messages:
  title: "Admin"
  menu:
    users: "Users"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.TDomain("en", "admin", "messages.title"); got != "Admin" {
		t.Fatalf("unexpected domain translation: %q", got)
	}
	payload := translator.Messages("en", "admin", "prefix=messages.menu")
	if len(payload) != 1 || payload["messages.menu.users"] != "Users" {
		t.Fatalf("unexpected filtered payload: %#v", payload)
	}
	if missing := translator.MissingKeys("en", "admin:messages.menu.users", "admin:messages.unknown"); len(missing) != 1 || missing[0] != "admin:messages.unknown" {
		t.Fatalf("unexpected missing keys: %#v", missing)
	}
}

func TestICUPluralAndSelectFormatting(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "messages.en.yaml"), []byte(`
cart:
  items: "{count, plural, =0 {No items} one {One item} other {# items}}"
notice:
  state: "{state, select, draft {Draft} published {Published} other {Unknown}}"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.T("en", "messages:cart.items", Params{"count": 0}); got != "No items" {
		t.Fatalf("unexpected ICU plural: %q", got)
	}
	if got := translator.T("en", "messages:cart.items", Params{"count": 3}); got != "3 items" {
		t.Fatalf("unexpected ICU plural: %q", got)
	}
	if got := translator.T("en", "messages:notice.state", Params{"state": "published"}); got != "Published" {
		t.Fatalf("unexpected ICU select: %q", got)
	}
}

func TestAuditSourcesAndExtractKeysFromTree(t *testing.T) {
	root := t.TempDir()
	translationsDir := filepath.Join(root, "translations")
	sourcesDir := filepath.Join(root, "src")
	if err := os.MkdirAll(translationsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(translationsDir, "messages.en.yaml"), []byte(`
messages:
  title: "Hello"
  nav:
    home: "Home"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "MainController.go"), []byte(`
package sample
func demo() {
  _ = T("messages:title")
  _ = T("messages:nav.home")
  _ = T("messages:nav.missing")
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(translationsDir, "en")
	if err != nil {
		t.Fatal(err)
	}
	report, err := ExtractKeysFromTree(sourcesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Keys) != 3 {
		t.Fatalf("unexpected extracted keys: %#v", report)
	}
	audit, err := translator.AuditSources("en", []string{"domain=messages"}, sourcesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(audit.Missing) != 1 || audit.Missing[0] != "messages:nav.missing" {
		t.Fatalf("unexpected source audit: %#v", audit)
	}
}

func TestICUExtendedOrdinalNumberAndDate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "messages.en.yaml"), []byte(`
report:
  position: "{place, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}"
  total: "{amount, number}"
  when: "{publishedAt, date}"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	translator, err := LoadDir(root, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := translator.T("en", "messages:report.position", Params{"place": 2}); got != "2nd" {
		t.Fatalf("unexpected ordinal ICU: %q", got)
	}
	if got := translator.T("en", "messages:report.total", Params{"amount": 12345.5}); got != "12,345.5" {
		t.Fatalf("unexpected number ICU: %q", got)
	}
	date := time.Date(2026, 4, 7, 13, 5, 0, 0, time.UTC)
	if got := translator.T("en", "messages:report.when", Params{"publishedAt": date}); got != "2026-04-07" {
		t.Fatalf("unexpected date ICU: %q", got)
	}
}
