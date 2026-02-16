# Backend (Go)

This folder contains two services:

- `cmd/worker`: price polling worker
- `cmd/ws`: WebSocket + REST API server

## WS/API routes

- `GET /health`
- `GET /ws`
- `GET /api/v1/positions`
- `GET /api/v1/lots`
- `POST /api/v1/lots`
- `PATCH /api/v1/lots/{lotID}`
- `DELETE /api/v1/lots/{lotID}`
- `GET /api/v1/assets/search`

Route contracts: `/Users/samlindstrom/Code/asset-tracker/docs/api-v1.md`

## Quick Start
1. Ensure env vars are set (see `internal/config`), per service:
   - Worker:
   - `DATABASE_URL`
   - `CRYPTO_PROVIDER_NAME`
   - `CRYPTO_PROVIDER_API_KEY`
   - optional `CRYPTO_PROVIDER_BASE_URL`
   - WebSocket:
   - `DATABASE_URL`
   - `SUPABASE_URL`
   - `SUPABASE_SECRET_KEY`
   - optional `PORT` (defaults to `8080`)
2. Install deps: `go mod tidy`
3. Run:
   - Worker: `go run ./cmd/worker`
   - WS server: `go run ./cmd/ws`

## Crypto provider (v1)

- Set `CRYPTO_PROVIDER_NAME=mobula` (or `coingecko`, `coingecko-pro`).
- Set `CRYPTO_PROVIDER_API_KEY`.
- Optional: set `CRYPTO_PROVIDER_BASE_URL`.
- Store ticker in `assets.symbol` (for example, `BTC`).
- Store provider lookup id in `assets.market_data_id`.
- For Mobula, use the asset key as `market_data_id` (for example, `bitcoin`).

## Fly deployment

- WebSocket app config: `/Users/samlindstrom/Code/asset-tracker/fly.ws.toml`
- Worker app config: `/Users/samlindstrom/Code/asset-tracker/fly.worker.toml`
- Deployment runbook: `/Users/samlindstrom/Code/asset-tracker/docs/fly-deploy.md`

## Tests

- `go test ./...`
- DB math integration test (`TestCostBasisAndPLViews`) runs when a test DB is reachable.
- Optional override for DB URL:
  - `ASSET_TRACKER_TEST_DATABASE_URL=postgresql://... go test ./internal/db -run TestCostBasisAndPLViews -v`
