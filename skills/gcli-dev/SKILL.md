---
name: gcli-dev
description: Build and ship the gcli Gmail CLI with production gates, tests, and release automation.
---

# gcli-dev skill

## Trigger
Use this skill when implementing features for the `gcli` project.

## Workflow
1. Read `README.md` and `AGENTS.md` to align with output contracts and gate requirements.
2. Implement command behavior in `cmd/gcli` and business logic in `pkg/*`.
3. Add/adjust tests in `pkg/*_test.go` and `e2e/`.
4. Run `make fmt vet lint test release-check`.
5. Update `CHANGELOG.md` for user-visible changes.

## Safety
- Never hardcode credentials.
- Keep JSON response schema backward-compatible.
- Prefer deterministic test doubles over live Gmail API calls.
