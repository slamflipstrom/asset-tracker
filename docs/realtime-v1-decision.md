# Realtime Architecture Decision (V1)

**Decision date:** February 18, 2026  
**Status:** Accepted for v1 rollout

## Decision

For v1, live portfolio updates use Supabase Realtime subscriptions in the frontend with polling fallback. Go WebSocket fanout is not the primary realtime delivery path for v1.

## Scope

- Frontend subscribes to `postgres_changes` on `public.lots` (filtered by `user_id`) and `public.prices_current`.
- Realtime events trigger debounced refreshes of `/api/v1/positions` and `/api/v1/lots`.
- Polling fallback remains enabled through `VITE_REFRESH_MS` (default `30000` ms).

## Why This Decision

- It matches the implemented and validated frontend behavior.
- It minimizes launch risk by avoiding a second primary realtime transport during v1 rollout.
- It keeps the system simpler while observability and incident operations are still being hardened.

## Deferred to Post-V1

- Go WebSocket fanout as the primary push channel for price/position updates.
- WS delivery guarantees and event-contract hardening for high-frequency streams.

## Revisit Triggers

Revisit this decision when one or more conditions are true:

- Realtime reliability issues become a recurring cause of stale portfolio views.
- Polling + Realtime load materially increases query or egress cost.
- Product requirements need lower-latency pushes than the current refresh model can provide.

## Implementation Notes

- `/api/v1` remains the source of truth for frontend-rendered portfolio state.
- `/ws` remains available for future iteration but is not required for v1 portfolio freshness.
