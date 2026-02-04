package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
)

type Asset struct {
	ID     int64
	Symbol string
	Type   string
	Name   string
}

type AppSettings struct {
	MinRefreshIntervalSec int
	MaxRefreshIntervalSec int
}

type UserSettings struct {
	UserID              string
	RefreshIntervalSec  int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Lot struct {
	ID          int64
	UserID      string
	AssetID     int64
	Quantity    float64
	UnitCost    float64
	PurchasedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TrackedAsset struct {
	ID                   int64
	Symbol               string
	Type                 string
	MinUserRefreshSec    int
}

type PriceUpdate struct {
	AssetID   int64
	Price     float64
	FetchedAt time.Time
	Provider  string
}

type Position struct {
	UserID        string
	AssetID       int64
	TotalQty      float64
	AvgCost       float64
	CurrentPrice  sql.NullFloat64
	UnrealizedPL  sql.NullFloat64
}

type LotPerformance struct {
	LotID        int64
	UserID       string
	AssetID      int64
	Quantity     float64
	UnitCost     float64
	PurchasedAt  time.Time
	CurrentPrice sql.NullFloat64
	UnrealizedPL sql.NullFloat64
}

func (d *DB) FetchAppSettings(ctx context.Context) (AppSettings, error) {
	row := d.pool.QueryRow(ctx, `
		select min_refresh_interval_sec, max_refresh_interval_sec
		from public.app_settings
		where id = 1
	`)
	var settings AppSettings
	if err := row.Scan(&settings.MinRefreshIntervalSec, &settings.MaxRefreshIntervalSec); err != nil {
		return settings, err
	}
	return settings, nil
}

func (d *DB) FetchUserSettings(ctx context.Context, userID string) (UserSettings, error) {
	row := d.pool.QueryRow(ctx, `
		select user_id, refresh_interval_sec, created_at, updated_at
		from public.user_settings
		where user_id = $1
	`, userID)
	var settings UserSettings
	if err := row.Scan(&settings.UserID, &settings.RefreshIntervalSec, &settings.CreatedAt, &settings.UpdatedAt); err != nil {
		return settings, err
	}
	return settings, nil
}

func (d *DB) FetchTrackedAssets(ctx context.Context) ([]TrackedAsset, error) {
	rows, err := d.pool.Query(ctx, `
		select a.id, a.symbol, a.type, min(us.refresh_interval_sec) as min_refresh_interval_sec
		from public.assets a
		join public.lots l on l.asset_id = a.id
		join public.user_settings us on us.user_id = l.user_id
		group by a.id, a.symbol, a.type
		order by a.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []TrackedAsset
	for rows.Next() {
		var asset TrackedAsset
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.Type, &asset.MinUserRefreshSec); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (d *DB) SearchAssets(ctx context.Context, query string, assetType string, limit int) ([]Asset, error) {
	rows, err := d.pool.Query(ctx, `
		select id, symbol, type, name
		from public.assets
		where (symbol ilike $1 or name ilike $1)
		and (case when $2 = '' then true else type = $2::public.asset_type end)
		order by symbol
		limit $3
	`, "%"+query+"%", assetType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.Type, &asset.Name); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (d *DB) ListLotsByUser(ctx context.Context, userID string) ([]Lot, error) {
	rows, err := d.pool.Query(ctx, `
		select id, user_id, asset_id, quantity, unit_cost, purchased_at, created_at, updated_at
		from public.lots
		where user_id = $1
		order by purchased_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []Lot
	for rows.Next() {
		var lot Lot
		if err := rows.Scan(&lot.ID, &lot.UserID, &lot.AssetID, &lot.Quantity, &lot.UnitCost, &lot.PurchasedAt, &lot.CreatedAt, &lot.UpdatedAt); err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}
	return lots, rows.Err()
}

func (d *DB) ListLotsByUserAsset(ctx context.Context, userID string, assetID int64) ([]Lot, error) {
	rows, err := d.pool.Query(ctx, `
		select id, user_id, asset_id, quantity, unit_cost, purchased_at, created_at, updated_at
		from public.lots
		where user_id = $1 and asset_id = $2
		order by purchased_at desc
	`, userID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []Lot
	for rows.Next() {
		var lot Lot
		if err := rows.Scan(&lot.ID, &lot.UserID, &lot.AssetID, &lot.Quantity, &lot.UnitCost, &lot.PurchasedAt, &lot.CreatedAt, &lot.UpdatedAt); err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}
	return lots, rows.Err()
}

func (d *DB) InsertLot(ctx context.Context, lot Lot) (int64, error) {
	row := d.pool.QueryRow(ctx, `
		insert into public.lots (user_id, asset_id, quantity, unit_cost, purchased_at)
		values ($1, $2, $3, $4, $5)
		returning id
	`, lot.UserID, lot.AssetID, lot.Quantity, lot.UnitCost, lot.PurchasedAt)

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (d *DB) UpdateLot(ctx context.Context, lot Lot) error {
	_, err := d.pool.Exec(ctx, `
		update public.lots
		set quantity = $1, unit_cost = $2, purchased_at = $3
		where id = $4 and user_id = $5
	`, lot.Quantity, lot.UnitCost, lot.PurchasedAt, lot.ID, lot.UserID)
	return err
}

func (d *DB) DeleteLot(ctx context.Context, userID string, lotID int64) error {
	_, err := d.pool.Exec(ctx, `
		delete from public.lots
		where id = $1 and user_id = $2
	`, lotID, userID)
	return err
}

func (d *DB) UpsertCurrentPrices(ctx context.Context, updates []PriceUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, update := range updates {
		batch.Queue(`
			insert into public.prices_current (asset_id, price, fetched_at, provider)
			values ($1, $2, $3, $4)
			on conflict (asset_id)
			do update set price = excluded.price, fetched_at = excluded.fetched_at, provider = excluded.provider
		`, update.AssetID, update.Price, update.FetchedAt, update.Provider)
	}
	br := d.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range updates {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) InsertPriceSnapshots(ctx context.Context, updates []PriceUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, update := range updates {
		batch.Queue(`
			insert into public.price_snapshots (asset_id, price, fetched_at, provider)
			values ($1, $2, $3, $4)
		`, update.AssetID, update.Price, update.FetchedAt, update.Provider)
	}
	br := d.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range updates {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) FetchPositionsForUser(ctx context.Context, userID string) ([]Position, error) {
	rows, err := d.pool.Query(ctx, `
		select user_id, asset_id, total_qty, avg_cost, current_price, unrealized_pl
		from public.positions_view
		where user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		if err := rows.Scan(&pos.UserID, &pos.AssetID, &pos.TotalQty, &pos.AvgCost, &pos.CurrentPrice, &pos.UnrealizedPL); err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

func (d *DB) FetchLotPerformance(ctx context.Context, userID string, assetID *int64) ([]LotPerformance, error) {
	rows, err := d.pool.Query(ctx, `
		select lot_id, user_id, asset_id, quantity, unit_cost, purchased_at, current_price, unrealized_pl
		from public.lot_performance_view
		where user_id = $1
		and ($2 is null or asset_id = $2)
		order by purchased_at desc
	`, userID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lots []LotPerformance
	for rows.Next() {
		var lot LotPerformance
		if err := rows.Scan(&lot.LotID, &lot.UserID, &lot.AssetID, &lot.Quantity, &lot.UnitCost, &lot.PurchasedAt, &lot.CurrentPrice, &lot.UnrealizedPL); err != nil {
			return nil, err
		}
		lots = append(lots, lot)
	}
	return lots, rows.Err()
}
