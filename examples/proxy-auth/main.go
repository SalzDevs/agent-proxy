package main

import (
	"log"
	"os"

	"github.com/SalzDevs/groxy"
)

func main() {
	password := os.Getenv("GROXY_PROXY_PASSWORD")
	if password == "" {
		log.Fatal("set GROXY_PROXY_PASSWORD before starting the proxy")
	}

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	if err := proxy.Use(groxy.ProxyBasicAuth("admin", password)); err != nil {
		log.Fatalf("failed to add proxy auth middleware: %v", err)
	}

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}
}
