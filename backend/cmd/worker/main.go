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

	scheduler := prices.NewScheduler(5*time.Minute, func(ctx context.Context) error {
		// TODO: load assets, compute effective refresh, poll providers, write prices.
		return nil
	})

	if err := scheduler.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("worker stopped: %v", err)
	}
}
