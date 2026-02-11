package prices

import (
	"context"
	"errors"
	"strings"
	"time"

	"asset-tracker/internal/db"
	"asset-tracker/internal/providers"
)

type Service struct {
	db     *db.DB
	stock  providers.StockProvider
	crypto providers.CryptoProvider
	state  map[int64]*assetState
}

type assetState struct {
	interval time.Duration
	nextDue  time.Time
}

type dueAsset struct {
	db.TrackedAsset
	interval time.Duration
}

func NewService(database *db.DB, stock providers.StockProvider, crypto providers.CryptoProvider) *Service {
	return &Service{
		db:     database,
		stock:  stock,
		crypto: crypto,
		state:  make(map[int64]*assetState),
	}
}

func (s *Service) Refresh(ctx context.Context) error {
	now := time.Now().UTC()

	settings, err := s.db.FetchAppSettings(ctx)
	if err != nil {
		return err
	}

	tracked, err := s.db.FetchTrackedAssets(ctx)
	if err != nil {
		return err
	}

	dueAssets := s.reconcile(now, settings, tracked)
	if len(dueAssets) == 0 {
		return nil
	}

	stockMap := map[string]dueAsset{}
	cryptoMap := map[string]dueAsset{}
	for _, asset := range dueAssets {
		key := lookupKeyForAsset(asset.TrackedAsset)
		if key == "" {
			continue
		}
		switch asset.Type {
		case "stock":
			stockMap[key] = asset
		case "crypto":
			cryptoMap[key] = asset
		}
	}

	var errs []error
	updates := make([]db.PriceUpdate, 0, len(dueAssets))

	if len(stockMap) > 0 {
		quotes, err := s.stock.FetchQuotes(ctx, keys(stockMap))
		if err != nil {
			errs = append(errs, err)
		} else {
			updates = append(updates, toUpdates(quotes, stockMap, now)...)
		}
	}

	if len(cryptoMap) > 0 {
		quotes, err := s.crypto.FetchQuotes(ctx, keys(cryptoMap))
		if err != nil {
			errs = append(errs, err)
		} else {
			updates = append(updates, toUpdates(quotes, cryptoMap, now)...)
		}
	}

	if len(updates) == 0 {
		return errors.Join(errs...)
	}

	if err := s.db.UpsertCurrentPrices(ctx, updates); err != nil {
		errs = append(errs, err)
	}
	if err := s.db.InsertPriceSnapshots(ctx, updates); err != nil {
		errs = append(errs, err)
	}

	for _, update := range updates {
		if state, ok := s.state[update.AssetID]; ok {
			state.nextDue = now.Add(state.interval)
		}
	}

	return errors.Join(errs...)
}

func (s *Service) reconcile(now time.Time, settings db.AppSettings, tracked []db.TrackedAsset) []dueAsset {
	seen := make(map[int64]struct{}, len(tracked))
	var due []dueAsset

	for _, asset := range tracked {
		seen[asset.ID] = struct{}{}
		intervalSec := clampInterval(asset.MinUserRefreshSec, settings)
		interval := time.Duration(intervalSec) * time.Second

		state, ok := s.state[asset.ID]
		if !ok {
			state = &assetState{interval: interval, nextDue: now}
			s.state[asset.ID] = state
		} else if state.interval != interval {
			state.interval = interval
		}

		if !now.Before(state.nextDue) {
			due = append(due, dueAsset{TrackedAsset: asset, interval: interval})
		}
	}

	for assetID := range s.state {
		if _, ok := seen[assetID]; !ok {
			delete(s.state, assetID)
		}
	}

	return due
}

func clampInterval(value int, settings db.AppSettings) int {
	interval := value
	if interval <= 0 {
		interval = settings.MinRefreshIntervalSec
	}
	if interval < settings.MinRefreshIntervalSec {
		interval = settings.MinRefreshIntervalSec
	}
	if interval > settings.MaxRefreshIntervalSec {
		interval = settings.MaxRefreshIntervalSec
	}
	return interval
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func toUpdates(quotes []providers.AssetQuote, assets map[string]dueAsset, fetchedAt time.Time) []db.PriceUpdate {
	updates := make([]db.PriceUpdate, 0, len(quotes))
	for _, quote := range quotes {
		asset, ok := assets[quote.LookupKey]
		if !ok {
			continue
		}
		updates = append(updates, db.PriceUpdate{
			AssetID:   asset.ID,
			Price:     quote.Price,
			FetchedAt: fetchedAt,
			Provider:  quote.Provider,
		})
	}
	return updates
}

func lookupKeyForAsset(asset db.TrackedAsset) string {
	switch asset.Type {
	case "crypto":
		if asset.MarketDataID != "" {
			return strings.ToLower(strings.TrimSpace(asset.MarketDataID))
		}
		return strings.ToLower(strings.TrimSpace(asset.Symbol))
	default:
		return strings.TrimSpace(asset.Symbol)
	}
}
