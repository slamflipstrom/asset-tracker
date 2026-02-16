package db

import "context"

func (d *DB) ListAssetsByIDs(ctx context.Context, ids []int64) ([]Asset, error) {
	if len(ids) == 0 {
		return []Asset{}, nil
	}

	rows, err := d.pool.Query(ctx, `
		select id, symbol, coalesce(market_data_id, ''), coalesce(lookup_blockchain, ''), coalesce(lookup_address, ''), type, name
		from public.assets
		where id = any($1::bigint[])
		order by symbol
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.MarketDataID, &asset.LookupBlockchain, &asset.LookupAddress, &asset.Type, &asset.Name); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (d *DB) FetchTrackedAssets(ctx context.Context) ([]TrackedAsset, error) {
	rows, err := d.pool.Query(ctx, `
		select a.id, a.symbol, coalesce(a.market_data_id, ''), coalesce(a.lookup_blockchain, ''), coalesce(a.lookup_address, ''), a.type, min(us.refresh_interval_sec) as min_refresh_interval_sec
		from public.assets a
		join public.lots l on l.asset_id = a.id
		join public.user_settings us on us.user_id = l.user_id
		group by a.id, a.symbol, a.market_data_id, a.lookup_blockchain, a.lookup_address, a.type
		order by a.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []TrackedAsset
	for rows.Next() {
		var asset TrackedAsset
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.MarketDataID, &asset.LookupBlockchain, &asset.LookupAddress, &asset.Type, &asset.MinUserRefreshSec); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (d *DB) SearchAssets(ctx context.Context, query string, assetType string, limit int) ([]Asset, error) {
	rows, err := d.pool.Query(ctx, `
		select id, symbol, coalesce(market_data_id, ''), coalesce(lookup_blockchain, ''), coalesce(lookup_address, ''), type, name
		from public.assets
		where (symbol ilike $1 or name ilike $1 or coalesce(market_data_id, '') ilike $1)
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
		if err := rows.Scan(&asset.ID, &asset.Symbol, &asset.MarketDataID, &asset.LookupBlockchain, &asset.LookupAddress, &asset.Type, &asset.Name); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}
