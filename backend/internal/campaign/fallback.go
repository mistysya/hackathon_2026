package campaign

import (
	"context"
	"fmt"

	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

type fallbackScenario struct {
	mode          openaiapi.Mode
	live, fixture ports.ScenarioAgent
}

// WithScenarioFallback preserves the original ScenarioAgent interface while
// making each stage independently eligible for deterministic fallback.
func WithScenarioFallback(mode openaiapi.Mode, live, fixture ports.ScenarioAgent) ports.ScenarioAgent {
	return &fallbackScenario{mode, live, fixture}
}

func (f *fallbackScenario) Generate(ctx context.Context, input ports.ScenarioInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	if f.mode == openaiapi.ModeFixture {
		return f.fixtureGenerate(ctx, input, feedback)
	}
	if f.live == nil {
		if f.mode == openaiapi.ModeAuto {
			return f.fixtureGenerate(ctx, input, feedback)
		}
		return nil, fmt.Errorf("%w: scenario client unavailable", openaiapi.ErrConfiguration)
	}
	result, err := f.live.Generate(ctx, input, feedback)
	if err == nil || f.mode == openaiapi.ModeLiveRequired || ctx.Err() != nil || !openaiapi.IsFallbackEligible(err) {
		return result, err
	}
	return f.fixtureGenerate(ctx, input, feedback)
}
func (f *fallbackScenario) fixtureGenerate(ctx context.Context, input ports.ScenarioInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	if f.fixture == nil {
		return nil, fmt.Errorf("fixture scenario agent unavailable")
	}
	return f.fixture.Generate(ctx, input, feedback)
}
