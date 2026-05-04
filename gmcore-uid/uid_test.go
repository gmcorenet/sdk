package gmcore_uid

import (
	"testing"
)

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550e8400-e29b-41d4-a716-44665544000", false},
		{"not-a-uuid", false},
		{"", false},
		{"   ", false},
		{"550e8400-e29b-41d4-a716-446655440000 ", true},
	}

	for _, tt := range tests {
		result := IsValidUUID(tt.value)
		if result != tt.expected {
			t.Errorf("IsValidUUID(%q) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestIsValidV4(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550e8400-e29b-11d4-a716-446655440000", false},
		{"not-a-uuid", false},
	}

	for _, tt := range tests {
		result := IsValidV4(tt.value)
		if result != tt.expected {
			t.Errorf("IsValidV4(%q) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestParseUUID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	uuid, err := ParseUUID(valid)
	if err != nil {
		t.Fatalf("ParseUUID(%q) failed: %v", valid, err)
	}

	got := uuid.String()
	if got != valid {
		t.Errorf("uuid.String() = %q, want %q", got, valid)
	}
}

func TestParseUUIDInvalid(t *testing.T) {
	_, err := ParseUUID("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestMustParseUUID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	uuid := MustParseUUID(valid)

	got := uuid.String()
	if got != valid {
		t.Errorf("uuid.String() = %q, want %q", got, valid)
	}
}

func TestMustParseUUIDPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid UUID")
		}
	}()

	MustParseUUID("not-a-uuid")
}

func TestIsValidPrimaryKey(t *testing.T) {
	if err := IsValidPrimaryKey("550e8400-e29b-41d4-a716-446655440000", PrimaryKeyUUID); err != nil {
		t.Errorf("expected valid UUID primary key, got error: %v", err)
	}

	if err := IsValidPrimaryKey("not-a-uuid", PrimaryKeyUUID); err == nil {
		t.Error("expected error for invalid UUID primary key")
	}

	if err := IsValidPrimaryKey("12345", PrimaryKeyInt); err != nil {
		t.Errorf("expected valid int primary key, got error: %v", err)
	}

	if err := IsValidPrimaryKey("not-an-int", PrimaryKeyInt); err == nil {
		t.Error("expected error for invalid int primary key")
	}
}

func TestIsValidInt(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"-456", true},
		{"", false},
		{"   ", false},
		{"12.34", false},
		{"abc", false},
	}

	for _, tt := range tests {
		result := IsValidInt(tt.value)
		if result != tt.expected {
			t.Errorf("IsValidInt(%q) = %v, want %v", tt.value, result, tt.expected)
		}
	}
}

func TestNewUUID(t *testing.T) {
	uuid, err := NewUUID()
	if err != nil {
		t.Fatalf("NewUUID() failed: %v", err)
	}

	str := uuid.String()
	if len(str) != 36 {
		t.Errorf("expected UUID string length 36, got %d", len(str))
	}
}

func TestNewULID(t *testing.T) {
	ulid, err := NewULID()
	if err != nil {
		t.Fatalf("NewULID() failed: %v", err)
	}

	str := ulid.String()
	if str == "" {
		t.Error("expected non-empty ULID string")
	}
}

func TestNewNanoID(t *testing.T) {
	id, err := NewNanoID(12)
	if err != nil {
		t.Fatalf("NewNanoID() failed: %v", err)
	}

	if len(id) != 12 {
		t.Errorf("expected NanoID length 12, got %d", len(id))
	}
}

func TestNewNanoIDInvalidSize(t *testing.T) {
	_, err := NewNanoID(0)
	if err == nil {
		t.Error("expected error for size 0")
	}

	_, err = NewNanoID(-1)
	if err == nil {
		t.Error("expected error for negative size")
	}
}

func TestFactory(t *testing.T) {
	f := NewFactory(16)

	nanoID := f.Make()
	if len(nanoID) != 16 {
		t.Errorf("expected NanoID length 16, got %d", len(nanoID))
	}

	uuid := f.MakeUUID()
	if uuid == "" {
		t.Error("expected non-empty UUID string")
	}

	ulid := f.MakeULID()
	if ulid == "" {
		t.Error("expected non-empty ULID string")
	}
}
