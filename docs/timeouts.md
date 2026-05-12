# Timeout semantics

Groxy separates timeouts by connection direction:

- **client → proxy**: the application or browser talking to Groxy
- **proxy → upstream**: Groxy connecting to the destination server

This matters because a timeout such as `Dial` applies to Groxy's outbound
connection to an upstream server, not to how long a client may take to connect to
Groxy.

## Defaults and zero values

If `Config.Timeouts` is nil, Groxy uses `DefaultTimeouts()`.

```go
proxy, err := groxy.New(groxy.Config{
	Addr: "127.0.0.1:8080",
})
```

If `Config.Timeouts` is provided, any zero-valued field is filled with its
default value:

```go
timeouts := groxy.DefaultTimeouts()
timeouts.Dial = 2 * time.Second

proxy, err := groxy.New(groxy.Config{
	Addr:     "127.0.0.1:8080",
	Timeouts: &timeouts,
})
```

A zero duration means "use the default"; it does **not** disable that timeout.
Negative durations are rejected during `groxy.New` validation.

## Timeout fields

| Field | Direction | Applies to | Default |
| --- | --- | --- | --- |
| `Dial` | proxy → upstream | Maximum time for Groxy to establish a TCP connection to the upstream server. Used for normal HTTP forwarding and CONNECT target dialing. | `10s` |
| `TLSHandshake` | proxy → upstream | Maximum time for Groxy's outbound HTTP client to complete a TLS handshake with an upstream server. | `10s` |
| `ResponseHeader` | proxy → upstream | Maximum time to wait for upstream response headers after the request is sent. | `30s` |
| `IdleConn` | proxy → upstream | Maximum time an unused upstream keep-alive connection remains open in Groxy's HTTP transport. | `90s` |
| `ReadHeader` | client → proxy | Maximum time for a client to send request headers to Groxy's built-in server. | `5s` |
| `Idle` | client → proxy | Maximum time an idle client connection to Groxy's built-in server remains open. | `60s` |

## HTTP forwarding

For plain HTTP proxy requests, Groxy uses the upstream HTTP client configured in
`New`.

Relevant proxy → upstream timeouts:

- `Dial`
- `ResponseHeader`
- `IdleConn`

If the forwarded request is HTTPS because Groxy is inspecting HTTPS traffic, the
outbound upstream connection also uses `TLSHandshake`.

## CONNECT tunneling

For normal HTTPS tunneling, the client sends a CONNECT request and Groxy opens a
TCP connection to the requested target.

Relevant timeout:

- `Dial`: maximum time to connect from Groxy to the CONNECT target

After the tunnel is established, Groxy copies bytes between the client and the
upstream connection. `ResponseHeader` does not apply to a raw CONNECT tunnel
because Groxy is not parsing upstream HTTP responses inside the encrypted tunnel.

## HTTPS inspection

When HTTPS inspection is enabled for a host, Groxy terminates TLS from the client
and opens its own HTTPS connection to the upstream server.

Relevant proxy → upstream timeouts:

- `Dial`
- `TLSHandshake`
- `ResponseHeader`
- `IdleConn`

`TLSHandshake` controls Groxy's outbound TLS handshake to the upstream server. It
does not currently control the client → proxy TLS handshake performed during
HTTPS inspection.

## Built-in server vs custom server

`ReadHeader` and `Idle` are applied to Groxy's built-in `http.Server` when you
start the proxy with `proxy.Start()`.

If you mount Groxy as a handler on your own `http.Server` with `ServeHTTP`, your
custom server owns client → proxy server timeouts. Configure
`ReadHeaderTimeout`, `IdleTimeout`, and related fields on that server directly.

Proxy → upstream timeouts still come from `Config.Timeouts` because Groxy still
uses its own upstream HTTP client and CONNECT dialer.
