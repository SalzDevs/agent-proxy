# Groxy documentation

These guides complement the short project overview in [`README.md`](../README.md)
and the generated API reference on
[pkg.go.dev](https://pkg.go.dev/github.com/SalzDevs/groxy).

## Start here

- [Building a forward proxy in Go with Groxy](building-forward-proxy.md) - a
  step-by-step introduction with middleware, access logs, blocking, body
  transforms, and HTTPS inspection.
- [Timeout semantics](timeouts.md) - exact client-to-proxy and proxy-to-upstream
  timeout behavior.
- [Proxy authentication](proxy-auth.md) - Basic proxy auth setup, custom
  validators, and security notes.
- [HTTPS inspection guide](https-inspection.md) - setup, CA trust instructions,
  host matching, passthrough behavior, and operational notes.
- [HTTPS inspection threat model](https-inspection-threat-model.md) - trust
  boundaries and safety assumptions for TLS interception.

## Runnable examples

See [`../examples`](../examples) for small programs you can run locally.

## Repository layout

```text
.
├── *.go                 # public groxy package
├── cmd/groxy            # small demo proxy binary
├── docs                 # guides and security/operational notes
├── examples             # runnable examples
├── .github              # CI, issue templates, PR template
├── README.md            # landing page / quickstart
├── CHANGELOG.md         # release history
├── CONTRIBUTING.md      # contributor guide
├── ROADMAP.md           # planned work
├── SECURITY.md          # vulnerability reporting policy
└── RELEASE.md           # maintainer release checklist
```

The Go source files intentionally live at the repository root because the module
exports a single public package: `github.com/SalzDevs/groxy`.
