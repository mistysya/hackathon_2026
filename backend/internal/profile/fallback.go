package profile

import (
	"context"
	"fmt"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

type fallbackEnrichment struct {
	mode          openaiapi.Mode
	live, fixture ports.EnrichmentAdapter
}
type fallbackProfile struct {
	mode          openaiapi.Mode
	live, fixture ports.ProfileAgent
}

func WithEnrichmentFallback(mode openaiapi.Mode, live, fixture ports.EnrichmentAdapter) ports.EnrichmentAdapter {
	return &fallbackEnrichment{mode, live, fixture}
}
func WithProfileFallback(mode openaiapi.Mode, live, fixture ports.ProfileAgent) ports.ProfileAgent {
	return &fallbackProfile{mode, live, fixture}
}

func (f *fallbackEnrichment) Search(ctx context.Context, employee domain.Employee) ([]ports.Evidence, error) {
	if f.mode == openaiapi.ModeFixture {
		return f.fixtureSearch(ctx, employee)
	}
	if f.live == nil {
		if f.mode == openaiapi.ModeAuto {
			return f.fixtureSearch(ctx, employee)
		}
		return nil, fmt.Errorf("%w: enrichment client unavailable", openaiapi.ErrConfiguration)
	}
	result, err := f.live.Search(ctx, employee)
	if err == nil || f.mode == openaiapi.ModeLiveRequired || ctx.Err() != nil || !openaiapi.IsFallbackEligible(err) {
		return result, err
	}
	return f.fixtureSearch(ctx, employee)
}
func (f *fallbackEnrichment) fixtureSearch(ctx context.Context, employee domain.Employee) ([]ports.Evidence, error) {
	if f.fixture == nil {
		return nil, fmt.Errorf("fixture enrichment unavailable")
	}
	return f.fixture.Search(ctx, employee)
}

func (f *fallbackProfile) Generate(ctx context.Context, input ports.ProfileInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	if f.mode == openaiapi.ModeFixture {
		return f.fixtureGenerate(ctx, input, feedback)
	}
	if f.live == nil {
		if f.mode == openaiapi.ModeAuto {
			return f.fixtureGenerate(ctx, input, feedback)
		}
		return nil, fmt.Errorf("%w: profile client unavailable", openaiapi.ErrConfiguration)
	}
	result, err := f.live.Generate(ctx, input, feedback)
	if err == nil || f.mode == openaiapi.ModeLiveRequired || ctx.Err() != nil || !openaiapi.IsFallbackEligible(err) {
		return result, err
	}
	return f.fixtureGenerate(ctx, input, feedback)
}
func (f *fallbackProfile) fixtureGenerate(ctx context.Context, input ports.ProfileInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	if f.fixture == nil {
		return nil, fmt.Errorf("fixture profile agent unavailable")
	}
	return f.fixture.Generate(ctx, input, feedback)
}
