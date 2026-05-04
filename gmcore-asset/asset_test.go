package gmcore_asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestLoadAndPackageURL(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.json")
	manifestJSON := `{"assets":{"/css/app.css":{"version":"/css/app.123.css"}}}`
	if err := os.WriteFile(manifestPath, []byte(manifestJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadManifestFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manager := Manager{BaseURL: "/assets", Manifest: manifest, Version: "1"}
	url := manager.URL("/css/app.css")
	if !strings.Contains(url, "/css/app.123.css") || !strings.Contains(url, "v=1") {
		t.Fatalf("unexpected asset url: %s", url)
	}
	if packageURL := manager.PackageURL("admin", "css/panel.css"); !strings.Contains(packageURL, "/admin/css/panel.css") {
		t.Fatalf("unexpected package url: %s", packageURL)
	}
}
