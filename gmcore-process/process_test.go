package gmcore_process

import (
	"context"
	"testing"
	"time"
)

func TestSpawnAndWait(t *testing.T) {
	p, err := Spawn("echo", []string{"hello"}, nil)
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	if p.Pid() <= 0 {
		t.Error("invalid pid")
	}

	exitCode, err := p.Wait()
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun(t *testing.T) {
	result, err := Run("echo", []string{"hello"}, nil)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout == "" {
		t.Error("expected stdout to have content")
	}
}

func TestRunWithArgs(t *testing.T) {
	result, err := Run("/bin/sh", []string{"-c", "echo hello world"}, nil)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRunWithTimeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := RunWithContext(ctx, "sleep", []string{"2"}, nil)
	duration := time.Since(start)

	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("run failed with unexpected error: %v", err)
	}

	if duration.Seconds() > 1.0 {
		t.Errorf("command should have been killed by timeout, took %v", duration)
	}
}

func TestProcessKill(t *testing.T) {
	p, err := Spawn("sleep", []string{"10"}, nil)
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	err = p.Kill()
	if err != nil {
		t.Fatalf("kill failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if p.Running() {
		t.Error("process should not be running after kill")
	}
}

func TestProcessRunning(t *testing.T) {
	p, err := Spawn("sleep", []string{"10"}, nil)
	if err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	if !p.Running() {
		t.Error("process should be running")
	}

	p.Kill()

	time.Sleep(50 * time.Millisecond)

	if p.Running() {
		t.Error("process should not be running after kill")
	}
}

func TestLookPath(t *testing.T) {
	path, err := LookPath("ls")
	if err != nil {
		t.Fatalf("LookPath failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}

	_, err = LookPath("nonexistent_command_12345")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestRunStreaming(t *testing.T) {
	var stdoutLines []string
	result, err := RunStreaming("/bin/sh", []string{"-c", "echo line1; echo line2"}, nil,
		func(line string) {
			stdoutLines = append(stdoutLines, line)
		}, nil)

	if err != nil {
		t.Fatalf("run streaming failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if len(stdoutLines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(stdoutLines))
	}
}
