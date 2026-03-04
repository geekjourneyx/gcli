# AGENTS

## Engineering policy

- Keep command behavior backward-compatible.
- Preserve JSON schema compatibility: additive changes only.
- Never commit real credentials or tokens.
- Prefer deterministic tests over live network calls.

## Required checks before merge

- `gofmt -l .` returns no output.
- `go vet ./...` passes.
- `golangci-lint run` passes.
- `CGO_ENABLED=1 go test -count=1 ./...` passes.
- `make release-check` passes.

## CLI contract rules

- Default output is JSON.
- Machine-consumed commands must stay stable across versions.
- Error handling must use structured error codes, not string matching.
