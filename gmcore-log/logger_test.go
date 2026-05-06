package gmcore_log

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestLogger_AddHandler_Nil(t *testing.T) {
	logger := New()
	logger.AddHandler(nil)
	logger.Info("test")
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetLevel(LevelWarn)
	logger.AddHandler(NewConsoleHandler(&buf))
	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	output := buf.String()
	if bytes.Contains([]byte(output), []byte("debug msg")) {
		t.Error("debug should be filtered")
	}
	if bytes.Contains([]byte(output), []byte("info msg")) {
		t.Error("info should be filtered")
	}
	if !bytes.Contains([]byte(output), []byte("warn msg")) {
		t.Error("warn should not be filtered")
	}
}

func TestLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetLevel(LevelDebug)
	logger.AddHandler(NewConsoleHandler(&buf))
	logger.WithField("request_id", "abc123").Info("request processed")

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("request_id=abc123")) {
		t.Error("expected field in output, got:", output)
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New()
	logger.SetLevel(LevelDebug)
	logger.AddHandler(NewConsoleHandler(&buf))
	logger.WithFields(map[string]interface{}{
		"user": "alice",
		"ip":   "192.168.1.1",
	}).Info("login")

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("user=alice")) {
		t.Error("expected user field in output, got:", output)
	}
	if !bytes.Contains([]byte(output), []byte("ip=192.168.1.1")) {
		t.Error("expected ip field in output, got:", output)
	}
}

func TestConsoleHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(&buf)
	h.Handle(Entry{
		Time:    time.Now(),
		Level:   LevelInfo,
		Message: "hello world",
		Fields:  nil,
	})
	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestTextFormat_FormatString(t *testing.T) {
	f := TextFormat{}
	out := f.FormatString(Entry{
		Time:    time.Now(),
		Level:   LevelError,
		Message: "test message",
		Fields:  map[string]interface{}{"key": "value"},
	})
	if out == "" {
		t.Error("expected formatted output")
	}
}

func TestJSONFormat_FormatString(t *testing.T) {
	f := JSONFormat{}
	out := f.FormatString(Entry{
		Time:    time.Now(),
		Level:   LevelInfo,
		Message: "test",
		Fields:  nil,
	})
	if out == "" {
		t.Error("expected output")
	}
}

func TestJSONFormat_FormatString_WithFields(t *testing.T) {
	f := JSONFormat{}
	out := f.FormatString(Entry{
		Time:    time.Now(),
		Level:   LevelInfo,
		Message: "test",
		Fields:  map[string]interface{}{"foo": "bar"},
	})
	if out == "" {
		t.Error("expected output")
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{Level(100), "UNKNOWN"},
	}
	for _, tt := range tests {
		if tt.level.String() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.level.String())
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"0", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"1", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"WARNING", LevelWarn},
		{"2", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"3", LevelError},
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"4", LevelFatal},
		{"unknown", LevelInfo},
	}
	for _, tt := range tests {
		got := ParseLevel(tt.input)
		if got != tt.expected {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestFileHandler_Handle(t *testing.T) {
	tmp := t.TempDir() + "/test.log"
	h, err := NewFileHandler(tmp)
	if err != nil {
		t.Fatalf("NewFileHandler: %v", err)
	}
	defer h.Close()
	h.Handle(Entry{Time: time.Now(), Level: LevelInfo, Message: "test", Fields: nil})
	content, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(content) == 0 {
		t.Error("expected file content")
	}
}

func TestNewRotatingFileHandler(t *testing.T) {
	tmp := t.TempDir() + "/rotating.log"
	h, err := NewRotatingFileHandler(tmp, 1024, 3)
	if err != nil {
		t.Fatalf("NewRotatingFileHandler: %v", err)
	}
	defer h.Close()
	if h.MaxBackups != 3 {
		t.Errorf("expected MaxBackups=3, got %d", h.MaxBackups)
	}
}

func TestSyslogHandler_Handle(t *testing.T) {
	h, err := NewSyslogHandler()
	if err != nil {
		t.Fatalf("NewSyslogHandler: %v", err)
	}
	h.Handle(Entry{Time: time.Now(), Level: LevelInfo, Message: "syslog test", Fields: nil})
}

func TestHandlerFunc(t *testing.T) {
	var called bool
	f := HandlerFunc(func(e Entry) {
		called = true
	})
	f.Handle(Entry{Time: time.Now(), Level: LevelInfo, Message: "test", Fields: nil})
	if !called {
		t.Error("HandlerFunc.Handle should call underlying func")
	}
}

func TestRotatingFileHandler_Rotation(t *testing.T) {
	tmp := t.TempDir() + "/rotate_test.log"
	h, err := NewRotatingFileHandler(tmp, 10, 2)
	if err != nil {
		t.Fatalf("NewRotatingFileHandler: %v", err)
	}
	defer h.Close()
	for i := 0; i < 5; i++ {
		h.Handle(Entry{Time: time.Now(), Level: LevelInfo, Message: "xxxxxxxxxx", Fields: nil})
	}
}
