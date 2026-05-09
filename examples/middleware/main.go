package main

import (
	"log"
	"os"

	"github.com/SalzDevs/groxy"
)

func main() {
	// Groxy is silent by default. Passing a logger lets you see what the proxy is doing.
	logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

	proxy, err := groxy.New(groxy.Config{
		Addr:   "127.0.0.1:8080",
		Logger: logger,
	})
	if err != nil {
		log.Fatalf("failed to create proxy: %v", err)
	}

	// Use installs reusable middleware.
	// Middleware can inspect, modify, or block traffic flowing through the proxy.
	if err := proxy.Use(
		// Add a header to normal HTTP requests before they are sent upstream.
		groxy.AddRequestHeader("X-Groxy-Request", "true"),

		// Add a header to normal HTTP responses before they are sent back to the client.
		groxy.AddResponseHeader("X-Groxy-Response", "true"),

		// Block normal HTTP requests to this host.
		groxy.BlockHost("blocked.example", 403, "blocked by groxy"),

		// Block HTTPS tunnels to this host.
		// HTTPS uses CONNECT, so it has a separate helper.
		groxy.BlockConnectHost("blocked.example", 403, "CONNECT blocked by groxy"),
	); err != nil {
		log.Fatalf("failed to add middleware: %v", err)
	}

	// Hooks can also be regular named functions.
	// This is useful when hook logic grows beyond a few lines.
	if err := proxy.OnRequest(logRequest); err != nil {
		log.Fatalf("failed to add request hook: %v", err)
	}
	if err := proxy.OnResponse(logResponse); err != nil {
		log.Fatalf("failed to add response hook: %v", err)
	}
	if err := proxy.OnConnect(logConnect); err != nil {
		log.Fatalf("failed to add connect hook: %v", err)
	}

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}

// logRequest runs before a normal HTTP request is sent upstream.
func logRequest(ctx *groxy.RequestContext) error {
	log.Printf("request: %s %s", ctx.Request.Method, ctx.Request.URL.String())
	return nil
}

// logResponse runs after the upstream HTTP response is received, but before it
// is sent back to the client.
func logResponse(ctx *groxy.ResponseContext) error {
	log.Printf("response: %s -> %d", ctx.Request.URL.String(), ctx.Response.StatusCode)
	return nil
}

// logConnect runs before Groxy opens an HTTPS CONNECT tunnel.
// The encrypted HTTPS request/response body is not visible in this hook.
func logConnect(ctx *groxy.ConnectContext) error {
	log.Printf("connect: %s", ctx.Host)
	return nil
}
