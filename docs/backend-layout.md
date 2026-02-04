# Backend Layout (Go + Fly)

## Goals
- Go worker polls separate stock and crypto providers on a fixed interval per asset.
- Go WebSocket server pushes price and position updates to connected clients.
- Supabase Auth + Postgres remains the source of truth.

## Suggested Repo Layout
- `backend/cmd/worker/main.go`
- `backend/cmd/ws/main.go`
- `backend/internal/config/`
- `backend/internal/db/`
- `backend/internal/providers/`
- `backend/internal/prices/`
- `backend/internal/auth/`
- `backend/internal/ws/`

## Package Responsibilities
- `internal/config`
  - Load env vars and validate required settings.
- `internal/db`
  - Query and upsert data in Supabase Postgres.
  - Shared SQL helpers for assets, lots, prices, settings.
- `internal/providers`
  - `StockProvider` and `CryptoProvider` interfaces.
  - Provider implementations and batching logic.
- `internal/prices`
  - Refresh scheduler.
  - Asset refresh planning using per-user intervals and global min/max.
- `internal/auth`
  - Supabase JWT verification via JWKS.
- `internal/ws`
  - WebSocket hub, subscription registry, fan-out.

## Worker Flow
- Load `app_settings` min and max refresh intervals.
- Build refresh plan per asset:
  - Effective interval = min(user intervals, max) and not lower than min.
- Poll providers per asset batch and update `prices_current` and `price_snapshots`.

## WebSocket Flow
- Client connects with `Authorization: Bearer <supabase_jwt>`.
- Server verifies token via Supabase JWKS.
- Client subscribes to `portfolio` or `asset` scope.
- Server pushes `price_update`, `position_update`, `lot_update` events.

## Required Env Vars (Initial)
- `SUPABASE_URL`
- `SUPABASE_SERVICE_ROLE_KEY`
- `SUPABASE_JWKS_URL`
- `STOCK_PROVIDER_API_KEY`
- `CRYPTO_PROVIDER_API_KEY`
- `WS_ALLOWED_ORIGINS`

## Fly Apps
- `asset-worker` for the price polling worker.
- `asset-ws` for the WebSocket server.

## Notes
- The worker should minimize external calls by batching per provider.
- The WebSocket server should query DB views for `positions_view` and `lot_performance_view` before push.
