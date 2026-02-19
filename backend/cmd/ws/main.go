package main

import (
	"context"
	"expvar"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"asset-tracker/internal/api"
	"asset-tracker/internal/auth"
	"asset-tracker/internal/config"
	"asset-tracker/internal/db"
	"asset-tracker/internal/ws"
	"github.com/go-chi/chi/v5"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.LoadForWS()
	if err != nil {
		slog.Error("failed to load ws config", "error", err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	database, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	hub := ws.NewHub()
	verifier := auth.NewSupabaseVerifier(cfg.SupabaseURL, cfg.SupabaseSecretKey)
	server := ws.NewServer(hub, verifier)
	apiServer := api.NewServer(database, verifier)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	router.Handle("/debug/vars", expvar.Handler())
	router.Get("/ws", server.Handler())
	apiServer.Mount(router)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()

	slog.Info("ws/api server started", "port", cfg.Port)

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErrCh:
		slog.Error("ws/api server terminated unexpectedly", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("ws/api server stopped")
}
