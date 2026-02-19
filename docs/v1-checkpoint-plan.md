# V1 Checkpoint Plan

**Checkpoint date:** February 15, 2026
**Assessment target:** Frontend + backend architecture readiness for v1 rollout

## 0. Progress Update (February 18, 2026)

Gate B progress snapshot:

1. Frontend reads/writes through Go API boundary: `PASS`
2. Server-side auth and validation for business actions: `PASS`
3. Realtime strategy finalized for v1: `PASS` (see `docs/realtime-v1-decision.md`)
4. Frontend smoke coverage (auth + lot CRUD + refresh): `PASS`
5. CI runs backend tests and frontend build on PRs: `PASS`
6. Operational baseline (error tracking, metrics, runbook): `FAIL`

Current production-readiness blocker: item 6 only (observability and incident operations).

## 1. Checkpoint Decision

- **Current decision:** `GO` for internal alpha/dogfooding.
- **Current decision:** `NO-GO` for broad production rollout.

Rationale:
- Core user flow exists (auth, lots CRUD, portfolio view, live updates via Supabase Realtime + polling fallback).
- Remaining critical production guardrails are primarily operational (error tracking, metrics, and runbook maturity).

## 2. V1 Scope Definition

In-scope for v1:
- Email/password auth.
- Portfolio positions view.
- Lots create/edit/delete.
- Near-live refresh behavior.
- Basic developer setup/run docs.

Out-of-scope for v1:
- Advanced analytics/tax reporting.
- Mobile-native app.
- Multi-tenant admin tooling.

## 3. Acceptance Gates

### Gate A: Internal Alpha (Must pass for team usage)

1. User can sign up/sign in/sign out from UI.
2. User can create, edit, and delete lots.
3. Positions render from current data without fatal UI errors.
4. Live updates function via Realtime, with polling fallback when Realtime is unavailable.
5. `pnpm build` succeeds for frontend and `go test ./...` succeeds for backend.
6. Local setup docs are sufficient for a new dev to run the stack.

**Status as of February 15, 2026:** `PASS`

### Gate B: Production Readiness (Must pass before wider launch)

1. Frontend reads/writes through a Go API boundary (not direct table/view access).
2. AuthZ and validation are enforced server-side for business actions.
3. Realtime strategy is finalized for scale/cost (Go WS fanout and/or constrained Supabase Realtime usage).
4. Automated frontend smoke coverage (auth + lots CRUD + live refresh).
5. CI checks run backend tests and frontend build on every PR.
6. Operational baseline exists (error tracking, request/latency metrics, connection metrics, runbooks).

**Status as of February 18, 2026:** `FAIL` (blocked by item 6)

## 4. Ordered Backlog

## P0 (Do next)

1. **Introduce Go API layer for portfolio and lots**
   - Add endpoints for:
     - `GET /api/positions`
     - `GET /api/lots`
     - `POST /api/lots`
     - `PATCH /api/lots/:id`
     - `DELETE /api/lots/:id`
     - `GET /api/assets/search`
   - Reuse existing DB query layer.
   - Validate inputs and return stable JSON contracts.

2. **Switch frontend data layer to Go API adapter**
   - Replace direct Supabase table/view calls for business operations.
   - Keep Supabase Auth session handling in frontend.

3. **Add contract tests for API routes**
   - Cover auth-required behavior, payload validation, and success/error responses.

## P1 (Immediately after P0)

1. **Define and implement realtime architecture decision**
   - Option A: Keep Supabase Realtime only for low-frequency updates.
   - Option B: Move high-frequency portfolio/price pushes to Go WS fanout.
   - Capture expected event schema and reconnection behavior.

2. **Frontend smoke tests (Playwright)**
   - Sign in flow.
   - Add/edit/delete lot flow.
   - Data refresh after update.

3. **CI baseline**
   - Backend: `go test ./...`
   - Frontend: `pnpm build`
   - Optional: smoke tests on merge queue/nightly.

## P2 (Hardening)

1. **Observability and ops**
   - Structured error logging.
   - Basic metrics dashboard (API latency/error rates, WS or Realtime connection health).
   - Incident runbook.

2. **UX resilience**
   - Better retry UX.
   - Distinguish auth/config/data errors in UI.
   - Accessibility pass on forms/tables.

## 5. Exit Criteria for V1 Launch

V1 can move from internal alpha to broader rollout only when all Gate B items are complete and validated in CI/staging.

## 6. Recommended Execution Sequence

1. P0 API boundary + frontend adapter.
2. P1 realtime decision and implementation.
3. P1 test/CI enforcement.
4. P2 hardening and launch checklist.
