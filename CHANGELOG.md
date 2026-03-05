# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-04
### Added
- Initial production-grade Gmail read-only CLI architecture (`gcli`).
- OAuth refresh-token environment auth and interactive `auth login` flow.
- Core commands: `mail list`, `mail get`, `mail search`.
- Stable JSON output envelope and structured error codes.
- CI/release workflow, installer script, and release consistency checks.
- Unit/integration/e2e test baseline.
- `auth login --auth-timeout` for interactive OAuth sessions.
- Structured error `details` field in JSON error envelope.
- `mail list/search --hydrate` compatibility mode for rich headers.

### Changed
- Switched `auth login` implementation from device flow to Authorization Code + PKCE for Gmail compatibility.
- Removed default list/search N+1 behavior to reduce API calls and latency.

### Fixed
- N/A (initial release).

### Docs
- Added README, AGENTS guidance, and skill packaging scaffold.

### Tests
- Added unit tests for config/output/errors and integration tests for Gmail adapter.
