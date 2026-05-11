# HTTPS inspection threat model

This document describes the trust boundaries and safety assumptions for Groxy's
optional HTTPS inspection feature.

Groxy is a library for building proxies. Applications embedding Groxy decide
whether to enable inspection, which hosts to inspect, which middleware to run,
and how to protect generated CA material.

## Default behavior

By default, Groxy does **not** inspect HTTPS.

When `Config.HTTPSInspection` is nil, HTTPS requests use normal CONNECT
tunneling. Groxy copies encrypted bytes between the client and upstream server
and cannot read or modify HTTPS request or response bodies.

## Inspection trust boundary

When HTTPS inspection is enabled for a host, Groxy terminates TLS from the
client with a certificate signed by a local Groxy CA. Groxy then opens its own
TLS connection to the upstream server.

That means Groxy and its middleware become part of the trusted path for inspected
traffic. Middleware may be able to observe or modify sensitive data, including:

- URLs and query parameters
- headers
- cookies
- authorization tokens
- request bodies
- response bodies

Only enable inspection in environments where this is expected and authorized.

## Scope controls

Inspection requires:

1. `Config.HTTPSInspection` to be non-nil.
2. A valid CA in `HTTPSInspectionConfig.CA`.
3. A host matcher in `HTTPSInspectionConfig.Intercept`.

Hosts that do not match `Intercept` are tunneled normally.

Prefer narrow matchers:

```go
Intercept: groxy.MatchHosts("example.com", "*.example.com")
```

Use broad matchers only when that is explicitly intended:

```go
Intercept: groxy.MatchAllHosts()
```

Applications should make inspection scope visible to operators and users. Avoid
configuration flows where `MatchAllHosts()` can be enabled accidentally.

## Bypass and passthrough behavior

Groxy fails closed by default for inspection setup errors. If a host matches the
inspection policy but Groxy cannot prepare inspection, the request fails instead
of silently falling back to a tunnel.

`PassthroughOnError` changes that behavior:

```go
HTTPSInspection: &groxy.HTTPSInspectionConfig{
	CA:                 ca,
	Intercept:          groxy.MatchHosts("example.com"),
	PassthroughOnError: true,
}
```

With `PassthroughOnError`, setup failures can fall back to normal CONNECT
tunneling. This may be useful for availability, but it is a policy bypass if
your application requires inspection.

Applications that require strict inspection should leave `PassthroughOnError`
false and monitor/log inspection failures.

## CA private key handling

The Groxy CA private key is highly sensitive. Anyone with the private key and a
trusted CA certificate can generate certificates trusted by configured clients.

Recommended practices:

- Install only the public CA certificate on client devices.
- Keep the CA private key out of source control.
- Restrict filesystem permissions for CA key files.
- Use separate CAs for development, testing, and production.
- Rotate/remove trusted CAs when they are no longer needed.
- Remove local CA trust from browsers/OS trust stores after testing.

## Safe deployment assumptions

HTTPS inspection is most appropriate for:

- local development and debugging
- test environments
- internal tools where users/operators explicitly consent
- traffic owned by the application or organization running the proxy

HTTPS inspection is not appropriate for intercepting third-party traffic without
authorization.

## What Groxy does not protect against

Groxy does not protect against:

- malicious middleware in the embedding application
- compromise of the CA private key
- users installing the CA into unintended trust stores
- legal/compliance misuse of HTTPS inspection
- endpoint compromise on clients or proxy hosts
- intentional broad inspection policies such as accidental `MatchAllHosts()` use

Groxy provides safer defaults, but the embedding application owns the final trust
model and deployment policy.

## Pre-v1 review questions

Before v1, Groxy should continue collecting feedback on:

- whether host matching APIs are explicit enough
- whether passthrough behavior is documented clearly enough
- whether applications need richer inspection policy hooks
- whether custom logging/auditing hooks should be first-class
- whether generated certificate lifecycle controls should be expanded

Related guide: [HTTPS inspection guide](https-inspection.md).
