# Asset Tracker

Asset Tracker is a Go backend for portfolio tracking with:

- a **worker** that refreshes market prices into Postgres
- a **WebSocket service** that authenticates users and manages subscriptions
- a **Supabase schema** for assets, lots, prices, and portfolio views

## Repository Layout

- `backend/`: Go services and internal packages
- `supabase/`: schema, RLS policies, and migrations
- `docs/`: deployment and architecture notes
- `fly.ws.toml`, `fly.worker.toml`: Fly app configs

## Services

- `backend/cmd/worker`
  - Loads tracked assets from DB
  - Computes per-asset refresh cadence using `app_settings` + `user_settings`
  - Fetches quotes from configured providers
  - Writes to `prices_current` and `price_snapshots`
- `backend/cmd/ws`
  - Exposes `GET /health` and `GET /ws`
  - Verifies Supabase bearer tokens via `/auth/v1/user`
  - Supports subscribe/unsubscribe messages for `portfolio` and `asset` scopes

## Providers

- Crypto:
  - `mobula` (implemented)
  - `coingecko` / `coingecko-pro` (implemented)
- Stock:
  - `http` provider exists as a placeholder and is not yet implemented

## Local Development

Prereqs: Go 1.24+, Supabase CLI (for local DB), and provider API keys.

1. Start Supabase locally (from repo root):
   - `supabase start`
   - `supabase db reset`
2. In `backend/`, set required env vars (for at least one service):
   - `DATABASE_URL` (required by both binaries)
   - `SUPABASE_URL`, `SUPABASE_SECRET_KEY` (required by `ws`)
   - Provider vars for worker, e.g. `CRYPTO_PROVIDER_NAME`, `CRYPTO_PROVIDER_API_KEY`
3. Run services:
   - Worker: `go run ./cmd/worker`
   - WebSocket: `go run ./cmd/ws`

## Test

From `backend/`:

- `go test ./...`

If your environment blocks default Go cache writes, set:

- `GOCACHE=/absolute/path/to/repo/.gocache go test ./...`

## Deployment

- Fly configs:
  - `fly.ws.toml`
  - `fly.worker.toml`
- Deployment guide:
  - `docs/fly-deploy.md`
