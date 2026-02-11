package providers

import "context"

type AssetQuote struct {
	LookupKey string
	Price     float64
	Provider  string
}

type StockProvider interface {
	FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error)
}

type CryptoProvider interface {
	FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error)
}
