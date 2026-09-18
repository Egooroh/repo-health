package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/httpapi"
	"repo-health/internal/sourcecraft"
	"repo-health/internal/storage"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Repo Health остановлен", "error", err)
		os.Exit(1)
	}
}
func run() error {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store, err := storage.Open(env("DATABASE_PATH", "data/repo-health.db"))
	if err != nil {
		return err
	}
	defer store.Close()
	token := strings.TrimSpace(os.Getenv("SOURCECRAFT_TOKEN"))
	service := app.New(store, sourcecraft.New(token), demo.Collector{}, log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workerDone := make(chan error, 1)
	go func() { workerDone <- service.Run(ctx) }()
	handler, closeSessions := httpapi.NewWithSessions(ctx, service, log, token != "")
	defer closeSessions()
	server := &http.Server{Addr: env("HTTP_ADDR", "127.0.0.1:8080"), Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()
	log.Info("Repo Health запущен", "address", server.Addr, "sourcecraft_token_configured", token != "")
	var result error
	workerStopped := false
	select {
	case <-ctx.Done():
	case result = <-serverDone:
	case result = <-workerDone:
		workerStopped = true
	}
	stop()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		server.Close()
		if result == nil {
			result = err
		}
	}
	if !workerStopped {
		if err := <-workerDone; result == nil {
			result = err
		}
	}
	if errors.Is(result, http.ErrServerClosed) {
		return nil
	}
	return result
}
func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
