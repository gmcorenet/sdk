package gmcore_options_resolver

import (
	"testing"
)

func TestAddOptions(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")
	opts.AddInt("age", "Your age", 30)
	opts.AddBool("verbose", "Verbose mode", false)

	if len(opts.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(opts.options))
	}
}

func TestParseStringOption(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{"--name=Jane"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	val, err := opts.GetString("name")
	if err != nil {
		t.Fatalf("GetString failed: %v", err)
	}
	if val != "Jane" {
		t.Errorf("expected Jane, got %s", val)
	}
}

func TestParseIntOption(t *testing.T) {
	opts := New()
	opts.AddInt("port", "Port number", 8080)

	err := opts.Parse([]string{"--port", "3000"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	val, err := opts.GetInt("port")
	if err != nil {
		t.Fatalf("GetInt failed: %v", err)
	}
	if val != 3000 {
		t.Errorf("expected 3000, got %d", val)
	}
}

func TestParseBoolFlag(t *testing.T) {
	opts := New()
	opts.AddBool("verbose", "Verbose mode", false)

	err := opts.Parse([]string{"--verbose"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	val, err := opts.GetBool("verbose")
	if err != nil {
		t.Fatalf("GetBool failed: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestParseShortFlag(t *testing.T) {
	opts := New()
	opts.AddFlag("verbose", "v", "Verbose mode")

	err := opts.Parse([]string{"-v"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	val, err := opts.GetBool("verbose")
	if err != nil {
		t.Fatalf("GetBool failed: %v", err)
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestParsePositionalArgs(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{"arg1", "arg2"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	positional := opts.GetPositional()
	if len(positional) != 2 {
		t.Errorf("expected 2 positional args, got %d", len(positional))
	}
}

func TestRequiredOption(t *testing.T) {
	opts := New()
	opts.AddStringRequired("name", "Your name")

	err := opts.Parse([]string{})
	if err == nil {
		t.Error("expected error for missing required option")
	}
}

func TestIsSet(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	if opts.IsSet("name") {
		t.Error("name should not be set before parsing")
	}

	opts.Parse([]string{"--name=Jane"})
	if !opts.IsSet("name") {
		t.Error("name should be set after parsing with value")
	}

	opts2 := New()
	opts2.AddString("name", "Your name", "John")
	opts2.Parse([]string{})
	if opts2.IsSet("name") {
		t.Error("name should not be set when using default value")
	}
}

func TestGetStringWrongType(t *testing.T) {
	opts := New()
	opts.AddInt("age", "Your age", 30)

	err := opts.Parse([]string{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, err = opts.GetString("age")
	if err == nil {
		t.Error("expected error when getting string as int option")
	}
}

func TestGetIntWrongType(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, err = opts.GetInt("name")
	if err == nil {
		t.Error("expected error when getting int as string option")
	}
}

func TestGetBoolWrongType(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	_, err = opts.GetBool("name")
	if err == nil {
		t.Error("expected error when getting bool as string option")
	}
}

func TestGetPositionalAt(t *testing.T) {
	opts := New()
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{"arg1", "arg2", "arg3"})
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	val, err := opts.GetPositionalAt(0)
	if err != nil {
		t.Fatalf("GetPositionalAt failed: %v", err)
	}
	if val != "arg1" {
		t.Errorf("expected arg1, got %s", val)
	}

	_, err = opts.GetPositionalAt(10)
	if err == nil {
		t.Error("expected error for out of bounds index")
	}
}

func TestStrictMode(t *testing.T) {
	opts := New()
	opts.SetStrictMode(true)
	opts.AddString("name", "Your name", "John")

	err := opts.Parse([]string{"--unknown"})
	if err == nil {
		t.Error("expected error in strict mode for unknown option")
	}
}
