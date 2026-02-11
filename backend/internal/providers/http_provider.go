package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HTTPProvider struct {
	kind    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewHTTPProvider(kind, baseURL, apiKey string) *HTTPProvider {
	return &HTTPProvider{
		kind:    kind,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *HTTPProvider) FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error) {
	if len(lookupKeys) == 0 {
		return nil, nil
	}
	if p.baseURL == "" {
		return nil, fmt.Errorf("%s provider base URL is not set", p.kind)
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("%s provider API key is not set", p.kind)
	}
	return nil, errors.New("http provider not implemented yet")
}
