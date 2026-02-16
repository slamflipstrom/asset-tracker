# API v1 (WS Service)

Base path: `/api/v1`

Auth:
- All routes require `Authorization: Bearer <supabase_access_token>`.
- Token is validated against Supabase `/auth/v1/user`.

## GET /positions

Returns the authenticated user's position rows.

```json
[
  {
    "asset_id": 1,
    "symbol": "BTC",
    "name": "Bitcoin",
    "type": "crypto",
    "total_qty": 0.5,
    "avg_cost": 40000,
    "current_price": 45000,
    "unrealized_pl": 2500
  }
]
```

## GET /lots

Returns the authenticated user's lots.

```json
[
  {
    "id": 10,
    "asset_id": 1,
    "symbol": "BTC",
    "name": "Bitcoin",
    "type": "crypto",
    "quantity": 0.25,
    "unit_cost": 38000,
    "purchased_at": "2026-02-15T00:00:00Z"
  }
]
```

## POST /lots

Creates a lot for the authenticated user.

Request body:

```json
{
  "asset_id": 1,
  "quantity": 0.25,
  "unit_cost": 38000,
  "purchased_at": "2026-02-15T00:00:00Z"
}
```

`purchased_at` accepts RFC3339 or `YYYY-MM-DD`.

Response (`201`):

```json
{ "id": 10 }
```

## PATCH /lots/{lotID}

Updates quantity, unit cost, and purchase date for a lot belonging to the authenticated user.

Request body:

```json
{
  "quantity": 0.3,
  "unit_cost": 39000,
  "purchased_at": "2026-02-16"
}
```

Response: `204 No Content`

## DELETE /lots/{lotID}

Deletes a lot belonging to the authenticated user.

Response: `204 No Content`

## GET /assets/search

Query params:
- `q` (optional): search term
- `type` (optional): `crypto` or `stock`
- `limit` (optional): positive integer, max 100, default 20

Response:

```json
[
  {
    "id": 1,
    "symbol": "BTC",
    "name": "Bitcoin",
    "type": "crypto"
  }
]
```

## Error format

```json
{ "error": "message" }
```
