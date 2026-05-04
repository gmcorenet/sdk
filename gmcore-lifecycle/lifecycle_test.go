package gmcore_lifecycle

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAppLayoutForSourceTree(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "src"))
	mustWriteFile(t, filepath.Join(root, "app.yaml"), "name: demo\nversion: 0.2.1\nruntime:\n  entrypoint: ./current/bin/demo\n")
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.19\n")

	layout, err := ResolveAppLayout(root)
	if err != nil {
		t.Fatalf("resolve layout: %v", err)
	}
	if layout.Packaged {
		t.Fatalf("expected source layout, got packaged")
	}
	if layout.ManifestRoot != root || layout.BuildRoot != root || layout.EnvRoot != root {
		t.Fatalf("unexpected layout: %+v", layout)
	}
}

func TestResolveAppLayoutForPackagedInstall(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "current")
	mustMkdirAll(t, filepath.Join(current, "src"))
	mustWriteFile(t, filepath.Join(current, "app.yaml"), "name: demo\nversion: 0.2.1\nruntime:\n  entrypoint: ./current/bin/demo\n")
	mustWriteFile(t, filepath.Join(current, "go.mod"), "module demo\n\ngo 1.19\n")

	layout, err := ResolveAppLayout(root)
	if err != nil {
		t.Fatalf("resolve layout: %v", err)
	}
	if !layout.Packaged {
		t.Fatalf("expected packaged layout")
	}
	if layout.ManifestRoot != current || layout.BuildRoot != current || layout.EnvRoot != root {
		t.Fatalf("unexpected layout: %+v", layout)
	}
}

func TestPathsIncludesInstanceMetadataPath(t *testing.T) {
	paths := NewPaths("/opt/gmcore/bin/demo")
	if got, want := filepath.Base(paths.InstancePath), "app.instance.json"; got != want {
		t.Fatalf("unexpected instance metadata path: got %q want %q", got, want)
	}
}

func TestInstallCreatesRuntimeEnvFromReleaseExample(t *testing.T) {
	t.Skip("Install functionality not yet implemented")
}

func TestInstallPreservesExistingRuntimeEnv(t *testing.T) {
	t.Skip("Install functionality not yet implemented")
}

func TestParseProcEnviron(t *testing.T) {
	values := parseProcEnviron([]byte("GMCORE_MANAGED_LAUNCH=1\x00APP_PID_FILE=/tmp/app.pid\x00APP_DATA_DIR=/tmp/data\x00"))
	if values["GMCORE_MANAGED_LAUNCH"] != "1" {
		t.Fatalf("expected managed launch marker")
	}
	if values["APP_PID_FILE"] != "/tmp/app.pid" {
		t.Fatalf("unexpected pid file value: %q", values["APP_PID_FILE"])
	}
	if values["APP_DATA_DIR"] != "/tmp/data" {
		t.Fatalf("unexpected data dir value: %q", values["APP_DATA_DIR"])
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeLifecycleTestTarGz(path string, files map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	for name, content := range files {
		header := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}
