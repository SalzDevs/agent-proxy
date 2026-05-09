package main

import (
	"log"

	"github.com/SalzDevs/groxy"
)

func main() {
	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	proxy.OnRequest(func(ctx *groxy.RequestContext) error {
		log.Printf("request hook: %s", ctx.Request.URL.String())

		ctx.Request.Header.Set("X-Groxy-Request", "true")

		return nil
	})

	if err := proxy.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
}
