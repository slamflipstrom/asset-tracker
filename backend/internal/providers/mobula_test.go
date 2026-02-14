package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMobulaProviderFetchQuotes_DataArray(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Fatalf("expected authorization header test-key, got %q", got)
		}
		if got := r.URL.Query().Get("ids"); got != "1,2" {
			t.Fatalf("expected ids query 1,2, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dataArray":[{"id":1,"price":88.8},{"id":"2","price":99}]}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"1", "2", "1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(quotes))
	}
	if quotes[0].LookupKey != "1" || quotes[0].Price != 88.8 || quotes[0].Provider != "mobula" {
		t.Fatalf("unexpected first quote: %+v", quotes[0])
	}
	if quotes[1].LookupKey != "2" || quotes[1].Price != 99 {
		t.Fatalf("unexpected second quote: %+v", quotes[1])
	}
}

func TestMobulaProviderFetchQuotes_DataArrayPrefersKeyOverID(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("assets"); got != "bitcoin" {
			t.Fatalf("expected assets query bitcoin, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dataArray":[{"key":"bitcoin","id":1,"price":88.8}]}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].LookupKey != "bitcoin" {
		t.Fatalf("expected lookup key bitcoin, got %+v", quotes[0])
	}
}

func TestMobulaProviderFetchQuotes_Data(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("assets"); got != "bitcoin" {
			t.Fatalf("expected assets query bitcoin, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"bitcoin","price":123.45}]}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].LookupKey != "bitcoin" || quotes[0].Price != 123.45 {
		t.Fatalf("unexpected quote: %+v", quotes[0])
	}
}

func TestMobulaProviderFetchQuotes_NormalizesKeyCase(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"key":"Bitcoin","price":123.45}]}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].LookupKey != "bitcoin" {
		t.Fatalf("expected lookup key bitcoin, got %q", quotes[0].LookupKey)
	}
}

func TestMobulaProviderFetchQuotes_DataObject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"bitcoin","price":123.45}}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	if quotes[0].LookupKey != "bitcoin" || quotes[0].Price != 123.45 {
		t.Fatalf("unexpected quote: %+v", quotes[0])
	}
}

func TestMobulaProviderFetchQuotes_DataKeyedObject(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"bitcoin":{"price":123.45},"ethereum":456.78}}`))
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	quotes, err := p.FetchQuotes(context.Background(), []string{"bitcoin", "ethereum"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(quotes))
	}

	got := map[string]float64{}
	for _, q := range quotes {
		got[q.LookupKey] = q.Price
	}
	if got["bitcoin"] != 123.45 {
		t.Fatalf("expected bitcoin price 123.45, got %v", got["bitcoin"])
	}
	if got["ethereum"] != 456.78 {
		t.Fatalf("expected ethereum price 456.78, got %v", got["ethereum"])
	}
}

func TestMobulaProviderFetchQuotes_Non200(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream exploded", http.StatusBadGateway)
	}))
	defer ts.Close()

	p := NewMobulaProvider(ts.URL, "test-key")
	_, err := p.FetchQuotes(context.Background(), []string{"1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "upstream exploded") {
		t.Fatalf("expected response body in error, got %v", err)
	}
}

func TestMobulaProviderFetchQuotes_MissingAPIKey(t *testing.T) {
	t.Parallel()

	p := NewMobulaProvider("https://api.mobula.io", "")
	_, err := p.FetchQuotes(context.Background(), []string{"1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api key") {
		t.Fatalf("expected api key error, got %v", err)
	}
}
