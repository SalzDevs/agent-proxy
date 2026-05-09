package main

import (
	"log"

	"github.com/SalzDevs/groxy"
)

func main() {
	// Create a new proxy that listens on localhost port 8080.
	// At this point the proxy is only configured; it is not running yet.
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	// Start blocks while the proxy is running.
	// Test it with:
	//
	//   curl -x http://127.0.0.1:8080 http://example.com
	//   curl -x http://127.0.0.1:8080 https://example.com
	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}
