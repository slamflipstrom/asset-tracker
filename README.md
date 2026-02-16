# Asset Tracker

Asset Tracker is a portfolio tracker with a Go backend and React frontend:

- a **worker** that refreshes market prices into Postgres
- a **WebSocket service** that authenticates users and manages subscriptions
- a **Supabase schema** for assets, lots, prices, and portfolio views
- a **frontend UI** for auth, portfolio overview, and lots CRUD

## Repository Layout

- `backend/`: Go services and internal packages
- `frontend/`: React UI for auth, portfolio, and lots management
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
  - Exposes `GET /health`, `GET /ws`, and versioned REST routes under `/api/v1`
  - Verifies Supabase bearer tokens via `/auth/v1/user`
  - Supports subscribe/unsubscribe messages for `portfolio` and `asset` scopes
  - API routes:
    - `GET /api/v1/positions`
    - `GET /api/v1/lots`
    - `POST /api/v1/lots`
    - `PATCH /api/v1/lots/{lotID}`
    - `DELETE /api/v1/lots/{lotID}`
    - `GET /api/v1/assets/search`
- `frontend/`
  - React + Vite app with Supabase Auth
  - Uses `/api/v1` routes from `backend/cmd/ws` for portfolio + lot management
  - Uses Supabase Realtime (`lots`, `prices_current`) with polling fallback

## Providers

- Crypto:
  - `mobula` (implemented)
  - `coingecko` / `coingecko-pro` (implemented)
- Stock:
  - `http` provider exists as a placeholder and is not yet implemented

## Local Development

Prereqs:
- Go 1.24+
- Node.js 20+
- pnpm 10+
- Docker (required for `supabase start`)
- Supabase CLI
- Provider API keys (worker)

1. Start local Supabase (repo root):
   - `supabase start`
   - `supabase db reset`
2. Configure frontend env:
   - `cd frontend`
   - `cp .env.example .env.local`
   - Set `VITE_SUPABASE_URL` and one key:
   - `VITE_SUPABASE_PUBLISHABLE_KEY` (recommended)
   - `VITE_SUPABASE_ANON_KEY` (legacy local/self-hosted fallback)
   - You can get local values from `supabase status`
3. Start WS/API server (required for frontend data operations):
   - In `backend/`, set:
   - `DATABASE_URL`
   - `SUPABASE_URL`
   - `SUPABASE_SECRET_KEY`
   - optional `PORT` (defaults to `8080`)
   - Run: `go run ./cmd/ws`
4. Start frontend:
   - `pnpm install`
   - `pnpm dev`
   - By default, Vite proxies `/api/*` to `http://127.0.0.1:8080`
5. Optional: start worker for live prices:
   - In `backend/`, set:
   - `DATABASE_URL`
   - `CRYPTO_PROVIDER_NAME`
   - `CRYPTO_PROVIDER_API_KEY`
   - optional `CRYPTO_PROVIDER_BASE_URL`
   - Run: `go run ./cmd/worker`

## Frontend Development

The UI is in `frontend/`.

1. In `frontend/`, create env vars:
   - `cp .env.example .env.local`
   - Set `VITE_SUPABASE_URL` and:
   - `VITE_SUPABASE_PUBLISHABLE_KEY` (hosted Supabase, recommended), or
   - `VITE_SUPABASE_ANON_KEY` (local/self-hosted fallback)
   - Optional: `VITE_API_BASE_URL` when API is on another origin
2. Start UI:
   - `pnpm install`
   - `pnpm dev`

Notes:
- Create a user from the sign-up flow before first sign-in on local Supabase.
- Portfolio/lots reads and writes go through `/api/v1` on `backend/cmd/ws`.
- Current prices populate when the worker writes `prices_current`.

## Test

From `backend/`:

- `go test ./...`

From `frontend/`:

- `pnpm build`

If your environment blocks default Go cache writes, set:

- `GOCACHE=/absolute/path/to/repo/.gocache go test ./...`

## Deployment

- Fly configs:
  - `fly.ws.toml`
  - `fly.worker.toml`
- Deployment guide:
  - `docs/fly-deploy.md`
- API contract:
  - `docs/api-v1.md`
- Planning:
  - `docs/v1-checkpoint-plan.md`
