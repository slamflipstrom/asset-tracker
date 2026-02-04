package providers

import "context"

type AssetQuote struct {
	Symbol   string
	Price    float64
	Provider string
}

type StockProvider interface {
	FetchQuotes(ctx context.Context, symbols []string) ([]AssetQuote, error)
}

type CryptoProvider interface {
	FetchQuotes(ctx context.Context, symbols []string) ([]AssetQuote, error)
}
