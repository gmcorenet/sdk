package gmcoreinstaller

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerInstallAndRemove(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "skeleton", "config"), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "skeleton", "config", "app.yaml"), []byte("name: demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	recipe := Recipe{
		Install: []Step{
			{Action: "ensure_dir", Path: "var/cache"},
			{Action: "copy_tree", From: "skeleton", To: ".", Overwrite: true},
			{Action: "write_file", Path: "README.md", Content: "demo", Overwrite: true},
		},
		Remove: []Step{
			{Action: "remove_file", Path: "README.md"},
		},
	}
	runner := Runner{SourceRoot: source, TargetRoot: target}
	if err := runner.RunInstall(recipe); err != nil {
		t.Fatalf("install recipe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "config", "app.yaml")); err != nil {
		t.Fatalf("expected copied app.yaml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "var", "cache")); err != nil {
		t.Fatalf("expected cache dir: %v", err)
	}
	if err := runner.RunRemove(recipe); err != nil {
		t.Fatalf("remove recipe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected README removed, err=%v", err)
	}
}

func TestRunnerRejectsPathTraversal(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	runner := Runner{SourceRoot: source, TargetRoot: target}
	err := runner.RunInstall(Recipe{Install: []Step{{Action: "write_file", Path: "../escape.txt", Content: "no"}}})
	if err == nil {
		t.Fatal("expected path traversal error")
	}
}

func TestRunnerRequiresConfirmation(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	runner := Runner{SourceRoot: source, TargetRoot: target}
	err := runner.RunInstall(Recipe{Install: []Step{{Action: "run", Command: "true", RequiresConfirmation: true}}})
	if err == nil {
		t.Fatal("expected confirmation error")
	}
}

func TestRunnerCopyFileCanPromptBeforeOverwrite(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, ".env.example"), []byte("APP_NAME=demo\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	envPath := filepath.Join(target, ".env")
	if err := os.WriteFile(envPath, []byte("APP_NAME=existing\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	recipe := Recipe{Install: []Step{{Action: "copy_file", From: ".env.example", To: ".env", PromptOverwrite: true}}}
	runner := Runner{SourceRoot: source, TargetRoot: target, Stdin: strings.NewReader("n\n")}
	if err := runner.RunInstall(recipe); err != nil {
		t.Fatalf("decline overwrite: %v", err)
	}
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "APP_NAME=existing\n" {
		t.Fatalf("expected existing env to remain, got %q", string(data))
	}
	runner.Stdin = strings.NewReader("y\n")
	if err := runner.RunInstall(recipe); err != nil {
		t.Fatalf("confirm overwrite: %v", err)
	}
	data, err = os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "APP_NAME=demo\n" {
		t.Fatalf("expected env overwrite, got %q", string(data))
	}
}

func TestPackageInstallCommandManagers(t *testing.T) {
	tests := []struct {
		action  string
		manager string
		name    string
		binary  string
		args    []string
	}{
		{action: "apt_package", name: "mysql-server", binary: "apt-get", args: []string{"install", "-y", "mysql-server"}},
		{action: "os_package", manager: "dnf", name: "mysql-server", binary: "dnf", args: []string{"install", "-y", "mysql-server"}},
		{action: "yum_package", name: "mysql-server", binary: "yum", args: []string{"install", "-y", "mysql-server"}},
		{action: "apk_package", name: "mysql", binary: "apk", args: []string{"add", "mysql"}},
		{action: "zypper_package", name: "mysql", binary: "zypper", args: []string{"--non-interactive", "install", "mysql"}},
		{action: "pacman_package", name: "mysql", binary: "pacman", args: []string{"-S", "--noconfirm", "mysql"}},
	}
	for _, tt := range tests {
		binary, args, err := packageInstallCommand(tt.action, tt.manager, tt.name)
		if err != nil {
			t.Fatalf("%s/%s: %v", tt.action, tt.manager, err)
		}
		if binary != tt.binary {
			t.Fatalf("expected binary %s, got %s", tt.binary, binary)
		}
		if len(args) != len(tt.args) {
			t.Fatalf("expected args %#v, got %#v", tt.args, args)
		}
		for idx := range args {
			if args[idx] != tt.args[idx] {
				t.Fatalf("expected args %#v, got %#v", tt.args, args)
			}
		}
	}
}

func TestRunnerRunRequiresExecutablePermission(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	script := filepath.Join(source, "script.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env sh\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	runner := Runner{SourceRoot: source, TargetRoot: target}
	err := runner.RunInstall(Recipe{Install: []Step{{Action: "run", Path: "script.sh"}}})
	if err == nil {
		t.Fatal("expected executable permission error")
	}
}

func TestRunnerExtractArchiveTarGz(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	archivePath := filepath.Join(source, "demo.tar.gz")
	if err := writeTestTarGz(archivePath, map[string]string{"pkg/README.md": "hello"}); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	runner := Runner{SourceRoot: source, TargetRoot: target}
	err := runner.RunInstall(Recipe{Install: []Step{{Action: "extract_archive", From: "demo.tar.gz", To: "tools/demo", StripComponents: 1}}})
	if err != nil {
		t.Fatalf("extract archive: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(target, "tools", "demo", "README.md"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected extracted content, got %q", string(data))
	}
}

func TestRunnerExtractArchiveRejectsTraversal(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	archivePath := filepath.Join(source, "bad.tar.gz")
	if err := writeTestTarGz(archivePath, map[string]string{"../escape.txt": "no"}); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	runner := Runner{SourceRoot: source, TargetRoot: target}
	err := runner.RunInstall(Recipe{Install: []Step{{Action: "extract_archive", From: "bad.tar.gz", To: "."}}})
	if err == nil {
		t.Fatal("expected traversal error")
	}
}

func writeTestTarGz(path string, files map[string]string) error {
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
