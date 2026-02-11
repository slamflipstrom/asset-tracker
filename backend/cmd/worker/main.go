package main

import (
	"context"
	"log"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	defer database.Close()

	providerSet := providers.NewFromConfig(cfg)
	service := prices.NewService(database, providerSet.Stock, providerSet.Crypto)

	scheduler := prices.NewScheduler(30*time.Second, func(ctx context.Context) error {
		if err := service.Refresh(ctx); err != nil {
			log.Printf("refresh error: %v", err)
		}
		return nil
	})

	if err := scheduler.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("worker stopped: %v", err)
	}
}
