package groxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"sync"
	"time"
)

const hostCertificateValidFor = 24 * time.Hour

type certCache struct {
	mu    sync.Mutex
	ca    *CA
	certs map[string]*tls.Certificate
}

func newCertCache(ca *CA) *certCache {
	return &certCache{
		ca:    ca,
		certs: make(map[string]*tls.Certificate),
	}
}

func (c *certCache) get(host string) (*tls.Certificate, error) {
	if c == nil || c.ca == nil || c.ca.cert == nil || c.ca.key == nil {
		return nil, fmt.Errorf("certificate cache CA is not initialized")
	}

	host = normalizeHost(host)
	if host == "" {
		return nil, fmt.Errorf("certificate host is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cert := c.certs[host]; cert != nil {
		return cert, nil
	}

	cert, err := c.generate(host)
	if err != nil {
		return nil, err
	}
	c.certs[host] = cert

	return cert, nil
}

func (c *certCache) generate(host string) (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate host private key: %w", err)
	}

	serial, err := randomSerialNumber()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(hostCertificateValidFor),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, c.ca.cert, &key.PublicKey, c.ca.key)
	if err != nil {
		return nil, fmt.Errorf("create host certificate: %w", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{der, c.ca.cert.Raw},
		PrivateKey:  key,
		Leaf:        template,
	}, nil
}
