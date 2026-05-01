package gmcoreconsole

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpListsDefaultCommands(t *testing.T) {
	var out bytes.Buffer
	console := New(t.TempDir(), WithOutput(&out, &out))
	if err := console.Run(nil); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(out.String(), "make") || !strings.Contains(out.String(), "run") {
		t.Fatalf("help missing default commands: %s", out.String())
	}
}

func TestCommandHelpShowsSpecificUsage(t *testing.T) {
	var out bytes.Buffer
	console := New(t.TempDir(), WithOutput(&out, &out))
	if err := console.Run([]string{"help", "make"}); err != nil {
		t.Fatalf("command help failed: %v", err)
	}
	if !strings.Contains(out.String(), "bin/console make <maker> <name>") {
		t.Fatalf("help missing make usage: %s", out.String())
	}
}

func TestExecutesAppLocalCommand(t *testing.T) {
	appRoot := t.TempDir()
	commandPath := filepath.Join(appRoot, "bin", "gmcore", "commands", "hello")
	if err := os.MkdirAll(filepath.Dir(commandPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(commandPath, []byte("#!/usr/bin/env sh\necho hello-$1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	console := New(appRoot, WithOutput(&out, &out))
	if err := console.Run([]string{"run", "hello", "world"}); err != nil {
		t.Fatalf("run command failed: %v", err)
	}
	if strings.TrimSpace(out.String()) != "hello-world" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
