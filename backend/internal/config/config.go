package config

import (
	"errors"
	"os"
	"strings"
)

// Config holds service configuration for both worker and WS server.
type Config struct {
	DatabaseURL           string
	SupabaseURL           string
	SupabaseServiceKey    string
	SupabaseJWKSURL       string
	StockProviderAPIKey   string
	StockProviderName     string
	StockProviderBaseURL  string
	CryptoProviderAPIKey  string
	CryptoProviderName    string
	CryptoProviderBaseURL string
	WSAllowedOrigins      []string
	Port                  string
	LogLevel              string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		SupabaseURL:           os.Getenv("SUPABASE_URL"),
		SupabaseServiceKey:    os.Getenv("SUPABASE_SECRET_KEY"),
		SupabaseJWKSURL:       os.Getenv("SUPABASE_JWKS_URL"),
		StockProviderAPIKey:   os.Getenv("STOCK_PROVIDER_API_KEY"),
		StockProviderName:     os.Getenv("STOCK_PROVIDER_NAME"),
		StockProviderBaseURL:  os.Getenv("STOCK_PROVIDER_BASE_URL"),
		CryptoProviderAPIKey:  os.Getenv("CRYPTO_PROVIDER_API_KEY"),
		CryptoProviderName:    os.Getenv("CRYPTO_PROVIDER_NAME"),
		CryptoProviderBaseURL: os.Getenv("CRYPTO_PROVIDER_BASE_URL"),
		Port:                  envDefault("PORT", "8080"),
		LogLevel:              envDefault("LOG_LEVEL", "info"),
		WSAllowedOrigins:      splitCSV(os.Getenv("WS_ALLOWED_ORIGINS")),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
