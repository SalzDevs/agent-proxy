# Changelog

All notable changes to Groxy will be documented in this file.

Groxy is currently pre-v1, so minor releases may include API changes.

## [Unreleased]

### Added

- `AccessLog` middleware for one-line HTTP and CONNECT traffic logs.
- Documentation index in `docs/README.md`.
- HTTPS inspection guide and threat model.
- Examples index in `examples/README.md`.

### Changed

- README now stays shorter and links to deeper guides for advanced topics.

### Planned

- Additional observability helpers
- Proxy authentication helpers
- Additional HTTPS inspection controls and docs

## [v0.3.1] - 2026-05-11

### Added

- Quickstart section: "Try it in 60 seconds".
- Forward proxy guide in `docs/building-forward-proxy.md`.
- CA trust instructions for HTTPS inspection.
- Open source project hygiene files, issue templates, and PR template.

## [v0.3.0] - 2026-05-10

### Added

- Opt-in HTTPS inspection using local TLS interception.
- `HTTPSInspectionConfig` and `Config.HTTPSInspection`.
- Local CA generation/loading with `NewCA`, `LoadCAFiles`, and `CA.WriteFiles`.
- Per-host certificate generation, caching, and renewal.
- Host matching helpers with `MatchHosts` and `MatchAllHosts`.
- HTTPS middleware support for request hooks, response hooks, blocking, and body transforms.
- HTTPS inspection example and README documentation.

### Changed

- CONNECT handling was split into tunneling and inspection paths.
- HTTP forwarding internals now share logic across HTTP and intercepted HTTPS.

## [v0.2.0] - 2026-05-10

### Added

- `Config.MaxBodySize`.
- `DefaultMaxBodySize`.
- `BodyTooLargeError`.
- Body size limits for body helpers and transform middleware.
- Documentation for body size configuration.

## [v0.1.0] - 2026-05-10

### Added

- Initial public release.
- HTTP forwarding.
- HTTPS CONNECT tunneling.
- Middleware registration with `Use`, `OnRequest`, `OnResponse`, and `OnConnect`.
- Blocking helpers.
- Header helpers.
- Request and response body transforms.
- Configurable timeouts and logging.
- Examples, benchmarks, CI, and release checklist.
