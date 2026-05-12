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
- access log middleware
- configurable timeouts
- configurable logging

## Install

```bash
go get github.com/SalzDevs/groxy
```

## Try it in 60 seconds

Run the demo proxy:

```bash
git clone https://github.com/SalzDevs/groxy.git
cd groxy
go run ./cmd/groxy
```

In another terminal, send HTTP and HTTPS requests through it:

```bash
curl -x http://127.0.0.1:8080 http://example.com
curl -x http://127.0.0.1:8080 https://example.com
```

You should see requests pass through the local proxy. By default, HTTPS uses
normal CONNECT tunneling, so encrypted HTTPS bodies are not inspected unless you
explicitly enable HTTPS inspection.

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
is **opt-in only**. Without HTTPS inspection config, Groxy tunnels HTTPS normally
and cannot read encrypted request or response bodies.

> Only inspect traffic you own or are authorized to inspect. Users must install
> and trust your Groxy CA certificate in their browser or operating system.

```go
ca, err := groxy.LoadCAFiles("groxy-ca.pem", "groxy-ca-key.pem")
if err != nil {
	log.Fatal(err)
}

proxy, err := groxy.New(groxy.Config{
	Addr: "127.0.0.1:8080",
	HTTPSInspection: &groxy.HTTPSInspectionConfig{
		CA:        ca,
		Intercept: groxy.MatchHosts("example.com", "*.example.com"),
	},
})
```

After enabling inspection, normal middleware works on matched HTTPS traffic.
Use `MatchHosts` for narrow allowlists and `MatchAllHosts` only when you
explicitly want broad inspection.

Detailed docs:

- [HTTPS inspection guide](docs/https-inspection.md)
- [HTTPS inspection threat model](docs/https-inspection-threat-model.md)

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

Groxy is silent by default. Pass a logger if you want internal proxy logs:

```go
logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

proxy, err := groxy.New(groxy.Config{
	Addr:   "127.0.0.1:8080",
	Logger: logger,
})
```

Use `AccessLog` when you want one-line traffic logs for HTTP requests and
CONNECT tunnels:

```go
if err := proxy.Use(groxy.AccessLog(logger)); err != nil {
	log.Fatal(err)
}
```

## Examples and guides

- [Documentation index](docs/README.md)
- [Runnable examples](examples/README.md)
- [`examples/access-log`](examples/access-log)
- [Building a forward proxy in Go with Groxy](docs/building-forward-proxy.md)
- [HTTPS inspection guide](docs/https-inspection.md)
- [HTTPS inspection threat model](docs/https-inspection-threat-model.md)

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
