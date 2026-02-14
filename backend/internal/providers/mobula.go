package providers

import (
	"bytes"
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
	Data      json.RawMessage `json:"data"`
	DataArray json.RawMessage `json:"dataArray"`
}

type mobulaAssetData struct {
	Key   string          `json:"key"`
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
	keys := normalizeMobulaLookupKeys(lookupKeys)
	if len(keys) == 0 {
		return nil, nil
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("mobula api key is not set")
	}

	numericIDs, assetNames := partitionMobulaLookupKeys(keys)

	allRows := make([]mobulaAssetData, 0, len(keys))
	if len(numericIDs) > 0 {
		rows, err := p.fetchRows(ctx, "ids", numericIDs)
		if err != nil {
			return nil, err
		}
		allRows = append(allRows, rows...)
	}
	if len(assetNames) > 0 {
		rows, err := p.fetchRows(ctx, "assets", assetNames)
		if err != nil {
			return nil, err
		}
		allRows = append(allRows, rows...)
	}
	quoteByLookup := make(map[string]AssetQuote, len(allRows))
	for _, row := range allRows {
		lookupKey := strings.TrimSpace(row.Key)
		if lookupKey == "" {
			id, err := parseMobulaID(row.ID)
			if err != nil || id == "" {
				continue
			}
			lookupKey = id
		}
		lookupKey = normalizeMobulaLookupKey(lookupKey)
		if lookupKey == "" {
			continue
		}
		quoteByLookup[lookupKey] = AssetQuote{
			LookupKey: lookupKey,
			Price:     row.Price,
			Provider:  "mobula",
		}
	}

	quotes := make([]AssetQuote, 0, len(quoteByLookup))
	for _, key := range keys {
		quote, ok := quoteByLookup[key]
		if !ok {
			continue
		}
		quotes = append(quotes, quote)
		delete(quoteByLookup, key)
	}
	for _, quote := range quoteByLookup {
		quotes = append(quotes, quote)
	}

	return quotes, nil
}

func (p *MobulaProvider) fetchRows(ctx context.Context, queryParam string, lookupKeys []string) ([]mobulaAssetData, error) {
	endpoint, err := url.Parse(p.baseURL + "/api/1/market/multi-data")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set(queryParam, strings.Join(lookupKeys, ","))
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

	rows, err := parseMobulaRows(payload.Data)
	if err != nil && hasJSONContent(payload.Data) {
		return nil, err
	}
	if len(rows) == 0 {
		rows, err = parseMobulaRows(payload.DataArray)
		if err != nil && hasJSONContent(payload.DataArray) {
			return nil, err
		}
	}
	return rows, nil
}

func parseMobulaRows(raw json.RawMessage) ([]mobulaAssetData, error) {
	if !hasJSONContent(raw) {
		return nil, nil
	}

	var rows []mobulaAssetData
	if err := json.Unmarshal(raw, &rows); err == nil {
		return rows, nil
	}

	var single mobulaAssetData
	if err := json.Unmarshal(raw, &single); err == nil && (len(single.ID) > 0 || strings.TrimSpace(single.Key) != "") {
		return []mobulaAssetData{single}, nil
	}

	var keyed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keyed); err != nil {
		return nil, fmt.Errorf("unsupported mobula data shape")
	}

	rows = make([]mobulaAssetData, 0, len(keyed))
	for key, value := range keyed {
		row, ok := parseMobulaKeyedRow(key, value)
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseMobulaKeyedRow(key string, raw json.RawMessage) (mobulaAssetData, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return mobulaAssetData{}, false
	}

	var row mobulaAssetData
	if err := json.Unmarshal(raw, &row); err == nil {
		if row.Key == "" {
			row.Key = key
		}
		if len(row.ID) == 0 {
			row.ID = json.RawMessage(strconv.Quote(key))
		}
		return row, true
	}

	var price float64
	if err := json.Unmarshal(raw, &price); err == nil {
		return mobulaAssetData{
			Key:   key,
			ID:    json.RawMessage(strconv.Quote(key)),
			Price: price,
		}, true
	}

	return mobulaAssetData{}, false
}

func hasJSONContent(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

func normalizeMobulaLookupKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = normalizeMobulaLookupKey(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func normalizeMobulaLookupKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if isMobulaNumericID(key) {
		return key
	}
	return strings.ToLower(key)
}

func partitionMobulaLookupKeys(keys []string) ([]string, []string) {
	ids := make([]string, 0, len(keys))
	assets := make([]string, 0, len(keys))
	for _, key := range keys {
		if isMobulaNumericID(key) {
			ids = append(ids, key)
			continue
		}
		assets = append(assets, key)
	}
	return ids, assets
}

func isMobulaNumericID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
