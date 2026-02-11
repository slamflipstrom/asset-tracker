package providers

import (
	"strings"

	"asset-tracker/internal/config"
)

type ProviderSet struct {
	Stock  StockProvider
	Crypto CryptoProvider
}

func NewFromConfig(cfg config.Config) ProviderSet {
	return ProviderSet{
		Stock:  buildStock(cfg),
		Crypto: buildCrypto(cfg),
	}
}

func buildStock(cfg config.Config) StockProvider {
	name := strings.TrimSpace(strings.ToLower(cfg.StockProviderName))
	if name == "" {
		return NewMissingProvider("stock")
	}

	switch name {
	case "http":
		return NewHTTPProvider("stock", cfg.StockProviderBaseURL, cfg.StockProviderAPIKey)
	default:
		return NewMissingProvider("stock")
	}
}

func buildCrypto(cfg config.Config) CryptoProvider {
	name := strings.TrimSpace(strings.ToLower(cfg.CryptoProviderName))
	if name == "" {
		return NewMissingProvider("crypto")
	}

	switch name {
	case "mobula":
		baseURL := cfg.CryptoProviderBaseURL
		if baseURL == "" {
			baseURL = mobulaDefaultBaseURL
		}
		return NewMobulaProvider(baseURL, cfg.CryptoProviderAPIKey)
	case "coingecko":
		baseURL := cfg.CryptoProviderBaseURL
		if baseURL == "" {
			baseURL = CoinGeckoDefaultBaseURL("public")
		}
		return NewCoinGeckoProvider(baseURL, cfg.CryptoProviderAPIKey)
	case "coingecko-pro":
		baseURL := cfg.CryptoProviderBaseURL
		if baseURL == "" {
			baseURL = CoinGeckoDefaultBaseURL("pro")
		}
		return NewCoinGeckoProvider(baseURL, cfg.CryptoProviderAPIKey)
	case "http":
		return NewHTTPProvider("crypto", cfg.CryptoProviderBaseURL, cfg.CryptoProviderAPIKey)
	default:
		return NewMissingProvider("crypto")
	}
}
