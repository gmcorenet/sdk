package gmcore_cert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCertificateManager_Generate(t *testing.T) {
	m := NewManager()

	cert, err := m.Generate("localhost", 365)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if cert.Subject.CommonName != "localhost" {
		t.Errorf("expected CN=localhost, got %s", cert.Subject.CommonName)
	}

	if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "GMCore" {
		t.Errorf("expected Organization=GMCore, got %v", cert.Subject.Organization)
	}
}

func TestCertificateManager_AddGetRemove(t *testing.T) {
	m := NewManager()

	cert1, _ := m.Generate("host1", 30)
	cert2, _ := m.Generate("host2", 60)

	m.Add("cert1", cert1)
	m.Add("cert2", cert2)

	if m.Get("cert1") != cert1 {
		t.Error("Get(cert1) should return cert1")
	}

	if m.Get("cert2") != cert2 {
		t.Error("Get(cert2) should return cert2")
	}

	if m.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) should return nil")
	}

	m.Remove("cert1")
	if m.Get("cert1") != nil {
		t.Error("after Remove, Get(cert1) should return nil")
	}
}

func TestCertificate_EncodeCertificate(t *testing.T) {
	m := NewManager()
	cert, _ := m.Generate("localhost", 365)

	pem, err := cert.EncodeCertificate()
	if err != nil {
		t.Fatalf("EncodeCertificate failed: %v", err)
	}

	if pem == "" {
		t.Error("encoded certificate should not be empty")
	}
}

func TestCertificate_EncodePrivateKey(t *testing.T) {
	m := NewManager()
	cert, _ := m.Generate("localhost", 365)

	pem, err := cert.EncodePrivateKey()
	if err != nil {
		t.Fatalf("EncodePrivateKey failed: %v", err)
	}

	if pem == "" {
		t.Error("encoded private key should not be empty")
	}
}

func TestCertificate_SaveToFile(t *testing.T) {
	m := NewManager()
	cert, _ := m.Generate("localhost", 365)

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	err := cert.SaveToFile(certFile, keyFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		t.Error("cert file should exist")
	}

	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		t.Error("key file should exist")
	}
}

func TestLoadCertificateFromFile(t *testing.T) {
	m := NewManager()
	cert, _ := m.Generate("localhost", 365)

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	err := cert.SaveToFile(certFile, keyFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	loaded, err := LoadCertificateFromFile(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadCertificateFromFile failed: %v", err)
	}

	if loaded.Subject.CommonName != "localhost" {
		t.Errorf("loaded cert CN should be localhost, got %s", loaded.Subject.CommonName)
	}
}