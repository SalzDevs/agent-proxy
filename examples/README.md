# Groxy examples

Run these examples from the repository root.

## Basic proxy

Starts a forward proxy on `127.0.0.1:8080`.

```bash
go run ./examples/basic
```

Try it from another terminal:

```bash
curl -x http://127.0.0.1:8080 http://example.com
curl -x http://127.0.0.1:8080 https://example.com
```

## Middleware

Shows request/response/CONNECT hooks and header helpers.

```bash
go run ./examples/middleware
```

## Body transform

Shows request and response body transforms.

```bash
go run ./examples/body-transform
```

## HTTPS inspection

Shows opt-in HTTPS inspection with a local CA.

```bash
go run ./examples/https-inspection
```

HTTPS inspection requires installing/trusting the generated CA certificate in the
client environment. See the [HTTPS inspection guide](../docs/https-inspection.md)
and [threat model](../docs/https-inspection-threat-model.md) before using it.
