# Changelog

All notable changes to Groxy will be documented in this file.

Groxy is currently pre-v1, so minor releases may include API changes.

## [Unreleased]

### Planned

- Observability and access logging helpers
- Proxy authentication helpers
- Additional HTTPS inspection controls and docs

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
