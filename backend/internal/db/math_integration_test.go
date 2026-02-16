package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

const defaultIntegrationDBURL = "postgresql://postgres:postgres@127.0.0.1:54322/postgres"

func TestCostBasisAndPLViews(t *testing.T) {
	t.Helper()

	database := mustOpenIntegrationDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	primaryUserID := randomUUID(t)
	secondaryUserID := randomUUID(t)

	mustInsertAuthUser(t, ctx, database, primaryUserID, "math-primary@example.com")
	mustInsertAuthUser(t, ctx, database, secondaryUserID, "math-secondary@example.com")
	defer cleanupAuthUser(t, context.Background(), database, primaryUserID)
	defer cleanupAuthUser(t, context.Background(), database, secondaryUserID)

	pricedAssetID := mustInsertStockAsset(t, ctx, database, "MATHP", "Math Priced")
	unpricedAssetID := mustInsertStockAsset(t, ctx, database, "MATHU", "Math Unpriced")
	otherUserAssetID := mustInsertStockAsset(t, ctx, database, "MATHO", "Math Other User")
	defer cleanupAsset(t, context.Background(), database, pricedAssetID)
	defer cleanupAsset(t, context.Background(), database, unpricedAssetID)
	defer cleanupAsset(t, context.Background(), database, otherUserAssetID)

	pricedLot1ID := mustInsertLot(t, ctx, database, primaryUserID, pricedAssetID, 2, 100)
	pricedLot2ID := mustInsertLot(t, ctx, database, primaryUserID, pricedAssetID, 1, 160)
	unpricedLotID := mustInsertLot(t, ctx, database, primaryUserID, unpricedAssetID, 5, 20)
	_ = mustInsertLot(t, ctx, database, secondaryUserID, otherUserAssetID, 7, 10)

	mustUpsertCurrentPrice(t, ctx, database, pricedAssetID, 150)
	defer cleanupCurrentPrice(t, context.Background(), database, pricedAssetID)

	positions, err := database.FetchPositionsForUser(ctx, primaryUserID)
	if err != nil {
		t.Fatalf("FetchPositionsForUser failed: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}

	positionByAsset := make(map[int64]Position, len(positions))
	for _, position := range positions {
		positionByAsset[position.AssetID] = position
	}

	pricedPosition, ok := positionByAsset[pricedAssetID]
	if !ok {
		t.Fatalf("missing priced position for asset %d", pricedAssetID)
	}
	assertApproxEqual(t, pricedPosition.TotalQty, 3, "priced total_qty")
	assertApproxEqual(t, pricedPosition.AvgCost, 120, "priced avg_cost")
	if !pricedPosition.CurrentPrice.Valid {
		t.Fatal("expected priced position current_price to be valid")
	}
	assertApproxEqual(t, pricedPosition.CurrentPrice.Float64, 150, "priced current_price")
	if !pricedPosition.UnrealizedPL.Valid {
		t.Fatal("expected priced position unrealized_pl to be valid")
	}
	assertApproxEqual(t, pricedPosition.UnrealizedPL.Float64, 90, "priced unrealized_pl")

	unpricedPosition, ok := positionByAsset[unpricedAssetID]
	if !ok {
		t.Fatalf("missing unpriced position for asset %d", unpricedAssetID)
	}
	assertApproxEqual(t, unpricedPosition.TotalQty, 5, "unpriced total_qty")
	assertApproxEqual(t, unpricedPosition.AvgCost, 20, "unpriced avg_cost")
	if unpricedPosition.CurrentPrice.Valid {
		t.Fatalf("expected unpriced current_price null, got %f", unpricedPosition.CurrentPrice.Float64)
	}
	if unpricedPosition.UnrealizedPL.Valid {
		t.Fatalf("expected unpriced unrealized_pl null, got %f", unpricedPosition.UnrealizedPL.Float64)
	}

	lotPerformance, err := database.FetchLotPerformance(ctx, primaryUserID, nil)
	if err != nil {
		t.Fatalf("FetchLotPerformance failed: %v", err)
	}
	if len(lotPerformance) != 3 {
		t.Fatalf("expected 3 lot performance rows, got %d", len(lotPerformance))
	}

	lotByID := make(map[int64]LotPerformance, len(lotPerformance))
	for _, lot := range lotPerformance {
		lotByID[lot.LotID] = lot
	}

	lot1, ok := lotByID[pricedLot1ID]
	if !ok {
		t.Fatalf("missing lot performance row for lot %d", pricedLot1ID)
	}
	if !lot1.CurrentPrice.Valid || !lot1.UnrealizedPL.Valid {
		t.Fatal("expected lot1 to have current_price and unrealized_pl")
	}
	assertApproxEqual(t, lot1.CurrentPrice.Float64, 150, "lot1 current_price")
	assertApproxEqual(t, lot1.UnrealizedPL.Float64, 100, "lot1 unrealized_pl")

	lot2, ok := lotByID[pricedLot2ID]
	if !ok {
		t.Fatalf("missing lot performance row for lot %d", pricedLot2ID)
	}
	if !lot2.CurrentPrice.Valid || !lot2.UnrealizedPL.Valid {
		t.Fatal("expected lot2 to have current_price and unrealized_pl")
	}
	assertApproxEqual(t, lot2.CurrentPrice.Float64, 150, "lot2 current_price")
	assertApproxEqual(t, lot2.UnrealizedPL.Float64, -10, "lot2 unrealized_pl")

	lot3, ok := lotByID[unpricedLotID]
	if !ok {
		t.Fatalf("missing lot performance row for lot %d", unpricedLotID)
	}
	if lot3.CurrentPrice.Valid {
		t.Fatalf("expected lot3 current_price null, got %f", lot3.CurrentPrice.Float64)
	}
	if lot3.UnrealizedPL.Valid {
		t.Fatalf("expected lot3 unrealized_pl null, got %f", lot3.UnrealizedPL.Float64)
	}
}

func mustOpenIntegrationDB(t *testing.T) *DB {
	t.Helper()

	dbURL := os.Getenv("ASSET_TRACKER_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultIntegrationDBURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	database, err := New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping DB integration test: %v", err)
	}
	if err := database.pool.Ping(ctx); err != nil {
		database.Close()
		t.Skipf("skipping DB integration test: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

func mustInsertAuthUser(t *testing.T, ctx context.Context, database *DB, userID string, email string) {
	t.Helper()
	_, err := database.pool.Exec(ctx, `
		insert into auth.users (id, email, aud, role, encrypted_password, created_at, updated_at, is_sso_user, is_anonymous)
		values ($1::uuid, $2, 'authenticated', 'authenticated', '', now(), now(), false, false)
	`, userID, email)
	if err != nil {
		t.Fatalf("failed to insert auth user: %v", err)
	}
}

func cleanupAuthUser(t *testing.T, ctx context.Context, database *DB, userID string) {
	t.Helper()
	_, _ = database.pool.Exec(ctx, `delete from auth.users where id = $1::uuid`, userID)
}

func mustInsertStockAsset(t *testing.T, ctx context.Context, database *DB, prefix string, name string) int64 {
	t.Helper()
	suffix := randomHex(t, 4)
	symbol := fmt.Sprintf("%s_%s", prefix, suffix)

	var assetID int64
	err := database.pool.QueryRow(ctx, `
		insert into public.assets (symbol, type, name)
		values ($1, 'stock', $2)
		returning id
	`, symbol, name).Scan(&assetID)
	if err != nil {
		t.Fatalf("failed to insert asset: %v", err)
	}
	return assetID
}

func cleanupAsset(t *testing.T, ctx context.Context, database *DB, assetID int64) {
	t.Helper()
	_, _ = database.pool.Exec(ctx, `delete from public.assets where id = $1`, assetID)
}

func mustInsertLot(t *testing.T, ctx context.Context, database *DB, userID string, assetID int64, quantity float64, unitCost float64) int64 {
	t.Helper()

	var lotID int64
	err := database.pool.QueryRow(ctx, `
		insert into public.lots (user_id, asset_id, quantity, unit_cost, purchased_at)
		values ($1::uuid, $2, $3, $4, now())
		returning id
	`, userID, assetID, quantity, unitCost).Scan(&lotID)
	if err != nil {
		t.Fatalf("failed to insert lot: %v", err)
	}
	return lotID
}

func mustUpsertCurrentPrice(t *testing.T, ctx context.Context, database *DB, assetID int64, price float64) {
	t.Helper()
	_, err := database.pool.Exec(ctx, `
		insert into public.prices_current (asset_id, price, fetched_at, provider)
		values ($1, $2, now(), 'test')
		on conflict (asset_id)
		do update set price = excluded.price, fetched_at = excluded.fetched_at, provider = excluded.provider
	`, assetID, price)
	if err != nil {
		t.Fatalf("failed to upsert current price: %v", err)
	}
}

func cleanupCurrentPrice(t *testing.T, ctx context.Context, database *DB, assetID int64) {
	t.Helper()
	_, _ = database.pool.Exec(ctx, `delete from public.prices_current where asset_id = $1`, assetID)
}

func assertApproxEqual(t *testing.T, got float64, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s mismatch: got %.12f want %.12f", label, got, want)
	}
}

func randomUUID(t *testing.T) string {
	t.Helper()
	b := randomBytes(t, 16)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	return fmt.Sprintf("%x", randomBytes(t, n))
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("failed to read random bytes: %v", err)
	}
	return b
}
