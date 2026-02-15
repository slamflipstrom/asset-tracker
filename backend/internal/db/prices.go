package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

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
