package gmcore_encryption

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		svc, err := New("abcdefghijklmnopqrstuvwxyz123456")
		if err != nil {
			t.Fatalf("New should succeed for 32-byte key: %v", err)
		}
		if svc == nil {
			t.Fatal("New should return non-nil service")
		}
	})

	t.Run("key too short", func(t *testing.T) {
		_, err := New("short")
		if err == nil {
			t.Fatal("New should fail for short key")
		}
	})

	t.Run("key too long", func(t *testing.T) {
		_, err := New(strings.Repeat("x", 64))
		if err == nil {
			t.Fatal("New should fail for long key")
		}
	})

	t.Run("key with whitespace (exact 32 after trim)", func(t *testing.T) {
		svc, err := New("  abcdefghijklmnopqrstuvwxyz123456  ")
		if err != nil {
			t.Fatalf("New should succeed after trimming: %v", err)
		}
		if svc == nil {
			t.Fatal("service should be non-nil")
		}
	})
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	svc, err := New("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	tests := []string{
		"hello world",
		"",
		"special chars: !@#$%^&*()",
		"unicode: こんにちは",
		strings.Repeat("a", 1000),
	}

	for _, plain := range tests {
		encrypted, err := svc.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt(%q) failed: %v", plain, err)
		}
		if encrypted == "" {
			t.Fatalf("Encrypt(%q) returned empty string", plain)
		}
		if encrypted == plain {
			t.Fatal("encrypted text should differ from plaintext")
		}

		decrypted, err := svc.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt(%q) failed: %v", encrypted, err)
		}
		if decrypted != plain {
			t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plain)
		}
	}
}

func TestDecrypt_InvalidInput(t *testing.T) {
	svc, err := New("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	_, err = svc.Decrypt("not-valid-base64!!!")
	if err == nil {
		t.Fatal("Decrypt should fail on invalid base64")
	}

	_, err = svc.Decrypt("")
	if err == nil {
		t.Fatal("Decrypt should fail on empty input")
	}
}

func TestEncrypt_WrongKey(t *testing.T) {
	svc1, _ := New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	svc2, _ := New("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	encrypted, err := svc1.Encrypt("secret message")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = svc2.Decrypt(encrypted)
	if err == nil {
		t.Fatal("Decrypt with wrong key should fail")
	}
}
