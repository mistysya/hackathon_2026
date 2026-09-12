package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	address := envOrDefault("HTTP_ADDR", ":8080")
	databaseDSN := envOrDefault("DATABASE_DSN", store.DefaultDSN)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	database, err := store.Open(ctx, databaseDSN)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	if err := store.Migrate(ctx, database); err != nil {
		logger.Error("migrate database", "error", err)
		os.Exit(1)
	}

	// The repository is the shared foundation for future route registrars.
	// Business routes are intentionally not registered in this phase.
	_ = store.New(database)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("listen", "error", err, "address", address)
		os.Exit(1)
	}

	server := httpapi.NewServer(address, httpapi.NewRouter(logger))
	logger.Info("api listening", "address", listener.Addr().String())
	if err := httpapi.Serve(ctx, server, listener, httpapi.DefaultShutdownTimeout); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
