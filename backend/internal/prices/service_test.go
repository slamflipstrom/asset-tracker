package prices

import (
	"testing"
	"time"

	"asset-tracker/internal/db"
	"asset-tracker/internal/providers"
)

func TestClampInterval(t *testing.T) {
	t.Parallel()

	settings := db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600}

	if got := clampInterval(0, settings); got != 60 {
		t.Fatalf("expected 60 for zero, got %d", got)
	}
	if got := clampInterval(10, settings); got != 60 {
		t.Fatalf("expected min clamp 60, got %d", got)
	}
	if got := clampInterval(7200, settings); got != 3600 {
		t.Fatalf("expected max clamp 3600, got %d", got)
	}
	if got := clampInterval(300, settings); got != 300 {
		t.Fatalf("expected unchanged 300, got %d", got)
	}
}

func TestLookupKeyForAsset(t *testing.T) {
	t.Parallel()

	cryptoWithLookup := db.TrackedAsset{Type: "crypto", MarketDataID: "  BITCOIN  ", Symbol: "BTC"}
	if got := lookupKeyForAsset(cryptoWithLookup); got != "bitcoin" {
		t.Fatalf("expected bitcoin, got %q", got)
	}

	cryptoFallback := db.TrackedAsset{Type: "crypto", Symbol: "  Eth  "}
	if got := lookupKeyForAsset(cryptoFallback); got != "eth" {
		t.Fatalf("expected eth fallback, got %q", got)
	}

	cryptoWhitespaceLookup := db.TrackedAsset{Type: "crypto", MarketDataID: "   ", Symbol: "  SOL  "}
	if got := lookupKeyForAsset(cryptoWhitespaceLookup); got != "sol" {
		t.Fatalf("expected sol fallback when market_data_id is whitespace, got %q", got)
	}

	stock := db.TrackedAsset{Type: "stock", Symbol: "  AAPL  "}
	if got := lookupKeyForAsset(stock); got != "AAPL" {
		t.Fatalf("expected AAPL, got %q", got)
	}
}

func TestToUpdates(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	assets := map[string]dueAsset{
		"bitcoin": {TrackedAsset: db.TrackedAsset{ID: 101}},
	}
	quotes := []providers.AssetQuote{
		{LookupKey: "bitcoin", Price: 1000, Provider: "mobula"},
		{LookupKey: "unknown", Price: 2000, Provider: "mobula"},
	}

	updates := toUpdates(quotes, assets, now)
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].AssetID != 101 || updates[0].Price != 1000 || updates[0].Provider != "mobula" {
		t.Fatalf("unexpected update: %+v", updates[0])
	}
	if !updates[0].FetchedAt.Equal(now) {
		t.Fatalf("expected fetched_at %v, got %v", now, updates[0].FetchedAt)
	}
}

func TestReconcileDueAndPruneState(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	s := &Service{state: map[int64]*assetState{
		1: {interval: 60 * time.Second, nextDue: now.Add(-1 * time.Minute)},
		2: {interval: 60 * time.Second, nextDue: now.Add(10 * time.Minute)},
	}}

	settings := db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600}
	tracked := []db.TrackedAsset{
		{ID: 1, Type: "crypto", MarketDataID: "bitcoin", MinUserRefreshSec: 30},
		{ID: 3, Type: "crypto", MarketDataID: "ethereum", MinUserRefreshSec: 10},
	}

	due := s.reconcile(now, settings, tracked)
	if len(due) != 2 {
		t.Fatalf("expected 2 due assets, got %d", len(due))
	}

	if _, ok := s.state[2]; ok {
		t.Fatal("expected stale state for asset 2 to be pruned")
	}
	if _, ok := s.state[3]; !ok {
		t.Fatal("expected new state for asset 3")
	}
	if got := s.state[1].interval; got != 60*time.Second {
		t.Fatalf("expected clamped interval 60s for asset 1, got %v", got)
	}
}
