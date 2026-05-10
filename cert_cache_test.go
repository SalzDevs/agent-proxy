package groxy

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestCertCache_GeneratesCertificateForHost(t *testing.T) {
	ca, err := NewCA(CAConfig{CommonName: "Test CA"})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 0)
	cert, err := cache.get("example.com:443")
	if err != nil {
		t.Fatalf("cache.get() error = %v", err)
	}
	if cert == nil {
		t.Fatal("expected certificate")
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	if leaf.Subject.CommonName != "example.com" {
		t.Fatalf("common name = %q, want %q", leaf.Subject.CommonName, "example.com")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != "example.com" {
		t.Fatalf("DNSNames = %v, want [example.com]", leaf.DNSNames)
	}
}

func TestCertCache_CertificateIsSignedByCA(t *testing.T) {
	ca, err := NewCA(CAConfig{CommonName: "Test CA"})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 0)
	cert, err := cache.get("example.com")
	if err != nil {
		t.Fatalf("cache.get() error = %v", err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)

	if _, err := leaf.Verify(x509.VerifyOptions{
		DNSName: "example.com",
		Roots:   roots,
	}); err != nil {
		t.Fatalf("certificate verification error = %v", err)
	}
}

func TestCertCache_ReusesCertificateForHost(t *testing.T) {
	ca, err := NewCA(CAConfig{})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 0)
	first, err := cache.get("EXAMPLE.COM:443")
	if err != nil {
		t.Fatalf("cache.get() first error = %v", err)
	}
	second, err := cache.get("example.com")
	if err != nil {
		t.Fatalf("cache.get() second error = %v", err)
	}

	if first != second {
		t.Fatal("expected cached certificate to be reused")
	}
}

func TestCertCache_GeneratesIPCertificate(t *testing.T) {
	ca, err := NewCA(CAConfig{})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 0)
	cert, err := cache.get("127.0.0.1:443")
	if err != nil {
		t.Fatalf("cache.get() error = %v", err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	if len(leaf.IPAddresses) != 1 || !leaf.IPAddresses[0].Equal([]byte{127, 0, 0, 1}) {
		t.Fatalf("IPAddresses = %v, want [127.0.0.1]", leaf.IPAddresses)
	}
}

func TestCertCache_RenewsCertificateBeforeExpiry(t *testing.T) {
	ca, err := NewCA(CAConfig{})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, time.Hour)
	first, err := cache.get("example.com")
	if err != nil {
		t.Fatalf("cache.get() first error = %v", err)
	}
	second, err := cache.get("example.com")
	if err != nil {
		t.Fatalf("cache.get() second error = %v", err)
	}

	if first == second {
		t.Fatal("expected certificate within renewal window to be regenerated")
	}
}

func TestCertCache_CapsCertificateExpiryAtCAExpiry(t *testing.T) {
	ca, err := NewCA(CAConfig{ValidFor: time.Hour})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 24*time.Hour)
	cert, err := cache.get("example.com")
	if err != nil {
		t.Fatalf("cache.get() error = %v", err)
	}

	if cert.Leaf.NotAfter.After(ca.cert.NotAfter) {
		t.Fatalf("cert expiry = %s, want not after CA expiry %s", cert.Leaf.NotAfter, ca.cert.NotAfter)
	}
}

func TestCertificateNeedsRenewal(t *testing.T) {
	now := time.Now()
	cert := &tls.Certificate{Leaf: &x509.Certificate{NotAfter: now.Add(2 * time.Hour)}}
	if certificateNeedsRenewal(cert, now) {
		t.Fatal("expected certificate outside renewal window not to need renewal")
	}

	cert.Leaf.NotAfter = now.Add(30 * time.Minute)
	if !certificateNeedsRenewal(cert, now) {
		t.Fatal("expected certificate inside renewal window to need renewal")
	}
}

func TestCertCache_RejectsEmptyHost(t *testing.T) {
	ca, err := NewCA(CAConfig{})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	cache := newCertCache(ca, 0)
	if _, err := cache.get(""); err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}
