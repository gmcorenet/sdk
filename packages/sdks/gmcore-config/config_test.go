package gmcore_config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAMLResolvesEnvAndParameters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.yaml")
	if err := os.WriteFile(path, []byte(`
parameters:
  base_dsn: "%env(APP_DB_DSN)%"
database:
  dsn: "%parameter.base_dsn%"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Database struct {
			DSN string `yaml:"dsn"`
		} `yaml:"database"`
	}
	err := LoadYAML(path, &cfg, Options{Env: map[string]string{"APP_DB_DSN": "data/app.sqlite"}, Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "data/app.sqlite" {
		t.Fatalf("expected env resolved DSN, got %q", cfg.Database.DSN)
	}
}

func TestLoadAppEnvMapsAppPrefixedKeys(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo-app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DEMO_APP_HTTP_ADDR=:8080\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := LoadAppEnv(dir)
	if env["APP_HTTP_ADDR"] != ":8080" {
		t.Fatalf("expected mapped APP_HTTP_ADDR, got %q", env["APP_HTTP_ADDR"])
	}
}
