package prices

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"asset-tracker/internal/db"
	"asset-tracker/internal/providers"
)

type mockStore struct {
	settings    db.AppSettings
	settingsErr error

	tracked    []db.TrackedAsset
	trackedErr error

	upsertErr       error
	upsertCalls     int
	upsertedUpdates []db.PriceUpdate

	snapshotErr       error
	snapshotCalls     int
	snapshottedUpdate []db.PriceUpdate
}

func (m *mockStore) FetchAppSettings(ctx context.Context) (db.AppSettings, error) {
	if m.settingsErr != nil {
		return db.AppSettings{}, m.settingsErr
	}
	return m.settings, nil
}

func (m *mockStore) FetchTrackedAssets(ctx context.Context) ([]db.TrackedAsset, error) {
	if m.trackedErr != nil {
		return nil, m.trackedErr
	}

	out := make([]db.TrackedAsset, len(m.tracked))
	copy(out, m.tracked)
	return out, nil
}

func (m *mockStore) UpsertCurrentPrices(ctx context.Context, updates []db.PriceUpdate) error {
	m.upsertCalls++
	m.upsertedUpdates = append([]db.PriceUpdate(nil), updates...)
	return m.upsertErr
}

func (m *mockStore) InsertPriceSnapshots(ctx context.Context, updates []db.PriceUpdate) error {
	m.snapshotCalls++
	m.snapshottedUpdate = append([]db.PriceUpdate(nil), updates...)
	return m.snapshotErr
}

type mockQuoteProvider struct {
	quotes []providers.AssetQuote
	err    error
	calls  [][]string
}

func (m *mockQuoteProvider) FetchQuotes(ctx context.Context, lookupKeys []string) ([]providers.AssetQuote, error) {
	copiedKeys := make([]string, len(lookupKeys))
	copy(copiedKeys, lookupKeys)
	m.calls = append(m.calls, copiedKeys)

	if m.err != nil {
		return nil, m.err
	}

	out := make([]providers.AssetQuote, len(m.quotes))
	copy(out, m.quotes)
	return out, nil
}

func sortedStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

func TestRefreshSuccessWritesPricesAndAdvancesSchedule(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		settings: db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600},
		tracked: []db.TrackedAsset{
			{ID: 1, Type: db.AssetTypeStock, Symbol: "AAPL", MinUserRefreshSec: 120},
			{ID: 2, Type: db.AssetTypeCrypto, Symbol: "BTC", MarketDataID: "bitcoin", MinUserRefreshSec: 30},
		},
	}
	stock := &mockQuoteProvider{
		quotes: []providers.AssetQuote{{LookupKey: "AAPL", Price: 190, Provider: "stock-test"}},
	}
	crypto := &mockQuoteProvider{
		quotes: []providers.AssetQuote{{LookupKey: "bitcoin", Price: 50000, Provider: "crypto-test"}},
	}

	svc := NewService(store, stock, crypto)
	start := time.Now().UTC()

	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if store.upsertCalls != 1 || store.snapshotCalls != 1 {
		t.Fatalf("expected one write to each table, got upsert=%d snapshots=%d", store.upsertCalls, store.snapshotCalls)
	}
	if len(store.upsertedUpdates) != 2 || len(store.snapshottedUpdate) != 2 {
		t.Fatalf("expected 2 updates written, got upsert=%d snapshots=%d", len(store.upsertedUpdates), len(store.snapshottedUpdate))
	}

	gotStockKeys := sortedStrings(stock.calls[0])
	if len(gotStockKeys) != 1 || gotStockKeys[0] != "AAPL" {
		t.Fatalf("unexpected stock lookup keys: %v", gotStockKeys)
	}
	gotCryptoKeys := sortedStrings(crypto.calls[0])
	if len(gotCryptoKeys) != 1 || gotCryptoKeys[0] != "bitcoin" {
		t.Fatalf("unexpected crypto lookup keys: %v", gotCryptoKeys)
	}

	if svc.state[1].interval != 120*time.Second {
		t.Fatalf("unexpected stock interval: %v", svc.state[1].interval)
	}
	if svc.state[2].interval != 60*time.Second {
		t.Fatalf("unexpected crypto interval clamp: %v", svc.state[2].interval)
	}
	if !svc.state[1].nextDue.After(start) || !svc.state[2].nextDue.After(start) {
		t.Fatalf("expected nextDue to advance beyond start, got stock=%v crypto=%v", svc.state[1].nextDue, svc.state[2].nextDue)
	}
}

func TestRefreshPartialProviderFailureStillWritesAvailableQuotes(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		settings: db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600},
		tracked: []db.TrackedAsset{
			{ID: 1, Type: db.AssetTypeStock, Symbol: "AAPL", MinUserRefreshSec: 120},
			{ID: 2, Type: db.AssetTypeCrypto, Symbol: "BTC", MarketDataID: "bitcoin", MinUserRefreshSec: 60},
		},
	}
	stock := &mockQuoteProvider{err: errors.New("stock provider unavailable")}
	crypto := &mockQuoteProvider{
		quotes: []providers.AssetQuote{{LookupKey: "bitcoin", Price: 50000, Provider: "crypto-test"}},
	}

	svc := NewService(store, stock, crypto)

	err := svc.Refresh(context.Background())
	if err == nil || !strings.Contains(err.Error(), "stock provider unavailable") {
		t.Fatalf("expected stock error in result, got %v", err)
	}

	if store.upsertCalls != 1 || store.snapshotCalls != 1 {
		t.Fatalf("expected writes with partial data, got upsert=%d snapshots=%d", store.upsertCalls, store.snapshotCalls)
	}
	if len(store.upsertedUpdates) != 1 || store.upsertedUpdates[0].AssetID != 2 {
		t.Fatalf("expected only crypto update, got %+v", store.upsertedUpdates)
	}
}

func TestRefreshProviderFailuresWithoutQuotesReturnsErrorAndSkipsWrites(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		settings: db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600},
		tracked: []db.TrackedAsset{
			{ID: 1, Type: db.AssetTypeStock, Symbol: "AAPL", MinUserRefreshSec: 120},
			{ID: 2, Type: db.AssetTypeCrypto, Symbol: "BTC", MarketDataID: "bitcoin", MinUserRefreshSec: 60},
		},
	}
	stock := &mockQuoteProvider{err: errors.New("stock failed")}
	crypto := &mockQuoteProvider{err: errors.New("crypto failed")}

	svc := NewService(store, stock, crypto)

	err := svc.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected combined provider error, got nil")
	}
	if !strings.Contains(err.Error(), "stock failed") || !strings.Contains(err.Error(), "crypto failed") {
		t.Fatalf("expected joined provider errors, got %v", err)
	}
	if store.upsertCalls != 0 || store.snapshotCalls != 0 {
		t.Fatalf("expected no writes when there are no quotes, got upsert=%d snapshots=%d", store.upsertCalls, store.snapshotCalls)
	}
}

func TestRefreshDBWriteErrorsAreJoined(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		settings: db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600},
		tracked: []db.TrackedAsset{
			{ID: 1, Type: db.AssetTypeStock, Symbol: "AAPL", MinUserRefreshSec: 120},
		},
		upsertErr:   errors.New("upsert failed"),
		snapshotErr: errors.New("snapshot failed"),
	}
	stock := &mockQuoteProvider{
		quotes: []providers.AssetQuote{{LookupKey: "AAPL", Price: 190, Provider: "stock-test"}},
	}
	crypto := &mockQuoteProvider{}

	svc := NewService(store, stock, crypto)

	err := svc.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected DB write error, got nil")
	}
	if !strings.Contains(err.Error(), "upsert failed") || !strings.Contains(err.Error(), "snapshot failed") {
		t.Fatalf("expected both DB errors, got %v", err)
	}
	if store.upsertCalls != 1 || store.snapshotCalls != 1 {
		t.Fatalf("expected both write paths to be attempted, got upsert=%d snapshots=%d", store.upsertCalls, store.snapshotCalls)
	}
}

func TestRefreshSkipsProvidersWhenNothingIsDue(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	store := &mockStore{
		settings: db.AppSettings{MinRefreshIntervalSec: 60, MaxRefreshIntervalSec: 3600},
		tracked:  []db.TrackedAsset{{ID: 1, Type: db.AssetTypeStock, Symbol: "AAPL", MinUserRefreshSec: 120}},
	}
	stock := &mockQuoteProvider{}
	crypto := &mockQuoteProvider{}

	svc := NewService(store, stock, crypto)
	svc.state[1] = &assetState{
		interval: 120 * time.Second,
		nextDue:  now.Add(5 * time.Minute),
	}

	if err := svc.Refresh(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(stock.calls) != 0 || len(crypto.calls) != 0 {
		t.Fatalf("expected no provider calls when nothing is due, got stock=%d crypto=%d", len(stock.calls), len(crypto.calls))
	}
	if store.upsertCalls != 0 || store.snapshotCalls != 0 {
		t.Fatalf("expected no writes when nothing is due, got upsert=%d snapshots=%d", store.upsertCalls, store.snapshotCalls)
	}
}

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
