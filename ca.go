package groxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

const (
	defaultCACommonName = "Groxy Local CA"
	defaultCAValidFor   = 365 * 24 * time.Hour
)

// CA is a certificate authority used to sign per-host certificates for HTTPS
// inspection.
type CA struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
	pem  []byte
}

// CAConfig configures local CA generation.
type CAConfig struct {
	// CommonName is the certificate common name. If empty, Groxy uses a default.
	CommonName string

	// ValidFor is how long the generated CA certificate is valid. If zero, Groxy
	// uses a default validity period.
	ValidFor time.Duration
}

// NewCA creates a new local certificate authority for HTTPS inspection.
func NewCA(config CAConfig) (*CA, error) {
	commonName := config.CommonName
	if commonName == "" {
		commonName = defaultCACommonName
	}

	validFor := config.ValidFor
	if validFor == 0 {
		validFor = defaultCAValidFor
	}
	if validFor < 0 {
		return nil, fmt.Errorf("CA validity duration cannot be negative")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate CA private key: %w", err)
	}

	serial, err := randomSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(validFor),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse generated CA certificate: %w", err)
	}

	return &CA{
		cert: cert,
		key:  key,
		pem:  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}, nil
}

// LoadCAFiles loads a CA certificate and RSA private key from PEM files.
func LoadCAFiles(certFile, keyFile string) (*CA, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate file: %w", err)
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read CA key file: %w", err)
	}

	cert, err := parseCACertificate(certPEM)
	if err != nil {
		return nil, err
	}

	key, err := parseRSAPrivateKey(keyPEM)
	if err != nil {
		return nil, err
	}

	return &CA{cert: cert, key: key, pem: certPEM}, nil
}

// WriteFiles writes the CA certificate and private key to PEM files.
func (ca *CA) WriteFiles(certFile, keyFile string) error {
	if ca == nil || ca.cert == nil || ca.key == nil {
		return fmt.Errorf("CA is not initialized")
	}

	if err := os.WriteFile(certFile, ca.pem, 0644); err != nil {
		return fmt.Errorf("write CA certificate file: %w", err)
	}

	keyDER := x509.MarshalPKCS1PrivateKey(ca.key)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("write CA key file: %w", err)
	}

	return nil
}

func parseCACertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("CA certificate PEM block not found")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, fmt.Errorf("certificate is not a CA")
	}

	return cert, nil
}

func parseRSAPrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("CA key PEM block not found")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA private key: %w", err)
	}

	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("CA private key must be RSA")
	}

	return rsaKey, nil
}

func randomSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return nil, fmt.Errorf("generate certificate serial number: %w", err)
	}

	return serial, nil
}
