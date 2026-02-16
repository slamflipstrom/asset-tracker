package main

import (
	"context"
	"log"
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
	cfg, err := config.LoadForWS()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	router := chi.NewRouter()
	database, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db init error: %v", err)
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
	router.Get("/ws", server.Handler())
	apiServer.Mount(router)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
}
