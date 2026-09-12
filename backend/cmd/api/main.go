package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/integration"
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

	agentConfig, err := integration.OpenAIConfigFromEnvironment(os.Getenv)
	if err != nil {
		logger.Error("load agent configuration", "error", err)
		os.Exit(1)
	}
	router, err := integration.NewRouter(database, logger, agentConfig, nil)
	if err != nil {
		logger.Error("initialize API routes", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("listen", "error", err, "address", address)
		os.Exit(1)
	}

	server := httpapi.NewServer(address, router)
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
