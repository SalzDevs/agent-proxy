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

	// Body transform middleware reads the full body, lets you replace it, and then
	// puts the new body back so the proxy can continue forwarding it.
	//
	// This is useful for redaction, rewriting, filtering, or things like token
	// counting for AI prompts.
	if err := proxy.Use(
		// Transform the request body before it is sent upstream.
		// This example replaces the word "secret" with "[redacted]".
		groxy.TransformRequestBody(func(body []byte) ([]byte, error) {
			return bytes.ReplaceAll(body, []byte("secret"), []byte("[redacted]")), nil
		}),

		// Transform the response body before it is sent back to the client.
		// This example rewrites visible text from example.com.
		groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
			return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
		}),
	); err != nil {
		log.Fatalf("failed to add middleware: %v", err)
	}

	// Test the response transform with:
	//
	//   curl -x http://127.0.0.1:8080 http://example.com
	//
	// Note: HTTPS traffic uses CONNECT tunneling, so Groxy cannot transform HTTPS
	// bodies unless TLS interception is added in the future.
	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}
