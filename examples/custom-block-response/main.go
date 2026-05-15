package main

import (
	"encoding/json"
	"log"
	"net/http"
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

	if err := proxy.Use(
		groxy.AccessLog(logger),
		blockMetadataService(),
		blockPrivateConnectHost(),
		jsonUpstreamErrors(),
	); err != nil {
		log.Fatalf("failed to add middleware: %v", err)
	}

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatalf("proxy stopped with error: %v", err)
	}
}

func blockMetadataService() groxy.Middleware {
	return groxy.OnRequest(func(ctx *groxy.RequestContext) error {
		if ctx.Request.URL.Hostname() != "169.254.169.254" {
			return nil
		}

		// Block returns a status code and body to the client.
		// Groxy writes block responses with net/http's default text error format.
		return groxy.Block(http.StatusForbidden, jsonError("metadata_blocked", "metadata service is not reachable through this proxy"))
	})
}

func blockPrivateConnectHost() groxy.Middleware {
	return groxy.OnConnect(func(ctx *groxy.ConnectContext) error {
		if ctx.Host != "private.example:443" {
			return nil
		}

		return groxy.Block(http.StatusForbidden, "CONNECT to private.example is blocked by policy")
	})
}

func jsonUpstreamErrors() groxy.Middleware {
	return groxy.OnResponse(func(ctx *groxy.ResponseContext) error {
		if ctx.Response.StatusCode < 500 {
			return nil
		}

		body := []byte(jsonError("upstream_unavailable", "upstream returned an error") + "\n")

		ctx.Response.StatusCode = http.StatusBadGateway
		ctx.Response.Status = "502 Bad Gateway"
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.Header.Del("Content-Encoding")
		ctx.SetBody(body)

		return nil
	})
}

func jsonError(code, message string) string {
	body, err := json.Marshal(map[string]string{
		"error":   code,
		"message": message,
	})
	if err != nil {
		return `{"error":"internal_error","message":"failed to build error response"}`
	}

	return string(body)
}
