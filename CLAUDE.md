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

Single root module `github.com/dylanlyu/pusher-go` (one `go.mod` at repo root, no `go.work`).

| Package | Import path | Purpose |
|---|---|---|
| `service/channels` | `github.com/dylanlyu/pusher-go/service/channels` | Pusher Channels HTTP API client |
| `service/beams` | `github.com/dylanlyu/pusher-go/service/beams` | Pusher Beams push notification client |
| `config` | `github.com/dylanlyu/pusher-go/config` | Shared `BaseConfig` and generic `Option[T]` |
| `internal/auth` | `github.com/dylanlyu/pusher-go/internal/auth` | HMAC signature helpers (private) |
| `internal/request` | `github.com/dylanlyu/pusher-go/internal/request` | Shared HTTP execution, `ErrHTTP` (private) |
| `pusher` | `github.com/dylanlyu/pusher-go/pusher` | Empty shell, reserved for future use |

### Dependency graph

```
service/channels → config, internal/auth, internal/request, golang.org/x/crypto
service/beams    → config, internal/request, golang-jwt/jwt/v5
internal/auth    → (stdlib only)
internal/request → (stdlib only)
config           → (stdlib only)
```

`internal/*` packages are not part of the public API and must never be imported by external consumers.

## Common Commands

```bash
# Build all packages
go build ./...

# Run all tests with race detector and coverage
go test -race -cover ./...

# Run a single test
go test -run TestTrigger -v ./service/channels/...

# Format
gofmt -w .

# Vet
go vet ./...
```

> Do **not** run `go mod init` — the root `go.mod` already exists.
> Do **not** use `go -C <module>` — the repo is now a single module.

## Architecture

```
pusher-go/
├── go.mod                       # module github.com/dylanlyu/pusher-go
├── go.sum
├── config/
│   └── config.go                # BaseConfig, generic Option[T any] func(*T)
├── pusher/                      # empty shell, reserved for future use
├── internal/
│   ├── auth/
│   │   ├── auth.go              # HMACSignature, CheckSignature, CreateAuthMap, MD5Hex
│   │   └── auth_test.go
│   └── request/
│       ├── request.go           # Do(): shared HTTP execution, ErrHTTP error type
│       └── request_test.go
└── service/
    ├── channels/
    │   ├── client.go            # Client interface, New(), all method implementations
    │   ├── options.go           # channelConfig, functional options
    │   ├── types.go             # all public types (Event, Webhook, Channel, Users, …)
    │   ├── crypto.go            # NaCl secretbox E2E encryption (private-encrypted-*)
    │   ├── encoder.go           # trigger payload encoding, payload size check
    │   ├── webhook.go           # parseWebhook()
    │   ├── url.go               # buildRequestURL(): HMAC-signed Pusher API URLs
    │   ├── util.go              # ValidChannel, validateSocketID, validUserID, …
    │   └── *_test.go
    └── beams/
        ├── client.go            # Client interface, New(), all method implementations
        ├── options.go           # beamConfig, functional options
        ├── types.go             # publishResponse, errorResponse
        └── *_test.go
```

## Design Constraints

- **Immutable request objects** — never mutate the caller's data (use `copyMapWithKey`).
- **Interface-first** — public surface is a `Client` interface; concrete `client` struct is unexported, enabling mocking via `http.RoundTripper` in tests.
- **Error wrapping** — all errors use `fmt.Errorf("pkg: ...: %w", err)` for `errors.Is`/`errors.As` compatibility.
- **No global mutable state** — no `init()` side effects; `defaultHeaders()` returns a fresh map on each call instead of a package-level var.
- **context.Context on all HTTP methods** — callers control timeout and cancellation.

## HTTP Client Contract

Both `service/channels` and `service/beams` accept an optional `*http.Client` via `WithHTTPClient(hc)` at construction time. Zero value falls back to `http.DefaultClient`. Tests use `roundTripFunc` (a `http.RoundTripper` adapter) for transport-level mocking without real network calls.

## Test Coverage Targets

| Package | Target | Current |
|---|---|---|
| `service/channels` | ≥ 80% | ~82% |
| `service/beams` | ≥ 80% | ~88% |
| `internal/auth` | ≥ 80% | 100% |
| `internal/request` | ≥ 80% | ~86% |
