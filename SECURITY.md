# Security Policy

Groxy is a proxy library and includes optional HTTPS inspection functionality, so
security reports are taken seriously.

## Supported versions

Groxy is currently pre-v1. Security fixes are prioritized for:

- the latest tagged release
- the `main` branch

Older pre-v1 releases may not receive backported fixes unless the issue is
critical and practical to patch.

## Reporting a vulnerability

Please do **not** open a public GitHub issue for security vulnerabilities.

Preferred reporting path:

1. Use GitHub's private vulnerability reporting / security advisory flow for
   this repository if available.
2. If that is not available, contact the maintainer privately through GitHub and
   avoid sharing exploit details publicly until a fix is ready.

Please include:

- affected Groxy version or commit
- a clear description of the issue
- reproduction steps or proof of concept, if safe to share privately
- expected impact
- any suggested fix or mitigation

## HTTPS inspection note

HTTPS inspection/MITM features must be used only on traffic you own or are
authorized to inspect. Users are responsible for installing and protecting their
local CA private key securely.
