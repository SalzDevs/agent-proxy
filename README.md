# Groxy

[![Go Reference](https://pkg.go.dev/badge/github.com/SalzDevs/groxy.svg)](https://pkg.go.dev/github.com/SalzDevs/groxy)
[![CI](https://github.com/SalzDevs/groxy/actions/workflows/ci.yml/badge.svg)](https://github.com/SalzDevs/groxy/actions/workflows/ci.yml)

Groxy is a small Go library for building forward proxy servers.

> **Status:** Groxy is currently pre-v1. The API is usable, but breaking changes
> may still happen before a stable `v1.0.0` release. See the
> [roadmap](ROADMAP.md) for planned work.

It supports:

- HTTP request forwarding
- HTTPS tunneling with `CONNECT`
- opt-in HTTPS inspection with local TLS interception
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
if err := proxy.Use(
	groxy.AddRequestHeader("X-Groxy-Request", "true"),
	groxy.AddResponseHeader("X-Groxy-Response", "true"),
); err != nil {
	log.Fatal(err)
}
```

You can also use hooks directly:

```go
if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	ctx.Request.Header.Set("X-From-Groxy", "true")
	return nil
}); err != nil {
	log.Fatal(err)
}
```

Named functions work too:

```go
func logRequest(ctx *groxy.RequestContext) error {
	log.Printf("request: %s %s", ctx.Request.Method, ctx.Request.URL.String())
	return nil
}

if err := proxy.OnRequest(logRequest); err != nil {
	log.Fatal(err)
}
```

## Blocking traffic

Use `groxy.Block` inside hooks:

```go
if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	if ctx.Request.URL.Hostname() == "blocked.example" {
		return groxy.Block(403, "blocked by policy")
	}

	return nil
}); err != nil {
	log.Fatal(err)
}
```

Or use built-in helpers:

```go
if err := proxy.Use(
	groxy.BlockHost("blocked.example", 403, "blocked by groxy"),
	groxy.BlockConnectHost("blocked.example", 403, "CONNECT blocked by groxy"),
); err != nil {
	log.Fatal(err)
}
```

## Body transforms

Groxy can transform HTTP request and response bodies.

```go
if err := proxy.Use(groxy.TransformRequestBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("secret"), []byte("[redacted]")), nil
})); err != nil {
	log.Fatal(err)
}
```

```go
if err := proxy.Use(groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
})); err != nil {
	log.Fatal(err)
}
```

Body helpers and body transform middleware buffer the full body in memory. Groxy
limits how much data they can read with `Config.MaxBodySize`.

```go
proxy, err := groxy.New(groxy.Config{
	Addr:        "127.0.0.1:8080",
	MaxBodySize: 5 << 20, // 5 MiB
})
```

If `MaxBodySize` is zero, Groxy uses `DefaultMaxBodySize`.

By default, HTTPS traffic uses CONNECT tunneling. Encrypted HTTPS bodies can only
be inspected or transformed when HTTPS inspection is explicitly enabled.

## HTTPS inspection

Groxy can inspect selected HTTPS traffic using local TLS interception/MITM. This
is **opt-in only**. Without this config, HTTPS traffic is tunneled normally and
Groxy cannot read encrypted request or response bodies.

> Only inspect traffic you own or are authorized to inspect. Users must install
> and trust your Groxy CA certificate in their browser or operating system.

```go
ca, err := groxy.LoadCAFiles("groxy-ca.pem", "groxy-ca-key.pem")
if err != nil {
	ca, err = groxy.NewCA(groxy.CAConfig{
		CommonName: "Groxy Local CA",
		ValidFor:  365 * 24 * time.Hour,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := ca.WriteFiles("groxy-ca.pem", "groxy-ca-key.pem"); err != nil {
		log.Fatal(err)
	}
}

proxy, err := groxy.New(groxy.Config{
	Addr: "127.0.0.1:8080",
	HTTPSInspection: &groxy.HTTPSInspectionConfig{
		CA:        ca,
		Intercept: groxy.MatchHosts("example.com", "*.example.com"),
	},
})
```

After enabling inspection, normal middleware works on matched HTTPS traffic:

```go
if err := proxy.Use(groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
})); err != nil {
	log.Fatal(err)
}
```

Host matching helpers:

```go
groxy.MatchHosts("example.com", "*.example.org")
groxy.MatchAllHosts() // explicitly inspect every CONNECT host
```

Current HTTPS inspection limitations:

- intercepted client traffic is HTTP/1.1 over TLS
- users must trust the generated CA manually
- generated per-host certificates are kept in memory and renewed before expiry

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
- [`examples/https-inspection`](examples/https-inspection)

## Roadmap

See [`ROADMAP.md`](ROADMAP.md) for planned work and good first issue ideas.

## Contributing

Contributions are welcome. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for setup,
testing, and pull request guidelines.

## Security

Please do not report security vulnerabilities in public issues. See
[`SECURITY.md`](SECURITY.md) for responsible disclosure guidance.

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md) for release history.

## Development

Run tests:

```bash
go test ./...
```

Run race tests:

```bash
go test -race ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem ./...
```

Benchmarks cover HTTP forwarding, middleware overhead, body transforms, blocking,
and CONNECT tunneling. Results depend on your machine, Go version, OS, and
network environment, so treat them as local performance baselines rather than
universal numbers.

Run vet:

```bash
go vet ./...
```

## License

Groxy is released under the [MIT License](LICENSE).

## Current limitations

- HTTPS traffic is tunneled by default; inspection requires explicit HTTPS inspection config and a trusted local CA.
- Body transforms buffer the full body in memory.
- HTTPS inspection currently targets HTTP/1.1 over TLS.
- No authentication helpers yet.
- No metrics/observability helpers yet.
