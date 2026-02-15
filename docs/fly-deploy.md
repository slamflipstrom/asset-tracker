# Fly Deployment

This project deploys as two Fly apps:

- `asset-ws` for WebSocket/API ingress
- `asset-worker` for scheduled price refresh

## Files

- `/Users/samlindstrom/Code/asset-tracker/fly.ws.toml`
- `/Users/samlindstrom/Code/asset-tracker/fly.worker.toml`
- `/Users/samlindstrom/Code/asset-tracker/backend/Dockerfile.ws`
- `/Users/samlindstrom/Code/asset-tracker/backend/Dockerfile.worker`

## One-time setup

1. Create both apps:
   - `fly apps create asset-ws`
   - `fly apps create asset-worker`
2. Set secrets for both apps (adjust as needed):
   - `fly secrets set --app asset-ws DATABASE_URL=... SUPABASE_URL=... SUPABASE_SECRET_KEY=...`
   - `fly secrets set --app asset-worker DATABASE_URL=... CRYPTO_PROVIDER_NAME=mobula CRYPTO_PROVIDER_API_KEY=...`

## Deploy

1. WebSocket service:
   - `fly deploy --config /Users/samlindstrom/Code/asset-tracker/fly.ws.toml`
2. Worker service:
   - `fly deploy --config /Users/samlindstrom/Code/asset-tracker/fly.worker.toml`

## Verify

1. Health check:
   - `fly status --app asset-ws`
2. Worker logs:
   - `fly logs --app asset-worker`
3. WebSocket logs:
   - `fly logs --app asset-ws`

## Notes

- `asset-ws` keeps one machine running to avoid websocket reconnect churn.
- `asset-worker` has no public service ports.
