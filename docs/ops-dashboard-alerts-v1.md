# Ops Dashboard + Alerts Spec (V1)

**Last updated:** February 19, 2026
**Status:** Baseline spec (ready for implementation in your monitoring stack)

## Goal

Define a minimal, production-usable dashboard and alert set for v1 using existing telemetry:

- `GET /debug/vars` from `asset-ws`
- structured JSON logs from `asset-ws` and `asset-worker`

## Data Sources

From `/debug/vars`:

- `api_requests_total`
- `api_requests_errors_total`
- `api_request_latency_ms_total`
- `api_request_latency_samples_total`
- `api_requests_by_route`
- `api_request_errors_by_route`
- `ws_connections_active`
- `ws_connections_total`
- `ws_auth_failures_total`
- `ws_session_init_failures_total`

From logs:

- `asset-worker`: `refresh cycle completed`
- `asset-worker`: `refresh cycle completed with errors`
- `asset-ws`: startup/shutdown and server errors

## Derived Metrics (5-minute windows)

Use delta/rate functions from your monitoring backend over a fixed window.

1. API error rate:
   - `api_error_rate_5m = delta(api_requests_errors_total) / max(delta(api_requests_total), 1)`
2. API average latency:
   - `api_avg_latency_ms_5m = delta(api_request_latency_ms_total) / max(delta(api_request_latency_samples_total), 1)`
3. WebSocket auth failure rate:
   - `ws_auth_failures_5m = delta(ws_auth_failures_total)`
4. WebSocket session init failures:
   - `ws_session_init_failures_5m = delta(ws_session_init_failures_total)`

## Dashboard Panels (Minimal)

1. API request volume (5m rate):
   - Source: `api_requests_total`
2. API error rate (%):
   - Source: derived `api_error_rate_5m`
3. API average latency (ms):
   - Source: derived `api_avg_latency_ms_5m`
4. API route error breakdown:
   - Source: `api_request_errors_by_route`
5. WebSocket active connections:
   - Source: `ws_connections_active`
6. WebSocket auth failures (5m):
   - Source: derived `ws_auth_failures_5m`
7. WebSocket session init failures (5m):
   - Source: derived `ws_session_init_failures_5m`
8. Worker refresh success/error log counts (5m):
   - Source: log query on refresh messages

## Alert Rules (V1 Baseline)

Severity levels:

- `P1`: user-visible outage or severe degradation
- `P2`: high risk, not total outage
- `P3`: warning/anomaly

1. `P1 API unavailable`
   - Condition: `/health` is non-200 for 3 consecutive minutes.
   - Notify: page on-call.

2. `P1 API high error rate`
   - Condition:
     - `delta(api_requests_total) >= 100` in 5m, and
     - `api_error_rate_5m >= 0.05` for 10m.
   - Notify: page on-call.

3. `P2 API latency regression`
   - Condition:
     - `delta(api_request_latency_samples_total) >= 100` in 5m, and
     - `api_avg_latency_ms_5m >= 750` for 15m.
   - Notify: on-call + team channel.

4. `P2 WebSocket auth failures spike`
   - Condition: `ws_auth_failures_5m >= 50` for 10m.
   - Notify: on-call + team channel.

5. `P2 WebSocket session init failures`
   - Condition: `ws_session_init_failures_5m >= 10` for 10m.
   - Notify: on-call + team channel.

6. `P1 Worker stalled`
   - Condition: no `refresh cycle completed` log events for 10m.
   - Notify: page on-call.

7. `P2 Worker persistent refresh failures`
   - Condition: `refresh cycle completed with errors` >= 5 events in 10m.
   - Notify: on-call + team channel.

## Alert Routing

1. `P1`: paging channel immediately.
2. `P2`: on-call notification + engineering incident channel.
3. `P3`: non-paging notification in engineering incident channel.

## Gate B Item 6 Completion Checklist

Operational baseline is considered complete when all items below are true:

1. Dashboard includes all eight panels above.
2. All seven alerts are configured with the listed thresholds.
3. Alerts route to on-call and incident channels.
4. `docs/ops-runbook.md` is linked from alert descriptions for responders.
5. `/debug/vars` signal presence is verified in staging and production using:
   - `backend/scripts/ops/verify-debug-vars.sh https://<asset-ws-host>/debug/vars`
