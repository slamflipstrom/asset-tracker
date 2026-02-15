package db

import (
	"database/sql"
	"time"
)

type AssetType string

const (
	AssetTypeCrypto AssetType = "crypto"
	AssetTypeStock  AssetType = "stock"
)

type Asset struct {
	ID               int64
	Symbol           string
	MarketDataID     string
	LookupBlockchain string
	LookupAddress    string
	Type             AssetType
	Name             string
}

type AppSettings struct {
	MinRefreshIntervalSec int
	MaxRefreshIntervalSec int
}

type UserSettings struct {
	UserID             string
	RefreshIntervalSec int
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	ID                int64
	Symbol            string
	MarketDataID      string
	LookupBlockchain  string
	LookupAddress     string
	Type              AssetType
	MinUserRefreshSec int
}

type PriceUpdate struct {
	AssetID   int64
	Price     float64
	FetchedAt time.Time
	Provider  string
}

type Position struct {
	UserID       string
	AssetID      int64
	TotalQty     float64
	AvgCost      float64
	CurrentPrice sql.NullFloat64
	UnrealizedPL sql.NullFloat64
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
