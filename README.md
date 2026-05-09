# Groxy

Groxy is a small Go library for building forward proxy servers.

It supports:

- HTTP request forwarding
- HTTPS tunneling with `CONNECT`
- middleware hooks for requests, responses, and CONNECT tunnels
- request/response blocking
- header helpers
- request/response body transforms
- configurable timeouts
- configurable logging

## Install

```bash
go get github.com/SalzDevs/groxy
```

## Basic usage

```go
package main

import (
	"log"

	"github.com/SalzDevs/groxy"
)

func main() {
	proxy, err := groxy.New(groxy.Config{
		Addr: "127.0.0.1:8080",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("proxy listening on %s", proxy.Addr())
	if err := proxy.Start(); err != nil {
		log.Fatal(err)
	}
}
```

Test it with:

```bash
curl -x http://127.0.0.1:8080 http://example.com
curl -x http://127.0.0.1:8080 https://example.com
```

## Middleware

Groxy middleware can inspect, modify, or block traffic.

```go
proxy.Use(
	groxy.AddRequestHeader("X-Groxy-Request", "true"),
	groxy.AddResponseHeader("X-Groxy-Response", "true"),
)
```

You can also use hooks directly:

```go
proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	ctx.Request.Header.Set("X-From-Groxy", "true")
	return nil
})
```

Named functions work too:

```go
func logRequest(ctx *groxy.RequestContext) error {
	log.Printf("request: %s %s", ctx.Request.Method, ctx.Request.URL.String())
	return nil
}

proxy.OnRequest(logRequest)
```

## Blocking traffic

Use `groxy.Block` inside hooks:

```go
proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	if ctx.Request.URL.Hostname() == "blocked.example" {
		return groxy.Block(403, "blocked by policy")
	}

	return nil
})
```

Or use built-in helpers:

```go
proxy.Use(
	groxy.BlockHost("blocked.example", 403, "blocked by groxy"),
	groxy.BlockConnectHost("blocked.example", 403, "CONNECT blocked by groxy"),
)
```

## Body transforms

Groxy can transform HTTP request and response bodies.

```go
proxy.Use(groxy.TransformRequestBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("secret"), []byte("[redacted]")), nil
}))
```

```go
proxy.Use(groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
}))
```

Note: HTTPS traffic uses CONNECT tunneling. Groxy cannot inspect or transform HTTPS request/response bodies unless TLS interception is added in the future.

## Timeouts

If no timeouts are provided, Groxy uses safe defaults.

```go
proxy, err := groxy.New(groxy.Config{
	Addr: "127.0.0.1:8080",
})
```

You can override only the values you care about:

```go
timeouts := groxy.DefaultTimeouts()
timeouts.Dial = 2 * time.Second

proxy, err := groxy.New(groxy.Config{
	Addr:     "127.0.0.1:8080",
	Timeouts: &timeouts,
})
```

## Logging

Groxy is silent by default. Pass a logger if you want logs:

```go
logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

proxy, err := groxy.New(groxy.Config{
	Addr:   "127.0.0.1:8080",
	Logger: logger,
})
```

## Examples

See:

- [`examples/basic`](examples/basic)
- [`examples/middleware`](examples/middleware)
- [`examples/body-transform`](examples/body-transform)

## Development

Run tests:

```bash
go test ./...
```

Run race tests:

```bash
go test -race ./...
```

Run vet:

```bash
go vet ./...
```

## Current limitations

- HTTPS traffic is tunneled, not decrypted.
- Body transforms buffer the full body in memory.
- No TLS interception/MITM support.
- No authentication helpers yet.
- No metrics/observability helpers yet.
