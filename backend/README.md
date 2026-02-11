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

## Crypto provider (v1)
- Set `CRYPTO_PROVIDER_NAME=mobula` (or `coingecko`, `coingecko-pro`).
- Set `CRYPTO_PROVIDER_API_KEY`.
- Optional: set `CRYPTO_PROVIDER_BASE_URL`.
- Store ticker in `assets.symbol` (for example, `BTC`).
- Store provider lookup id in `assets.market_data_id`.
- For Mobula, use the Mobula `id` as `market_data_id`.

## Fly deployment
- WebSocket app config: `/Users/samlindstrom/Code/asset-tracker/fly.ws.toml`
- Worker app config: `/Users/samlindstrom/Code/asset-tracker/fly.worker.toml`
- Deployment runbook: `/Users/samlindstrom/Code/asset-tracker/docs/fly-deploy.md`
