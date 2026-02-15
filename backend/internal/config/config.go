package config

import (
	"errors"
	"os"
	"strings"
)

type Mode string

const (
	ModeWorker Mode = "worker"
	ModeWS     Mode = "ws"
)

// Config holds service configuration shared by worker and WS server.
type Config struct {
	DatabaseURL           string
	SupabaseURL           string
	SupabaseSecretKey     string
	StockProviderAPIKey   string
	StockProviderName     string
	StockProviderBaseURL  string
	CryptoProviderAPIKey  string
	CryptoProviderName    string
	CryptoProviderBaseURL string
	Port                  string
}

func LoadForWorker() (Config, error) {
	return load(ModeWorker)
}

func LoadForWS() (Config, error) {
	return load(ModeWS)
}

func load(mode Mode) (Config, error) {
	cfg := Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		SupabaseURL:           os.Getenv("SUPABASE_URL"),
		SupabaseSecretKey:     os.Getenv("SUPABASE_SECRET_KEY"),
		StockProviderAPIKey:   os.Getenv("STOCK_PROVIDER_API_KEY"),
		StockProviderName:     os.Getenv("STOCK_PROVIDER_NAME"),
		StockProviderBaseURL:  os.Getenv("STOCK_PROVIDER_BASE_URL"),
		CryptoProviderAPIKey:  os.Getenv("CRYPTO_PROVIDER_API_KEY"),
		CryptoProviderName:    os.Getenv("CRYPTO_PROVIDER_NAME"),
		CryptoProviderBaseURL: os.Getenv("CRYPTO_PROVIDER_BASE_URL"),
		Port:                  envDefault("PORT", "8080"),
	}

	var validationErrs []string
	requireEnv("DATABASE_URL", cfg.DatabaseURL, &validationErrs)

	switch mode {
	case ModeWorker:
		requireEnv("CRYPTO_PROVIDER_NAME", cfg.CryptoProviderName, &validationErrs)
		requireEnv("CRYPTO_PROVIDER_API_KEY", cfg.CryptoProviderAPIKey, &validationErrs)
	case ModeWS:
		requireEnv("SUPABASE_URL", cfg.SupabaseURL, &validationErrs)
		requireEnv("SUPABASE_SECRET_KEY", cfg.SupabaseSecretKey, &validationErrs)
	default:
		validationErrs = append(validationErrs, "unknown service mode")
	}

	if len(validationErrs) > 0 {
		return cfg, errors.New(strings.Join(validationErrs, "; "))
	}

	return cfg, nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(name, value string, errs *[]string) {
	if strings.TrimSpace(value) == "" {
		*errs = append(*errs, name+" is required")
	}
}
