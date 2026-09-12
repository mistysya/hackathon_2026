// Package openaiapi isolates the Responses API wire contract from domain code.
package openaiapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Mode string

const (
	ModeAuto         Mode = "auto"
	ModeFixture      Mode = "fixture"
	ModeLiveRequired Mode = "live_required"
)

// Config is deliberately free of logging helpers: API keys must never be
// accidentally formatted into application logs.
type Config struct {
	Mode            Mode
	APIKey          string
	Model           string
	BaseURL         string
	Timeout         time.Duration
	MaxRetries      int
	MaxOutputTokens int
}

func (c Config) Normalized() (Config, error) {
	c.Mode = Mode(strings.TrimSpace(string(c.Mode)))
	if c.Mode == "" {
		c.Mode = ModeAuto
	}
	if c.Mode != ModeAuto && c.Mode != ModeFixture && c.Mode != ModeLiveRequired {
		return Config{}, fmt.Errorf("%w: unknown mode", ErrConfiguration)
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com/v1"
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.Host == "" {
		return Config{}, fmt.Errorf("%w: invalid base URL", ErrConfiguration)
	}
	if c.Timeout == 0 {
		c.Timeout = 20 * time.Second
	}
	if c.Timeout <= 0 {
		return Config{}, fmt.Errorf("%w: timeout must be positive", ErrConfiguration)
	}
	if c.MaxRetries < 0 {
		return Config{}, fmt.Errorf("%w: max retries must not be negative", ErrConfiguration)
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 1
	}
	if c.MaxOutputTokens == 0 {
		c.MaxOutputTokens = 1200
	}
	if c.MaxOutputTokens <= 0 {
		return Config{}, fmt.Errorf("%w: max output tokens must be positive", ErrConfiguration)
	}
	return c, nil
}

// LiveReady reports whether a mode is permitted to construct a Responses
// client. Auto intentionally returns false for incomplete configuration so a
// caller can select its deterministic fixture without sending a request.
func (c Config) LiveReady() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.Model) != ""
}
