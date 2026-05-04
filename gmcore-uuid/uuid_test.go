package gmcore_uuid

import (
	"testing"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"  ", false},
		{"not-a-uuid", false},
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550e8400-e29b-41d4-a716-446655440000 ", true},
		{" 550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"550e8400-e29b-41d4-a716-44665544000", false},
		{"550e8400-e29b-41d4-a716-4466554400000", false},
		{"550e8400-e29b-41d4-a716-44665544000g", false},
	}

	for _, tt := range tests {
		got := IsValid(tt.input)
		if got != tt.expected {
			t.Errorf("IsValid(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestNew(t *testing.T) {
	u := New()
	if !IsValid(u) {
		t.Errorf("New() = %q, want valid UUID", u)
	}
}

func TestIsValidPrimaryKey(t *testing.T) {
	tests := []struct {
		key      string
		pkType   PrimaryKeyType
		wantErr  bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", PrimaryKeyUUID, false},
		{"invalid", PrimaryKeyUUID, true},
		{"123", PrimaryKeyInt, false},
		{"abc", PrimaryKeyInt, true},
		{"", PrimaryKeyUUID, true},
	}

	for _, tt := range tests {
		err := IsValidPrimaryKey(tt.key, tt.pkType)
		if (err != nil) != tt.wantErr {
			t.Errorf("IsValidPrimaryKey(%q, %s) error = %v, wantErr %v", tt.key, tt.pkType, err, tt.wantErr)
		}
	}
}

func TestIsValidInt(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"  ", false},
		{"0", true},
		{"123", true},
		{"-456", true},
		{"+789", true},
		{"12.34", false},
		{"abc", false},
		{"1a2", false},
	}

	for _, tt := range tests {
		got := IsValidInt(tt.input)
		if got != tt.expected {
			t.Errorf("IsValidInt(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
