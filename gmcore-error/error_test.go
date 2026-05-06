package gmcore_error

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeNotFound, "resource not found")
	if err == nil {
		t.Fatal("New returned nil")
	}
	if err.Code() != CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %d", err.Code())
	}
	if err.Error() != "resource not found" {
		t.Fatalf("expected 'resource not found', got %s", err.Error())
	}
	if err.message != "resource not found" {
		t.Fatalf("expected message 'resource not found', got %s", err.message)
	}
	if err.technical != "resource not found" {
		t.Fatalf("expected technical 'resource not found', got %s", err.technical)
	}
	if len(err.StackTrace()) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
}

func TestNewTech(t *testing.T) {
	err := NewTech(CodeInternal, "sql: connection refused at 10.0.0.1:5432")
	if err == nil {
		t.Fatal("NewTech returned nil")
	}
	if err.Code() != CodeInternal {
		t.Fatalf("expected CodeInternal, got %d", err.Code())
	}
	if err.Technical() != "sql: connection refused at 10.0.0.1:5432" {
		t.Fatalf("unexpected technical: %s", err.Technical())
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps a standard error", func(t *testing.T) {
		stdErr := errors.New("connection refused")
		err := Wrap(stdErr, CodeNetwork, "database unavailable")

		if err == nil {
			t.Fatal("Wrap returned nil")
		}
		if err.Code() != CodeNetwork {
			t.Fatalf("expected CodeNetwork, got %d", err.Code())
		}
		if err.Error() != "database unavailable" {
			t.Fatalf("expected 'database unavailable', got %s", err.Error())
		}
		if err.Unwrap() != stdErr {
			t.Fatal("Unwrap should return the original error")
		}
	})

	t.Run("returns nil for nil input", func(t *testing.T) {
		err := Wrap(nil, CodeInternal, "msg")
		if err != nil {
			t.Fatal("Wrap(nil) should return nil")
		}
	})

	t.Run("returns the same GmcoreError if already wrapped", func(t *testing.T) {
		original := New(CodeNotFound, "not found")
		wrapped := Wrap(original, CodeInternal, "should not change")
		if wrapped != original {
			t.Fatal("Wrap should return the same GmcoreError instance")
		}
	})
}

func TestWrapTech(t *testing.T) {
	stdErr := fmt.Errorf("i/o timeout")
	err := WrapTech(stdErr, CodeTimeout, "read timed out")
	if err == nil {
		t.Fatal("WrapTech returned nil")
	}
	if !errors.Is(err, stdErr) {
		t.Fatal("errors.Is should find the original error")
	}
}

func TestGmcoreError_WithMethods(t *testing.T) {
	err := New(CodeInvalidInput, "validation failed")
	err.WithMessage("bad email format").WithTechnical("email validation regex failed")
	if err.Error() != "bad email format" {
		t.Fatalf("expected 'bad email format', got %s", err.Error())
	}
	if err.Technical() != "email validation regex failed" {
		t.Fatalf("unexpected technical: %s", err.Technical())
	}
}

func TestGmcoreError_WithCause(t *testing.T) {
	root := errors.New("root cause")
	err := New(CodeInternal, "processing error").WithCause(root)
	if err.Unwrap() != root {
		t.Fatal("Unwrap should return the cause")
	}
}

func TestGmcoreError_String(t *testing.T) {
	err := New(CodeNotFound, "page not found")
	s := err.String()
	if len(s) == 0 {
		t.Fatal("String should not be empty")
	}
	if !contains(s, "NOT_FOUND") {
		t.Fatal("String should contain code string")
	}
}

func TestIsGmcoreError(t *testing.T) {
	t.Run("GmcoreError is recognized", func(t *testing.T) {
		if !IsGmcoreError(New(CodeInternal, "err")) {
			t.Fatal("IsGmcoreError should return true")
		}
	})
	t.Run("standard error is not recognized", func(t *testing.T) {
		if IsGmcoreError(errors.New("nope")) {
			t.Fatal("IsGmcoreError should return false for standard error")
		}
	})
}

func TestAsGmcoreError(t *testing.T) {
	original := New(CodeForbidden, "access denied")
	retrieved, ok := AsGmcoreError(original)
	if !ok {
		t.Fatal("AsGmcoreError should return ok=true")
	}
	if retrieved != original {
		t.Fatal("should return the same instance")
	}

	_, ok = AsGmcoreError(errors.New("plain"))
	if ok {
		t.Fatal("AsGmcoreError should return ok=false for standard error")
	}
}

func TestCodeToString(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{CodeNotFound, "NOT_FOUND"},
		{CodeInvalidInput, "INVALID_INPUT"},
		{CodeUnauthorized, "UNAUTHORIZED"},
		{CodeForbidden, "FORBIDDEN"},
		{CodeInternal, "INTERNAL"},
		{CodeNetwork, "NETWORK"},
		{CodeTimeout, "TIMEOUT"},
		{CodeNotImplemented, "NOT_IMPLEMENTED"},
		{CodeConfiguration, "CONFIGURATION"},
		{CodeUnknown, "UNKNOWN"},
		{ErrorCode(999), "UNKNOWN"},
	}
	for _, tt := range tests {
		if codeToString(tt.code) != tt.expected {
			t.Fatalf("codeToString(%d) = %s, want %s", tt.code, codeToString(tt.code), tt.expected)
		}
	}
}

func TestExitCodeMapping(t *testing.T) {
	e := New(CodeNotFound, "not found")
	if e.ExitCode() != ExitNoInput {
		t.Fatalf("expected ExitNoInput, got %d", e.ExitCode())
	}

	e2 := New(CodeForbidden, "forbidden")
	if e2.ExitCode() != ExitNoPerm {
		t.Fatalf("expected ExitNoPerm, got %d", e2.ExitCode())
	}
}

func TestExitCode_ToErrorCode(t *testing.T) {
	if ExitConfig.ToErrorCode() != CodeConfiguration {
		t.Fatalf("ExitConfig should map to CodeConfiguration")
	}
	if ExitUsage.ToErrorCode() != CodeInvalidInput {
		t.Fatalf("ExitUsage should map to CodeInvalidInput")
	}
	if ExitNoPerm.ToErrorCode() != CodeForbidden {
		t.Fatalf("ExitNoPerm should map to CodeForbidden")
	}
}

func TestErrorCodeValues(t *testing.T) {
	if CodeUnknown != 1 {
		t.Fatalf("CodeUnknown should be 1, got %d", CodeUnknown)
	}
	if CodeNotFound != 2 {
		t.Fatalf("CodeNotFound should be 2, got %d", CodeNotFound)
	}
}

func TestGmcoreError_Error_ReturnsMessage(t *testing.T) {
	e := &GmcoreError{message: "user message", technical: "tech details"}
	if e.Error() != "user message" {
		t.Fatal("Error() should return message")
	}
	e2 := &GmcoreError{technical: "tech only"}
	if e2.Error() != "tech only" {
		t.Fatal("Error() should return technical when message is empty")
	}
}

func TestTry(t *testing.T) {
	err := Try(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("Try should return nil for successful function, got %v", err)
	}

	err = Try(func() error {
		return errors.New("failed")
	})
	if err == nil {
		t.Fatal("Try should return the error")
	}
}

func TestTryValue(t *testing.T) {
	val, err := TryValue(func() string { return "hello" })
	if err != nil {
		t.Fatalf("TryValue should not error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %s", val)
	}

	val2, err2 := TryValue(func() string { panic("boom") })
	if err2 == nil {
		t.Fatal("TryValue should capture panic as error")
	}
	if val2 != "" {
		t.Fatalf("expected zero value on panic, got %s", val2)
	}
}

func TestMust(t *testing.T) {
	val := Must(42, nil)
	if val != 42 {
		t.Fatalf("Must should return the value: %d", val)
	}
}

func TestSetPanicHandler(t *testing.T) {
	original := panicHandler
	SetPanicHandler(func(v interface{}) {
		_ = v
	})
	if panicHandler == nil {
		t.Fatal("panic handler should be set")
	}
	SetPanicHandler(original)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
