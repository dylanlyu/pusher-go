# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Purpose

Unofficial Go SDK for Pusher, covering two products:

- **Channels** — server-side publishing of WebSocket events via the Pusher HTTP API
- **Beams** — server-side trigger of push notifications via the Pusher Beams API

## Module Structure

This repo uses a **Go workspace** (`go.work`). There are three independent modules:

| Module | Path | Purpose |
|---|---|---|
| `github.com/dylanlyu/pusher-go/channels` | `./channels` | Pusher Channels HTTP API client |
| `github.com/dylanlyu/pusher-go/beams` | `./beams` | Pusher Beams push notification client |
| `github.com/dylanlyu/pusher-go/internal` | `./internal` | Shared utilities (e.g. HTTP request helpers) |

## Common Commands

```bash
# Build all modules (run from repo root, workspace resolves deps)
go build ./...

# Run all tests
go test ./...

# Run tests with race detector
go test -race ./...

# Run a single test
go test ./channels/... -run TestPublish -v

# Lint (requires golangci-lint)
golangci-lint run ./...

# Format
gofmt -w .

# Vet
go vet ./...
```

> Do **not** run `go mod init` — all three modules already have their own `go.mod`.

## Architecture

```
pusher-go/
├── go.work                  # Workspace: ties channels / beams / internal together
├── channels/
│   ├── go.mod               # module github.com/dylanlyu/pusher-go/channels
│   └── client.go
├── beams/
│   ├── go.mod               # module github.com/dylanlyu/pusher-go/beams
│   └── client.go
└── internal/
    ├── go.mod               # module github.com/dylanlyu/pusher-go/internal
    └── request/             # shared HTTP request utilities
```

`channels` and `beams` are the public-facing packages. `internal` holds cross-cutting utilities (currently HTTP request helpers); it is imported only by the two product packages and is not part of the public API.

### Design Constraints

- **Immutable request objects** — construct, then send; never mutate after construction.
- **Interface-first** — the public surface is a `Client` interface; the concrete struct is unexported or clearly marked as an implementation detail, enabling mocking in consumer tests.
- **Error wrapping** — all returned errors are wrapped with `fmt.Errorf("...: %w", err)` so callers can `errors.Is`/`errors.As`.
- **No global state** — no `init()` side effects, no package-level variables that change at runtime.

### HTTP Client Contract

Both `channels` and `beams` accept an optional `*http.Client` at construction time (functional option or struct field). The zero value falls back to `http.DefaultClient`. This makes tests deterministic via transport-level mocking (`http.RoundTripper`).
