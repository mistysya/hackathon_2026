// Package integration owns the production composition root. Feature packages
// expose narrow adapters and route registrars; this package decides how they
// are assembled without leaking provider configuration into HTTP handlers.
package integration

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mistysya/hackathon_2026/backend/internal/campaign"
	"github.com/mistysya/hackathon_2026/backend/internal/employee"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/profile"
	"github.com/mistysya/hackathon_2026/backend/internal/roleb"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
	"github.com/mistysya/hackathon_2026/backend/internal/structured"
)

// NewRouter builds every frozen endpoint from one shared database connection.
// httpClient is injectable so integration tests never need an API credential.
func NewRouter(database *sql.DB, logger *slog.Logger, agentConfig openaiapi.Config, httpClient *http.Client) (http.Handler, error) {
	if database == nil {
		return nil, fmt.Errorf("database is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	config, err := agentConfig.Normalized()
	if err != nil {
		return nil, fmt.Errorf("agent configuration: %w", err)
	}
	validator, err := structured.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("initialize structured validator: %w", err)
	}
	fixtureAdapter, err := profile.NewFixtureAdapter()
	if err != nil {
		return nil, fmt.Errorf("initialize profile fixture adapter: %w", err)
	}
	fixtureProfileAgent := profile.NewFixtureAgent()
	fixtureScenarioAgent, err := campaign.NewFixtureScenarioAgent()
	if err != nil {
		return nil, fmt.Errorf("initialize campaign fixture agent: %w", err)
	}

	var profileAdapter ports.EnrichmentAdapter = fixtureAdapter
	var profileAgent ports.ProfileAgent = fixtureProfileAgent
	var scenarioAgent ports.ScenarioAgent = fixtureScenarioAgent
	if config.Mode == openaiapi.ModeLiveRequired {
		if !config.LiveReady() {
			return nil, fmt.Errorf("initialize live OpenAI client: API key and model are required")
		}
		client, err := openaiapi.NewHTTPClient(config, httpClient)
		if err != nil {
			return nil, fmt.Errorf("initialize live OpenAI client: %w", err)
		}
		profileAdapter = profile.NewOpenAIEnrichmentAdapter(client)
		profileAgent = profile.NewOpenAIProfileAgent(client)
		scenarioAgent = campaign.NewOpenAIScenarioAgent(client)
	} else if config.Mode == openaiapi.ModeAuto && config.LiveReady() {
		client, clientErr := openaiapi.NewHTTPClient(config, httpClient)
		if clientErr == nil {
			profileAdapter = profile.WithEnrichmentFallback(config.Mode, profile.NewOpenAIEnrichmentAdapter(client), fixtureAdapter)
			profileAgent = profile.WithProfileFallback(config.Mode, profile.NewOpenAIProfileAgent(client), fixtureProfileAgent)
			scenarioAgent = campaign.WithScenarioFallback(config.Mode, campaign.NewOpenAIScenarioAgent(client), fixtureScenarioAgent)
		}
	}
	logger.Info("agent pipeline configured", "selectedMode", config.Mode, "liveConfigured", config.LiveReady(), "model", config.Model)

	repository := store.New(database)
	employeeRoutes := employee.NewRoutes(employee.NewService(repository), logger)
	profileRoutes := profile.NewRoutes(profile.NewService(repository, profileAdapter, profileAgent, validator), logger)
	campaignRoutes := campaign.NewRoutes(campaign.NewService(repository, repository, scenarioAgent, validator, campaign.NewCryptoIDGenerator()), logger)
	roleBRoutes := roleb.NewHandler(roleb.NewRepository(database))
	return httpapi.NewRouter(logger, employeeRoutes, profileRoutes, campaignRoutes, roleBRoutes), nil
}
