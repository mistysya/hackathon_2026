package integration

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
)

// OpenAIConfigFromEnvironment parses only server-side environment values. It
// returns errors without including the API key, so callers can safely log them.
func OpenAIConfigFromEnvironment(getenv func(string) string) (openaiapi.Config, error) {
	if getenv == nil {
		return openaiapi.Config{}, fmt.Errorf("environment reader is required")
	}
	timeout, err := time.ParseDuration(orDefault(getenv("OPENAI_TIMEOUT"), "20s"))
	if err != nil {
		return openaiapi.Config{}, fmt.Errorf("OPENAI_TIMEOUT: %w", err)
	}
	retries, err := strconv.Atoi(orDefault(getenv("OPENAI_MAX_RETRIES"), "1"))
	if err != nil {
		return openaiapi.Config{}, fmt.Errorf("OPENAI_MAX_RETRIES: %w", err)
	}
	return (openaiapi.Config{
		Mode:       openaiapi.Mode(orDefault(getenv("AGENT_MODE"), string(openaiapi.ModeAuto))),
		APIKey:     getenv("OPENAI_API_KEY"),
		Model:      orDefault(getenv("OPENAI_MODEL"), "gpt-4o-mini"),
		BaseURL:    orDefault(getenv("OPENAI_BASE_URL"), "https://api.openai.com/v1"),
		Timeout:    timeout,
		MaxRetries: retries,
	}).Normalized()
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
