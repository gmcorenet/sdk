package gmcore_resolver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		AppRoot:     "/app",
		SystemRoot:  "/system",
		BundleRoots: []string{"/bundle1", "/bundle2"},
	}

	if cfg.AppRoot != "/app" {
		t.Errorf("expected AppRoot /app, got %s", cfg.AppRoot)
	}
	if cfg.SystemRoot != "/system" {
		t.Errorf("expected SystemRoot /system, got %s", cfg.SystemRoot)
	}
	if len(cfg.BundleRoots) != 2 {
		t.Errorf("expected 2 bundle roots, got %d", len(cfg.BundleRoots))
	}
}

func TestResolvedFile(t *testing.T) {
	rf := ResolvedFile{
		Path:   "/app/templates/base.html",
		Source: "app",
	}

	if rf.Path != "/app/templates/base.html" {
		t.Errorf("unexpected path: %s", rf.Path)
	}
	if rf.Source != "app" {
		t.Errorf("unexpected source: %s", rf.Source)
	}
}

func TestResolveRelativeFile_AppRoot(t *testing.T) {
	tmp := t.TempDir()
	appFile := filepath.Join(tmp, "test.txt")
	os.WriteFile(appFile, []byte("test"), 0644)

	cfg := Config{
		AppRoot: tmp,
	}
	rf, ok := ResolveRelativeFile(cfg, "test.txt")
	if !ok {
		t.Fatal("expected to resolve file")
	}
	if rf.Source != "app" {
		t.Errorf("expected source 'app', got %s", rf.Source)
	}
}

func TestResolveRelativeFile_BundleRoot(t *testing.T) {
	tmp := t.TempDir()
	bundleDir := filepath.Join(tmp, "mybundle")
	os.Mkdir(bundleDir, 0755)
	bundleFile := filepath.Join(bundleDir, "test.txt")
	os.WriteFile(bundleFile, []byte("test"), 0644)

	bundleManifest := filepath.Join(bundleDir, "bundle.yaml")
	os.WriteFile(bundleManifest, []byte("name: my-bundle\n"), 0644)

	cfg := Config{
		BundleRoots: []string{bundleDir},
	}
	rf, ok := ResolveRelativeFile(cfg, "test.txt")
	if !ok {
		t.Fatal("expected to resolve file from bundle")
	}
	if rf.Source != "my-bundle" {
		t.Errorf("expected source 'my-bundle', got %s", rf.Source)
	}
}

func TestResolveRelativeFile_SystemRoot(t *testing.T) {
	tmp := t.TempDir()
	systemFile := filepath.Join(tmp, "system.txt")
	os.WriteFile(systemFile, []byte("test"), 0644)

	cfg := Config{
		SystemRoot: tmp,
	}
	rf, ok := ResolveRelativeFile(cfg, "system.txt")
	if !ok {
		t.Fatal("expected to resolve file")
	}
	if rf.Source != "system" {
		t.Errorf("expected source 'system', got %s", rf.Source)
	}
}

func TestResolveRelativeFile_Precedence(t *testing.T) {
	tmp := t.TempDir()

	appFile := filepath.Join(tmp, "shared.txt")
	os.WriteFile(appFile, []byte("app"), 0644)

	bundleDir := filepath.Join(tmp, "bundle")
	os.Mkdir(bundleDir, 0755)
	bundleFile := filepath.Join(bundleDir, "shared.txt")
	os.WriteFile(bundleFile, []byte("bundle"), 0644)

	cfg := Config{
		AppRoot:     tmp,
		BundleRoots: []string{bundleDir},
	}
	rf, ok := ResolveRelativeFile(cfg, "shared.txt")
	if !ok {
		t.Fatal("expected to resolve file")
	}
	if rf.Source != "app" {
		t.Errorf("expected precedence to app, got %s", rf.Source)
	}
}

func TestResolveRelativeFile_NotFound(t *testing.T) {
	cfg := Config{
		AppRoot: "/nonexistent",
	}
	_, ok := ResolveRelativeFile(cfg, "nonexistent.txt")
	if ok {
		t.Error("expected not found")
	}
}

func TestResolveRelativeFile_EmptyPath(t *testing.T) {
	cfg := Config{}
	_, ok := ResolveRelativeFile(cfg, "")
	if ok {
		t.Error("expected empty path to fail")
	}

	_, ok = ResolveRelativeFile(cfg, ".")
	if ok {
		t.Error("expected '.' path to fail")
	}
}

func TestResolveRelativeFile_Directory(t *testing.T) {
	tmp := t.TempDir()
	os.Mkdir(filepath.Join(tmp, "mydir"), 0755)

	cfg := Config{
		AppRoot: tmp,
	}
	_, ok := ResolveRelativeFile(cfg, "mydir")
	if ok {
		t.Error("expected directory to be skipped")
	}
}

func TestResolveTemplate(t *testing.T) {
	tmp := t.TempDir()
	templateFile := filepath.Join(tmp, "templates", "base.html")
	os.Mkdir(filepath.Join(tmp, "templates"), 0755)
	os.WriteFile(templateFile, []byte("test"), 0644)

	cfg := Config{
		AppRoot: tmp,
	}
	rf, ok := ResolveTemplate(cfg, "base.html")
	if !ok {
		t.Fatal("expected to resolve template")
	}
	if rf.Source != "app" {
		t.Errorf("expected source 'app', got %s", rf.Source)
	}
}

func TestResolveSource(t *testing.T) {
	tmp := t.TempDir()
	sourceFile := filepath.Join(tmp, "internal", "handler.go")
	os.Mkdir(filepath.Join(tmp, "internal"), 0755)
	os.WriteFile(sourceFile, []byte("package internal"), 0644)

	cfg := Config{
		AppRoot: tmp,
	}
	rf, ok := ResolveSource(cfg, "internal/handler.go")
	if !ok {
		t.Fatal("expected to resolve source")
	}
	if rf.Source != "app" {
		t.Errorf("expected source 'app', got %s", rf.Source)
	}
}

func TestCleanRelative(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{".", ""},
		{"/", ""},
		{"  ", ""},
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"path/to/file.txt", "path/to/file.txt"},
		{"/path/to/file.txt", "path/to/file.txt"},
		{"path/./to/file.txt", "path/to/file.txt"},
		{"path/../to/file.txt", "to/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanRelative(tt.input)
			if result != tt.expected {
				t.Errorf("cleanRelative(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBundleSource(t *testing.T) {
	tmp := t.TempDir()

	t.Run("no manifest", func(t *testing.T) {
		result := bundleSource(tmp)
		if result != filepath.Base(tmp) {
			t.Errorf("expected %s, got %s", filepath.Base(tmp), result)
		}
	})

	t.Run("empty root", func(t *testing.T) {
		result := bundleSource("")
		if result != "bundle" {
			t.Errorf("expected 'bundle', got %s", result)
		}
	})

	t.Run("with bundle.yaml", func(t *testing.T) {
		bundleDir := filepath.Join(tmp, "testbundle")
		os.Mkdir(bundleDir, 0755)
		manifest := filepath.Join(bundleDir, "bundle.yaml")
		os.WriteFile(manifest, []byte("name: test-bundle-name\n"), 0644)

		result := bundleSource(bundleDir)
		if result != "test-bundle-name" {
			t.Errorf("expected 'test-bundle-name', got %s", result)
		}
	})

	t.Run("with bundle.yaml empty name", func(t *testing.T) {
		bundleDir := filepath.Join(tmp, "emptybundle")
		os.Mkdir(bundleDir, 0755)
		manifest := filepath.Join(bundleDir, "bundle.yaml")
		os.WriteFile(manifest, []byte("name: \n"), 0644)

		result := bundleSource(bundleDir)
		if result != "emptybundle" {
			t.Errorf("expected 'emptybundle', got %s", result)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		bundleDir := filepath.Join(tmp, "invalidyaml")
		os.Mkdir(bundleDir, 0755)
		manifest := filepath.Join(bundleDir, "bundle.yaml")
		os.WriteFile(manifest, []byte("invalid: [yaml\n"), 0644)

		result := bundleSource(bundleDir)
		if result != "invalidyaml" {
			t.Errorf("expected 'invalidyaml', got %s", result)
		}
	})

	t.Run("manifest not found", func(t *testing.T) {
		bundleDir := filepath.Join(tmp, "nobundle")
		os.Mkdir(bundleDir, 0755)

		result := bundleSource(bundleDir)
		if result != "nobundle" {
			t.Errorf("expected 'nobundle', got %s", result)
		}
	})
}

func TestResolveRelativeFile_WithMultipleBundleRoots(t *testing.T) {
	tmp := t.TempDir()

	bundle1 := filepath.Join(tmp, "bundle1")
	bundle2 := filepath.Join(tmp, "bundle2")
	os.Mkdir(bundle1, 0755)
	os.Mkdir(bundle2, 0755)

	os.WriteFile(filepath.Join(bundle1, "test.txt"), []byte("bundle1"), 0644)
	os.WriteFile(filepath.Join(bundle2, "test.txt"), []byte("bundle2"), 0644)

	bundle1Manifest := filepath.Join(bundle1, "bundle.yaml")
	os.WriteFile(bundle1Manifest, []byte("name: bundle-one\n"), 0644)
	bundle2Manifest := filepath.Join(bundle2, "bundle.yaml")
	os.WriteFile(bundle2Manifest, []byte("name: bundle-two\n"), 0644)

	cfg := Config{
		BundleRoots: []string{bundle1, bundle2},
	}

	rf, ok := ResolveRelativeFile(cfg, "test.txt")
	if !ok {
		t.Fatal("expected to resolve file")
	}
	if rf.Source != "bundle-one" {
		t.Errorf("expected first bundle to win, got %s", rf.Source)
	}
}
