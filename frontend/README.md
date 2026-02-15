# Frontend UI (React + Vite)

Basic UI for:
- Supabase email/password auth
- Portfolio overview from `positions_view`
- Lots CRUD (`lots` table)
- Asset search (`assets` table)

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

For local Supabase, use the values printed by `supabase status`.

3. Install and run:

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
- Editing a lot currently updates quantity, unit cost, and purchase date.
- Asset is locked during edit to avoid accidental asset re-linking.

## Troubleshooting

- `Invalid API key` on login:
  - Your project URL and key likely do not match. Use a key from the same Supabase project as `VITE_SUPABASE_URL`.
- `Invalid login credentials` on local:
  - The user usually does not exist in the local project yet. Use the sign-up flow first, then sign in.
- Prices are empty:
  - `positions_view` can return `null` `current_price` until `prices_current` is populated by the worker.
