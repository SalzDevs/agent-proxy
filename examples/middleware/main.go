package main

import (
	"log"
	"os"

	"github.com/SalzDevs/groxy"
)

func main() {
	logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

	proxy, err := groxy.New(groxy.Config{
		Addr:   "127.0.0.1:8080",
		Logger: logger,
	})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	proxy.Use(
		groxy.AddRequestHeader("X-Groxy-Request", "true"),
		groxy.AddResponseHeader("X-Groxy-Response", "true"),
		groxy.BlockHost("blocked.example", 403, "blocked by groxy"),
		groxy.BlockConnectHost("blocked.example", 403, "CONNECT blocked by groxy"),
	)

	proxy.OnRequest(logRequest)
	proxy.OnResponse(logResponse)
	proxy.OnConnect(logConnect)

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}

func logRequest(ctx *groxy.RequestContext) error {
	log.Printf("request: %s %s", ctx.Request.Method, ctx.Request.URL.String())
	return nil
}

func logResponse(ctx *groxy.ResponseContext) error {
	log.Printf("response: %s -> %d", ctx.Request.URL.String(), ctx.Response.StatusCode)
	return nil
}

func logConnect(ctx *groxy.ConnectContext) error {
	log.Printf("connect: %s", ctx.Host)
	return nil
}
