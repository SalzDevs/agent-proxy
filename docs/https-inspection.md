# HTTPS inspection guide

Groxy can inspect selected HTTPS traffic using local TLS interception/MITM.
This is powerful, sensitive functionality and is **off by default**.

Without `HTTPSInspectionConfig`, Groxy handles HTTPS with normal CONNECT
tunneling. In that mode, Groxy forwards encrypted bytes and cannot read HTTPS
request or response bodies.

## Safety defaults

- HTTPS inspection is explicit opt-in.
- Inspection requires both a CA and an `Intercept` host matcher.
- Hosts that do not match `Intercept` are tunneled normally.
- Inspection setup failures fail closed by default.
- `PassthroughOnError` must be explicitly enabled to fall back to tunneling when
  inspection setup fails.
- Client applications must trust the Groxy CA certificate before inspected HTTPS
  connections will succeed.

Only inspect traffic you own or are authorized to inspect.

## Basic setup

Create or load a local CA, then choose which hosts to inspect:

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

After this, normal request and response middleware can run on matched HTTPS
traffic.

## Host matching

Use `MatchHosts` for exact and wildcard host patterns:

```go
groxy.MatchHosts("example.com", "*.example.com")
```

Notes:

- Matching is case-insensitive.
- Hosts may include ports.
- `*.example.com` matches subdomains such as `api.example.com`.
- `*.example.com` does **not** match `example.com`; include both if needed.

Use `MatchAllHosts()` only when you explicitly want broad inspection:

```go
groxy.MatchAllHosts()
```

For most applications, prefer a narrow allowlist with `MatchHosts`.

## Trusting the Groxy CA

`CA.WriteFiles("groxy-ca.pem", "groxy-ca-key.pem")` writes the public CA
certificate and private key separately.

- Install only `groxy-ca.pem` on client devices.
- Keep `groxy-ca-key.pem` private.
- Remove the CA from trust stores when you no longer need HTTPS inspection.

Common trust-store setup:

### Firefox

Settings → Privacy & Security → Certificates → View Certificates → Authorities
→ Import. Select `groxy-ca.pem`, then enable trust for websites.

### Chrome/Chromium

Chrome uses the operating system trust store on macOS and Windows.

On Linux, Chromium-based browsers may use an NSS database. One common setup is:

```bash
certutil -A -d "sql:$HOME/.pki/nssdb" -n "Groxy Local CA" -t "C,," -i groxy-ca.pem
```

### macOS

Open Keychain Access, import `groxy-ca.pem` into the System or login keychain,
open the certificate, and set Trust → Secure Sockets Layer (SSL) to Always Trust.

### Windows

Run `certmgr.msc` or Manage User Certificates, then import `groxy-ca.pem` into
Trusted Root Certification Authorities → Certificates.

### Linux system trust

On Debian/Ubuntu:

```bash
sudo cp groxy-ca.pem /usr/local/share/ca-certificates/groxy-ca.crt
sudo update-ca-certificates
```

On Fedora/RHEL:

```bash
sudo cp groxy-ca.pem /etc/pki/ca-trust/source/anchors/groxy-ca.pem
sudo update-ca-trust
```

Restart the browser or application after importing the certificate.

## Passthrough on inspection setup errors

By default, inspection setup errors fail closed. If Groxy cannot prepare
inspection for a matched host, it returns an error rather than silently tunneling
the connection.

To explicitly fall back to a normal CONNECT tunnel on inspection setup errors:

```go
HTTPSInspection: &groxy.HTTPSInspectionConfig{
	CA:                 ca,
	Intercept:          groxy.MatchHosts("example.com"),
	PassthroughOnError: true,
}
```

Use this only if bypassing inspection is acceptable for your application.

## What middleware can do

For inspected HTTPS traffic, Groxy parses the HTTP requests inside the TLS
connection and runs normal middleware:

- request hooks
- response hooks
- blocking helpers
- header helpers
- request/response body transforms

Treat middleware as privileged code. It may be able to observe or modify
sensitive headers, cookies, tokens, and bodies for inspected hosts.

## Current limitations

- Intercepted client traffic is HTTP/1.1 over TLS.
- HTTP/2 inspection is not implemented yet.
- Users must install and trust the local CA manually.
- Generated per-host certificates are kept in memory and renewed before expiry.
- Certificate persistence hooks are not implemented yet.

For trust boundaries and deployment assumptions, see the
[HTTPS inspection threat model](https-inspection-threat-model.md).
