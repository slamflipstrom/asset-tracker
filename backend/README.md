# Backend (Go)

This folder contains two services:
- `cmd/worker`: price polling worker
- `cmd/ws`: WebSocket server

## Quick start (later)
1. Ensure env vars are set (see `internal/config`).
2. Install deps: `go mod tidy`
3. Run:
   - Worker: `go run ./cmd/worker`
   - WS server: `go run ./cmd/ws`

## Notes
- This is scaffolding only; provider clients, JWT verification, and SQL queries are stubs.
- We'll add provider integrations and concrete DB queries next.
