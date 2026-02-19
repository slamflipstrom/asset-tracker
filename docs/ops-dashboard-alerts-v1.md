# Ops Monitoring Spec (V1 Lean Baseline)

**Last updated:** February 19, 2026
**Status:** Solo-friendly baseline

## Goal

Use the smallest monitoring setup that still catches real outages and data staleness early.

## Signals to Collect

1. `GET /health` from `asset-ws`.
2. `GET /debug/vars` from `asset-ws`.
3. Structured logs from `asset-ws` and `asset-worker`.

## Required Metrics

From `/debug/vars`:

- `api_requests_total`
- `api_requests_errors_total`
- `api_request_latency_ms_total`
- `api_request_latency_samples_total`

From logs:

- `asset-worker`: `refresh cycle completed`
- `asset-worker`: `refresh cycle completed with errors`

## Derived Metrics

1. API error rate (10m):
   - `api_error_rate_10m = delta(api_requests_errors_total) / max(delta(api_requests_total), 1)`
2. API average latency (10m):
   - `api_avg_latency_ms_10m = delta(api_request_latency_ms_total) / max(delta(api_request_latency_samples_total), 1)`

## Lean Dashboard Panels

1. API request volume (10m rate).
2. API error rate (%).
3. API average latency (ms).
4. Worker refresh success and error counts (from logs).

## Lean Alerts

1. API down:
   - `/health` non-200 for 3 consecutive minutes.
2. API error rate too high:
   - `delta(api_requests_total) >= 20` in 10m and `api_error_rate_10m >= 0.05` for 10m.
3. Worker stalled:
   - no `refresh cycle completed` log events for 15m.

Optional warning alert:

- Worker noisy failures:
  - `refresh cycle completed with errors` >= 5 in 15m.

## Deferred Until Growth

1. Per-route error dashboards.
2. WS-specific alerting (`ws_auth_failures_total`, `ws_session_init_failures_total`, `ws_connections_active`).
3. Multi-severity paging model (`P1/P2/P3`).
4. Broad alert matrix and incident automation.

## Completion Checklist (Gate B Item 6)

1. One dashboard exists with the four lean panels.
2. Three lean alerts are configured.
3. Alerts deliver to your primary notification channel.
4. Metrics keys are verified in staging and production with:
   - `backend/scripts/ops/verify-debug-vars.sh https://<asset-ws-host>/debug/vars`
