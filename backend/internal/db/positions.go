package db

import "context"

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
		and ($2::bigint is null or asset_id = $2::bigint)
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
