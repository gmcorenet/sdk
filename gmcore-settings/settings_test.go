package gmcore_settings

import (
	"context"
	"path/filepath"
	"testing"
)

type testEncryptor struct{}

func (testEncryptor) Encrypt(value string) (string, error) {
	return "enc:" + value, nil
}

func (testEncryptor) Decrypt(value string) (string, error) {
	return value[len("enc:"):], nil
}

func TestStoreEncryptsAndDecryptsValues(t *testing.T) {
	store, err := OpenWithConfig(context.Background(), Config{
		DSN:       filepath.Join(t.TempDir(), "settings.sqlite"),
		Encryptor: testEncryptor{},
	})
	if err != nil {
		t.Fatalf("open settings: %v", err)
	}
	if err := store.SetWithOptions(context.Background(), "secret.key", "value", "secret", "desc", false, true); err != nil {
		t.Fatalf("set encrypted value: %v", err)
	}
	current, ok := store.Get("secret.key")
	if !ok {
		t.Fatal("expected secret.key to exist")
	}
	if current.Value != "value" {
		t.Fatalf("unexpected decrypted value: %q", current.Value)
	}
	if !current.Encrypted {
		t.Fatal("expected encrypted flag")
	}
}

func TestSeedWithOptionsMigratesExistingValueToEncrypted(t *testing.T) {
	store, err := OpenWithConfig(context.Background(), Config{
		DSN:       filepath.Join(t.TempDir(), "settings.sqlite"),
		Encryptor: testEncryptor{},
	})
	if err != nil {
		t.Fatalf("open settings: %v", err)
	}
	if err := store.SetWithOptions(context.Background(), "security.api_secret", "plain", "secret", "desc", false, false); err != nil {
		t.Fatalf("seed plain value: %v", err)
	}
	if err := store.SeedWithOptions(context.Background(), "security.api_secret", "ignored", "secret", "desc", false, true); err != nil {
		t.Fatalf("migrate seed to encrypted: %v", err)
	}
	current, ok := store.Get("security.api_secret")
	if !ok {
		t.Fatal("expected migrated setting")
	}
	if current.Value != "plain" {
		t.Fatalf("unexpected migrated value: %q", current.Value)
	}
	if !current.Encrypted {
		t.Fatal("expected migrated encrypted flag")
	}
}
