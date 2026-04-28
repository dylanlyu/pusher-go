# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Purpose

Unofficial Go SDK for Pusher, covering two products:

- **Channels** — server-side publishing of WebSocket events via the Pusher HTTP API
- **Beams** — server-side trigger of push notifications via the Pusher Beams API

This is a self-maintained fork of the unmaintained official libraries (`pusher-http-go` and `push-notifications-go`). Key improvements over the originals:

- Replaced archived `dgrijalva/jwt-go` with `golang-jwt/jwt/v5`
- Replaced deprecated `ioutil` with `io`
- Added `context.Context` to all HTTP methods
- Fixed silent body mutation bug in publish methods
- Fixed webhook signature bypass security issue

## Module Structure

This repo uses a **Go workspace** (`go.work`). Five independent modules:

| Module | Path | Purpose | Independently versioned |
|---|---|---|---|
| `github.com/dylanlyu/pusher-go/channels` | `./channels` | Pusher Channels HTTP API client | ✅ |
| `github.com/dylanlyu/pusher-go/beams` | `./beams` | Pusher Beams push notification client | ✅ |
| `github.com/dylanlyu/pusher-go/config` | `./config` | Shared connection config types and generic Option[T] | ✅ |
| `github.com/dylanlyu/pusher-go/pusher` | `./pusher` | Reserved for future public shared utilities | ✅ |
| `github.com/dylanlyu/pusher-go/internal` | `./internal` | Private shared implementations (HTTP, HMAC auth) | ❌ |

### Dependency graph

```
channels → config, internal, golang.org/x/crypto
beams    → config, internal, golang-jwt/jwt/v5
internal → (stdlib only)
config   → (stdlib only)
pusher   → (empty shell, no deps)
```

`internal` is not part of the public API and must never be imported by external consumers.

## Common Commands

```bash
# Build per module (workspace root `go build ./...` does not work with go.work)
go -C channels build ./...
go -C beams build ./...
go -C internal build ./...

# Run all tests with race detector and coverage
go -C channels test -race -cover ./...
go -C beams test -race -cover ./...
go -C internal test -race -cover ./...

# Run a single test
go -C channels test -run TestTrigger -v ./...

# Format
gofmt -w channels/ beams/ internal/ config/

# Vet
go -C channels vet ./...
go -C beams vet ./...
go -C internal vet ./...
```

> Do **not** run `go mod init` — all modules already have their own `go.mod`.
> Do **not** run `go build ./...` from repo root — use `-C <module>` per module.

## Architecture

```
pusher-go/
├── go.work                      # Workspace: ties all modules together
├── go.work.sum
├── config/
│   ├── go.mod                   # module github.com/dylanlyu/pusher-go/config
│   └── config.go                # BaseConfig, generic Option[T any] func(*T)
├── pusher/
│   └── go.mod                   # module github.com/dylanlyu/pusher-go/pusher (empty shell)
├── internal/
│   ├── go.mod                   # module github.com/dylanlyu/pusher-go/internal
│   ├── request/
│   │   ├── request.go           # Do(): shared HTTP execution, ErrHTTP error type
│   │   └── request_test.go
│   └── auth/
│       ├── auth.go              # HMACSignature, CheckSignature, CreateAuthMap, MD5Hex
│       └── auth_test.go
├── channels/
│   ├── go.mod                   # module github.com/dylanlyu/pusher-go/channels
│   ├── client.go                # Client interface, New(), all method implementations
│   ├── options.go               # channelConfig, functional options
│   ├── types.go                 # all public types (Event, Webhook, Channel, Users, …)
│   ├── crypto.go                # NaCl secretbox E2E encryption (private-encrypted-*)
│   ├── encoder.go               # trigger payload encoding, payload size check
│   ├── webhook.go               # parseWebhook()
│   ├── url.go                   # buildRequestURL(): HMAC-signed Pusher API URLs
│   ├── util.go                  # ValidChannel, validateSocketID, validUserID, …
│   └── *_test.go
└── beams/
    ├── go.mod                   # module github.com/dylanlyu/pusher-go/beams
    ├── client.go                # Client interface, New(), all method implementations
    ├── options.go               # beamConfig, functional options
    ├── types.go                 # publishResponse, errorResponse
    └── *_test.go
```

## Design Constraints

- **Immutable request objects** — never mutate the caller's data (use `copyMapWithKey`).
- **Interface-first** — public surface is a `Client` interface; concrete `client` struct is unexported, enabling mocking via `http.RoundTripper` in tests.
- **Error wrapping** — all errors use `fmt.Errorf("pkg: ...: %w", err)` for `errors.Is`/`errors.As` compatibility.
- **No global mutable state** — no `init()` side effects; `defaultHeaders()` returns a fresh map on each call instead of a package-level var.
- **context.Context on all HTTP methods** — callers control timeout and cancellation.

## HTTP Client Contract

Both `channels` and `beams` accept an optional `*http.Client` via `WithHTTPClient(hc)` at construction time. Zero value falls back to `http.DefaultClient`. Tests use `roundTripFunc` (a `http.RoundTripper` adapter) for transport-level mocking without real network calls.

## workspace-local module resolution

`channels/go.mod` and `beams/go.mod` contain `replace` directives pointing to `../config` and `../internal`. These are required alongside `go.work` entries so that `go get` resolves local modules without hitting the module proxy.

## Test Coverage Targets

| Module | Target | Current |
|---|---|---|
| `channels` | ≥ 80% | ~82% |
| `beams` | ≥ 80% | ~88% |
| `internal/auth` | ≥ 80% | 100% |
| `internal/request` | ≥ 80% | ~86% |
