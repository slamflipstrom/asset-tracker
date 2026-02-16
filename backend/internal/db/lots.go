package db

import (
	"context"
	"time"
)

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

func (d *DB) UpdateLotForUser(ctx context.Context, userID string, lotID int64, quantity float64, unitCost float64, purchasedAt time.Time) (bool, error) {
	tag, err := d.pool.Exec(ctx, `
		update public.lots
		set quantity = $1, unit_cost = $2, purchased_at = $3
		where id = $4 and user_id = $5
	`, quantity, unitCost, purchasedAt, lotID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (d *DB) DeleteLot(ctx context.Context, userID string, lotID int64) error {
	_, err := d.pool.Exec(ctx, `
		delete from public.lots
		where id = $1 and user_id = $2
	`, lotID, userID)
	return err
}

func (d *DB) DeleteLotForUser(ctx context.Context, userID string, lotID int64) (bool, error) {
	tag, err := d.pool.Exec(ctx, `
		delete from public.lots
		where id = $1 and user_id = $2
	`, lotID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
