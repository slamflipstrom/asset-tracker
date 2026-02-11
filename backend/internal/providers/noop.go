package providers

import (
	"context"
	"fmt"
)

type MissingProvider struct {
	Name string
}

func NewMissingProvider(name string) MissingProvider {
	return MissingProvider{Name: name}
}

func (p MissingProvider) FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error) {
	if len(lookupKeys) == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf("%s provider not configured", p.Name)
}
