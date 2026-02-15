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

## Notes

- Uses Supabase Realtime subscriptions on `public.lots` and `public.prices_current` with polling fallback (`VITE_REFRESH_MS`, default 30s).
- If live updates never connect, enable Realtime for those tables in your Supabase project.
- Editing a lot currently updates quantity, unit cost, and purchase date.
- Asset is locked during edit to avoid accidental asset re-linking.
