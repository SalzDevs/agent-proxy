# Roadmap

Groxy is pre-v1. This roadmap is intentionally flexible and may change based on
user feedback.

## v0.4.0: Observability

Goal: make Groxy easier to debug and monitor in real applications.

Potential work:

- access log middleware
- request duration tracking
- status code tracking
- bytes in/out tracking
- structured event hooks
- benchmark coverage for HTTPS inspection

## v0.5.0: Access control

Goal: make it easier to run Groxy in controlled environments.

Potential work:

- proxy authentication helpers
- allowlist/denylist helpers
- token-based access control examples
- better custom error responses

## v0.6.0: HTTPS inspection hardening

Goal: improve safety, configurability, and production ergonomics for HTTPS
inspection.

Potential work:

- richer host matching controls
- custom upstream TLS settings
- better certificate lifecycle visibility
- optional certificate persistence hooks
- more documentation for browser/OS trust setup

## v1.0.0: API stabilization

Goal: stabilize the public API after real-world feedback.

Before v1:

- review exported names and docs
- confirm middleware API ergonomics
- document compatibility guarantees
- finalize error handling behavior
- ensure examples cover common use cases

## Good first issue ideas

These are intentionally scoped for new contributors:

- Add an access log middleware example.
- Add a benchmark for HTTPS inspection.
- Add more docs for installing the Groxy CA in common browsers/OSes.
- Add examples for proxy authentication middleware.
- Improve custom block/error response examples.
