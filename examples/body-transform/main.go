package main

import (
	"bytes"
	"log"

	"github.com/SalzDevs/groxy"
)

func main() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	proxy.Use(
		groxy.TransformRequestBody(func(body []byte) ([]byte, error) {
			return bytes.ReplaceAll(body, []byte("secret"), []byte("[redacted]")), nil
		}),
		groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
			return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
		}),
	)

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}
