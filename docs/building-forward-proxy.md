# Building a forward proxy in Go with Groxy

This guide walks through a small forward proxy using Groxy.

A forward proxy accepts requests from a client and forwards them to upstream
servers. Clients opt into using it, for example with `curl -x` or browser proxy
settings.

Groxy handles the proxy plumbing and gives you middleware hooks for requests,
responses, and CONNECT tunnels.

## 1. Start with a minimal proxy

Create `main.go`:

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

Run it:

```bash
go mod init example.com/groxy-demo
go get github.com/SalzDevs/groxy
go run .
```

In another terminal:

```bash
curl -x http://127.0.0.1:8080 http://example.com
curl -x http://127.0.0.1:8080 https://example.com
```

HTTP requests are forwarded directly. HTTPS requests use CONNECT tunneling by
default, so the proxy opens a TCP tunnel and does not read encrypted HTTPS
request or response bodies.

## 2. Add request and response middleware

Middleware is registered before `Start`:

```go
if err := proxy.Use(
	groxy.AddRequestHeader("X-From-Groxy", "true"),
	groxy.AddResponseHeader("X-Proxied-By", "groxy"),
); err != nil {
	log.Fatal(err)
}
```

You can also write hooks directly:

```go
if err := proxy.OnRequest(func(ctx *groxy.RequestContext) error {
	log.Printf("request: %s %s", ctx.Request.Method, ctx.Request.URL.String())
	return nil
}); err != nil {
	log.Fatal(err)
}
```

## 3. Add access logs

Use `AccessLog` to write one-line traffic logs for HTTP requests and CONNECT
tunnels:

```go
logger := log.New(os.Stdout, "groxy: ", log.LstdFlags)

if err := proxy.Use(groxy.AccessLog(logger)); err != nil {
	log.Fatal(err)
}
```

HTTP requests are logged when they are sent upstream and when they finish with a
response, block, or forwarding error. CONNECT tunnels are logged when the
CONNECT hook runs.

## 4. Block traffic

Request hooks can return `groxy.Block` to stop a request with a specific status
code and message:

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

For common cases, use the built-in helpers:

```go
if err := proxy.Use(
	groxy.BlockHost("blocked.example", 403, "blocked by groxy"),
	groxy.BlockConnectHost("blocked.example", 403, "CONNECT blocked by groxy"),
); err != nil {
	log.Fatal(err)
}
```

## 5. Transform bodies

Body transforms buffer the full body in memory. Groxy protects these helpers with
`Config.MaxBodySize`.

```go
proxy, err := groxy.New(groxy.Config{
	Addr:        "127.0.0.1:8080",
	MaxBodySize: 5 << 20, // 5 MiB
})
if err != nil {
	log.Fatal(err)
}
```

Then register a transform:

```go
if err := proxy.Use(groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
	return bytes.ReplaceAll(body, []byte("Example Domain"), []byte("Groxy Domain")), nil
})); err != nil {
	log.Fatal(err)
}
```

Normal HTTPS traffic is encrypted, so response body transforms apply to plain
HTTP by default. To inspect selected HTTPS hosts, enable HTTPS inspection.

## 6. Inspect selected HTTPS hosts explicitly

HTTPS inspection uses a local certificate authority to generate per-host
certificates. This is powerful and sensitive, so Groxy keeps it explicit:

- HTTPS tunnels are the default.
- Inspection requires `HTTPSInspectionConfig`.
- Inspection requires both a CA and an `Intercept` host matcher.
- Clients must trust the generated CA certificate.
- Only inspect traffic you own or are authorized to inspect.

A minimal setup looks like this:

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
if err != nil {
	log.Fatal(err)
}
```

Install only `groxy-ca.pem` into the client trust store. Keep
`groxy-ca-key.pem` private.

After that, normal request/response middleware runs on matched HTTPS traffic.

For full CA trust setup, host matching notes, passthrough behavior, and safety
assumptions, read the [HTTPS inspection guide](https-inspection.md) and
[HTTPS inspection threat model](https-inspection-threat-model.md).

## Complete examples

See [the examples index](../examples/README.md) for runnable programs:

- [`examples/basic`](../examples/basic)
- [`examples/middleware`](../examples/middleware)
- [`examples/body-transform`](../examples/body-transform)
- [`examples/https-inspection`](../examples/https-inspection)

## Next steps

Useful production features usually include access logs, proxy authentication,
metrics, and stricter HTTPS inspection policy docs. Those are tracked in the
[roadmap](../ROADMAP.md) and GitHub issues.
