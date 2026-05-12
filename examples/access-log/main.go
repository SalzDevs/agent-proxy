package main

import (
	"log"
	"os"

	"github.com/SalzDevs/groxy"
)

func main() {
	logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	if err := proxy.Use(groxy.AccessLog(logger)); err != nil {
		log.Fatalf("failed to add access log middleware: %v", err)
	}

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("failed to start proxy: %v", err)
	}
}
