package groxy

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCA_CreatesValidCA(t *testing.T) {
	ca, err := NewCA(CAConfig{
		CommonName: "Test Groxy CA",
		ValidFor:   time.Hour,
	})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}
	if ca.cert == nil {
		t.Fatal("expected CA certificate")
	}
	if ca.key == nil {
		t.Fatal("expected CA private key")
	}
	if !ca.cert.IsCA {
		t.Fatal("expected generated certificate to be a CA")
	}
	if ca.cert.Subject.CommonName != "Test Groxy CA" {
		t.Fatalf("common name = %q, want %q", ca.cert.Subject.CommonName, "Test Groxy CA")
	}
	if ca.cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("expected generated CA to allow certificate signing")
	}
}

func TestNewCA_AppliesDefaults(t *testing.T) {
	ca, err := NewCA(CAConfig{})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}
	if ca.cert.Subject.CommonName != defaultCACommonName {
		t.Fatalf("common name = %q, want %q", ca.cert.Subject.CommonName, defaultCACommonName)
	}
}

func TestNewCA_RejectsNegativeValidFor(t *testing.T) {
	if _, err := NewCA(CAConfig{ValidFor: -time.Second}); err == nil {
		t.Fatal("expected error for negative validity duration, got nil")
	}
}

func TestCA_WriteFilesAndLoadCAFiles(t *testing.T) {
	ca, err := NewCA(CAConfig{CommonName: "Write Load Test CA"})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	dir := t.TempDir()
	certFile := filepath.Join(dir, "ca.pem")
	keyFile := filepath.Join(dir, "ca-key.pem")

	if err := ca.WriteFiles(certFile, keyFile); err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	loaded, err := LoadCAFiles(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadCAFiles() error = %v", err)
	}
	if !loaded.cert.Equal(ca.cert) {
		t.Fatal("loaded CA certificate does not match written certificate")
	}
	if loaded.key.N.Cmp(ca.key.N) != 0 {
		t.Fatal("loaded CA key does not match written key")
	}
}

func TestLoadCAFiles_ReturnsErrorForInvalidCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "ca.pem")
	keyFile := filepath.Join(dir, "ca-key.pem")

	if err := os.WriteFile(certFile, []byte("not pem"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(keyFile, []byte("not pem"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadCAFiles(certFile, keyFile); err == nil {
		t.Fatal("expected error for invalid certificate, got nil")
	}
}

func TestCA_WriteFilesRejectsNilCA(t *testing.T) {
	var ca *CA
	if err := ca.WriteFiles("ca.pem", "ca-key.pem"); err == nil {
		t.Fatal("expected error for nil CA, got nil")
	}
}
