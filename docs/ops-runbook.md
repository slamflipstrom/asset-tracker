# Operations Runbook (V1 Baseline)

**Last updated:** February 19, 2026

## Scope

This runbook covers first-response checks for the v1 services:

- `asset-ws` (`backend/cmd/ws`) for API + websocket ingress
- `asset-worker` (`backend/cmd/worker`) for price refresh jobs

## Quick Health Checks

1. Confirm service health endpoint:
   - `GET /health` on `asset-ws` should return `200 ok`.
2. Confirm API traffic and error counters:
   - `GET /debug/vars` on `asset-ws`.
3. Confirm worker logs are advancing:
   - Recent `refresh cycle completed` log entries should be present.

## Metrics (from `/debug/vars`)

- `api_requests_total`: total `/api/v1` requests.
- `api_requests_errors_total`: `/api/v1` requests with status `>=400`.
- `api_request_latency_ms_total`: summed API latency in milliseconds.
- `api_request_latency_samples_total`: number of latency samples.
- `api_requests_by_route`: request count by `METHOD + route`.
- `api_request_errors_by_route`: error count by `METHOD + route`.
- `ws_connections_active`: active websocket sessions.
- `ws_connections_total`: total websocket sessions accepted since process start.
- `ws_auth_failures_total`: websocket auth failures.
- `ws_session_init_failures_total`: websocket session init failures.

Average API latency can be estimated as:

`api_request_latency_ms_total / api_request_latency_samples_total`

## Common Incidents

### Incident: API error rate spike

1. Check `api_requests_errors_total` and `api_request_errors_by_route`.
2. Identify the failing route and inspect `asset-ws` logs for matching errors.
3. Validate upstream dependencies:
   - DB reachability.
   - Supabase auth verification path.
4. If caused by a bad deploy, roll back and monitor counters for 10-15 minutes.

### Incident: WebSocket connection churn or failures

1. Check `ws_connections_active`, `ws_connections_total`, and `ws_auth_failures_total`.
2. If auth failures increase, validate Supabase token issuance and clock skew.
3. If active connections collapse, inspect `asset-ws` logs and platform restarts.
4. Fail over to polling behavior in frontend while investigating.

### Incident: Stale prices/positions

1. Confirm worker logs show recent successful refresh cycles.
2. Confirm API routes still return updated positions/lots data.
3. Confirm frontend realtime status; polling fallback should continue refreshing.
4. If worker is failing provider calls, rotate keys or degrade to last-known prices.

## Logging Baseline

- `asset-ws` and `asset-worker` emit structured JSON logs (`slog`).
- Key worker cycle fields: `tracked`, `due`, `quotes`, `updates_written`, `duration`.
