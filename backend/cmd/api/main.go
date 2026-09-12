package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/mistysya/hackathon_2026/backend/internal/campaign"
	"github.com/mistysya/hackathon_2026/backend/internal/employee"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/profile"
	"github.com/mistysya/hackathon_2026/backend/internal/roleb"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
	"github.com/mistysya/hackathon_2026/backend/internal/structured"
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

	repository := store.New(database)
	validator, err := structured.NewValidator()
	if err != nil {
		logger.Error("initialize structured validator", "error", err)
		os.Exit(1)
	}
	profileAdapter, err := profile.NewFixtureAdapter()
	if err != nil {
		logger.Error("initialize profile fixture adapter", "error", err)
		os.Exit(1)
	}
	campaignAgent, err := campaign.NewFixtureScenarioAgent()
	if err != nil {
		logger.Error("initialize campaign fixture agent", "error", err)
		os.Exit(1)
	}

	employeeRoutes := employee.NewRoutes(employee.NewService(repository), logger)
	profileRoutes := profile.NewRoutes(profile.NewService(repository, profileAdapter, profile.NewFixtureAgent(), validator), logger)
	campaignRoutes := campaign.NewRoutes(campaign.NewService(repository, repository, campaignAgent, validator, campaign.NewCryptoIDGenerator()), logger)
	roleBRoutes := roleb.NewHandler(roleb.NewRepository(database))

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Error("listen", "error", err, "address", address)
		os.Exit(1)
	}

	server := httpapi.NewServer(address, httpapi.NewRouter(logger, employeeRoutes, profileRoutes, campaignRoutes, roleBRoutes))
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
