# Frontend UI (React + Vite)

Basic UI for:
- Supabase email/password auth
- Portfolio overview from `GET /api/v1/positions`
- Lots CRUD via `/api/v1/lots`
- Asset search via `/api/v1/assets/search`

## Requirements

- Node.js 20+
- Supabase project (local or hosted)

## Setup

1. Create env file:

```bash
cd frontend
cp .env.example .env.local
```

2. Set values in `.env.local`:
- `VITE_SUPABASE_URL`
- `VITE_SUPABASE_PUBLISHABLE_KEY` (hosted Supabase, recommended), or
- `VITE_SUPABASE_ANON_KEY` (local/self-hosted fallback)
- Optional: `VITE_API_BASE_URL` if API is served on another origin

For local Supabase, use the values printed by `supabase status`.

3. Start backend WS/API server (repo root):

```bash
cd backend
go run ./cmd/ws
```

4. Install and run frontend:

```bash
pnpm install
pnpm dev
```

## Scripts

- `pnpm dev`: Start local dev server
- `pnpm build`: Type-check and build production assets
- `pnpm preview`: Preview the production build locally

## Notes

- Uses Supabase Realtime subscriptions on `public.lots` and `public.prices_current` with polling fallback (`VITE_REFRESH_MS`, default 30s).
- If live updates never connect, enable Realtime for those tables in your Supabase project.
- In local dev, `/api/*` is proxied to `http://127.0.0.1:8080` by default.
- Override dev proxy target with `VITE_DEV_API_PROXY_TARGET`.
- Editing a lot currently updates quantity, unit cost, and purchase date.
- Asset is locked during edit to avoid accidental asset re-linking.

## Troubleshooting

- `Invalid API key` on login:
  - Your project URL and key likely do not match. Use a key from the same Supabase project as `VITE_SUPABASE_URL`.
- `Invalid login credentials` on local:
  - The user usually does not exist in the local project yet. Use the sign-up flow first, then sign in.
- Prices are empty:
  - `positions_view` can return `null` `current_price` until `prices_current` is populated by the worker.
