package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	coinGeckoPublicBaseURL = "https://api.coingecko.com/api/v3"
	coinGeckoProBaseURL    = "https://pro-api.coingecko.com/api/v3"
)

type CoinGeckoProvider struct {
	baseURL      string
	apiKey       string
	apiKeyHeader string
	client       *http.Client
	vsCurrency   string
}

func NewCoinGeckoProvider(baseURL, apiKey string) *CoinGeckoProvider {
	resolvedBaseURL := strings.TrimRight(baseURL, "/")
	if resolvedBaseURL == "" {
		resolvedBaseURL = coinGeckoPublicBaseURL
	}

	header := "x-cg-demo-api-key"
	if strings.Contains(resolvedBaseURL, "pro-api.coingecko.com") {
		header = "x-cg-pro-api-key"
	}

	return &CoinGeckoProvider{
		baseURL:      resolvedBaseURL,
		apiKey:       apiKey,
		apiKeyHeader: header,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		vsCurrency: "usd",
	}
}

func (p *CoinGeckoProvider) FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error) {
	ids := normalizeIDs(lookupKeys)
	if len(ids) == 0 {
		return nil, nil
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("coingecko api key is not set")
	}

	endpoint, err := url.Parse(p.baseURL + "/simple/price")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("ids", strings.Join(ids, ","))
	query.Set("vs_currencies", p.vsCurrency)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set(p.apiKeyHeader, p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("coingecko error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	quotes := make([]AssetQuote, 0, len(payload))
	for id, values := range payload {
		price, ok := values[p.vsCurrency]
		if !ok {
			continue
		}
		quotes = append(quotes, AssetQuote{
			LookupKey: id,
			Price:     price,
			Provider:  "coingecko",
		})
	}

	return quotes, nil
}

func normalizeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(strings.ToLower(id))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func CoinGeckoDefaultBaseURL(plan string) string {
	if strings.EqualFold(plan, "pro") {
		return coinGeckoProBaseURL
	}
	return coinGeckoPublicBaseURL
}
