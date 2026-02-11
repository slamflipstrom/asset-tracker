package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const mobulaDefaultBaseURL = "https://api.mobula.io"

type MobulaProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type mobulaMultiDataResponse struct {
	Data      []mobulaAssetData `json:"data"`
	DataArray []mobulaAssetData `json:"dataArray"`
}

type mobulaAssetData struct {
	ID    json.RawMessage `json:"id"`
	Price float64         `json:"price"`
}

func NewMobulaProvider(baseURL, apiKey string) *MobulaProvider {
	resolvedBaseURL := strings.TrimRight(baseURL, "/")
	if resolvedBaseURL == "" {
		resolvedBaseURL = mobulaDefaultBaseURL
	}

	return &MobulaProvider{
		baseURL: resolvedBaseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *MobulaProvider) FetchQuotes(ctx context.Context, lookupKeys []string) ([]AssetQuote, error) {
	ids := normalizeMobulaIDs(lookupKeys)
	if len(ids) == 0 {
		return nil, nil
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("mobula api key is not set")
	}

	endpoint, err := url.Parse(p.baseURL + "/api/1/market/multi-data")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("ids", strings.Join(ids, ","))
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("mobula error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload mobulaMultiDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	rows := payload.Data
	if len(rows) == 0 {
		rows = payload.DataArray
	}

	quotes := make([]AssetQuote, 0, len(rows))
	for _, row := range rows {
		id, err := parseMobulaID(row.ID)
		if err != nil || id == "" {
			continue
		}
		quotes = append(quotes, AssetQuote{
			LookupKey: id,
			Price:     row.Price,
			Provider:  "mobula",
		})
	}

	return quotes, nil
}

func normalizeMobulaIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
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

func parseMobulaID(raw json.RawMessage) (string, error) {
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str), nil
	}

	var intID int64
	if err := json.Unmarshal(raw, &intID); err == nil {
		return strconv.FormatInt(intID, 10), nil
	}

	var floatID float64
	if err := json.Unmarshal(raw, &floatID); err == nil {
		if floatID == float64(int64(floatID)) {
			return strconv.FormatInt(int64(floatID), 10), nil
		}
		return strconv.FormatFloat(floatID, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("invalid mobula id")
}
