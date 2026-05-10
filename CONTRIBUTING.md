# Contributing to Groxy

Thanks for considering a contribution to Groxy.

Groxy is currently pre-v1. The API is intentionally still open to feedback, so
issues and pull requests about API design, naming, docs, examples, and developer
experience are very welcome.

## Before you start

For larger changes, please open an issue first so we can discuss the design
before you spend time implementing it.

Good contribution areas include:

- docs and examples
- tests and benchmarks
- middleware helpers
- proxy correctness fixes
- observability and logging
- HTTPS inspection safety improvements

Please do not report security vulnerabilities in public issues. See
[`SECURITY.md`](SECURITY.md) instead.

## Development setup

Requirements:

- Go 1.24 or newer
- Git

Clone the repository:

```bash
git clone https://github.com/SalzDevs/groxy.git
cd groxy
```

Run the test suite:

```bash
go test ./...
```

Run vet:

```bash
go vet ./...
```

Run race tests before larger changes:

```bash
go test -race ./...
```

Run benchmarks when changing hot paths:

```bash
go test -run '^$' -bench=. -benchmem ./...
```

Format Go code before opening a PR:

```bash
gofmt -w .
```

## Pull request guidelines

A good PR should:

- explain the problem and solution clearly
- include tests for behavior changes
- update README/examples/godoc when public API changes
- keep public APIs beginner-friendly
- avoid unrelated formatting or drive-by changes

For public API changes, please include a short usage example in the PR
description.

## Commit messages

Use concise, conventional-style commit messages when possible:

```text
feat: add access log middleware
fix: preserve proxy headers correctly
docs: clarify HTTPS inspection setup
test: cover blocked HTTPS requests
```

## Release process

Maintainers use [`RELEASE.md`](RELEASE.md) before tagging releases.
