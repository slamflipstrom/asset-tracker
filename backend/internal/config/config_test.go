package config

import (
	"strings"
	"testing"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"DATABASE_URL",
		"SUPABASE_URL",
		"SUPABASE_SECRET_KEY",
		"STOCK_PROVIDER_API_KEY",
		"STOCK_PROVIDER_NAME",
		"STOCK_PROVIDER_BASE_URL",
		"CRYPTO_PROVIDER_API_KEY",
		"CRYPTO_PROVIDER_NAME",
		"CRYPTO_PROVIDER_BASE_URL",
		"PORT",
	} {
		t.Setenv(key, "")
	}
}

func TestLoadForWSSuccess(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://db")
	t.Setenv("SUPABASE_URL", "https://supabase.example.com")
	t.Setenv("SUPABASE_SECRET_KEY", "service-key")
	t.Setenv("PORT", "9090")

	cfg, err := LoadForWS()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.DatabaseURL != "postgresql://db" {
		t.Fatalf("unexpected DATABASE_URL: %q", cfg.DatabaseURL)
	}
	if cfg.SupabaseURL != "https://supabase.example.com" {
		t.Fatalf("unexpected SUPABASE_URL: %q", cfg.SupabaseURL)
	}
	if cfg.SupabaseSecretKey != "service-key" {
		t.Fatalf("unexpected SUPABASE_SECRET_KEY: %q", cfg.SupabaseSecretKey)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %q", cfg.Port)
	}
}

func TestLoadForWSValidation(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://db")

	_, err := LoadForWS()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "SUPABASE_URL is required") || !strings.Contains(err.Error(), "SUPABASE_SECRET_KEY is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLoadForWorkerSuccessUsesDefaultPort(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://db")
	t.Setenv("CRYPTO_PROVIDER_NAME", "mobula")
	t.Setenv("CRYPTO_PROVIDER_API_KEY", "key")

	cfg, err := LoadForWorker()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
}

func TestLoadForWorkerValidation(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://db")

	_, err := LoadForWorker()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "CRYPTO_PROVIDER_NAME is required") || !strings.Contains(err.Error(), "CRYPTO_PROVIDER_API_KEY is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestLoadUnknownMode(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgresql://db")

	_, err := load(Mode("invalid-mode"))
	if err == nil {
		t.Fatal("expected unknown mode error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown service mode") {
		t.Fatalf("unexpected error for unknown mode: %v", err)
	}
}
