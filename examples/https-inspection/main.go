package main

import (
	"bytes"
	"log"
	"os"
	"time"

	"github.com/SalzDevs/groxy"
)

func main() {
	ca, err := loadOrCreateCA("groxy-ca.pem", "groxy-ca-key.pem")
	if err != nil {
		log.Fatal(err)
	}

	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &groxy.HTTPSInspectionConfig{
			CA: ca,

			// Be explicit about which hosts you inspect. Use MatchAllHosts only
			// when you intentionally want to inspect every CONNECT host.
			Intercept: groxy.MatchHosts("example.com", "*.example.com"),

			// Optional. If zero, Groxy uses a safe default and renews generated
			// host certificates before they expire.
			CertificateTTL: 24 * time.Hour,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := proxy.Use(groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
		return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
	})); err != nil {
		log.Fatal(err)
	}

	log.Println("HTTPS inspection proxy listening on", proxy.Addr())
	log.Println("Install/trust groxy-ca.pem in your browser or OS before testing.")
	log.Fatal(proxy.Start())
}

func loadOrCreateCA(certFile, keyFile string) (*groxy.CA, error) {
	ca, err := groxy.LoadCAFiles(certFile, keyFile)
	if err == nil {
		return ca, nil
	}
	if !os.IsNotExist(err) {
		log.Printf("could not load existing CA, creating a new one: %v", err)
	}

	ca, err = groxy.NewCA(groxy.CAConfig{
		CommonName: "Groxy Local CA",
		ValidFor:   365 * 24 * time.Hour,
	})
	if err != nil {
		return nil, err
	}

	if err := ca.WriteFiles(certFile, keyFile); err != nil {
		return nil, err
	}

	return ca, nil
}
