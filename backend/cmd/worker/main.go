package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"asset-tracker/internal/config"
	"asset-tracker/internal/db"
	"asset-tracker/internal/prices"
	"asset-tracker/internal/providers"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.LoadForWorker()
	if err != nil {
		slog.Error("failed to load worker config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	providerSet := providers.NewFromConfig(cfg)
	service := prices.NewService(database, providerSet.Stock, providerSet.Crypto)

	scheduler := prices.NewScheduler(30*time.Second, func(ctx context.Context) error {
		if err := service.Refresh(ctx); err != nil {
			slog.Error("worker refresh cycle failed", "error", err)
		}
		return nil
	})

	slog.Info("worker started", "interval_seconds", 30)
	if err := scheduler.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("worker stopped unexpectedly", "error", err)
		os.Exit(1)
	}
	slog.Info("worker stopped")
}
