package profile

import (
	"context"
	"errors"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

type testEnrichment struct {
	calls  int
	result []ports.Evidence
	err    error
}

func (s *testEnrichment) Search(context.Context, domain.Employee) ([]ports.Evidence, error) {
	s.calls++
	return s.result, s.err
}

func TestEnrichmentFallbackModes(t *testing.T) {
	fixture := &testEnrichment{result: []ports.Evidence{{Fact: "fixture"}}}
	live := &testEnrichment{err: &openaiapi.ProviderError{Kind: openaiapi.ErrorQuota, StatusCode: 429}}
	got, err := WithEnrichmentFallback(openaiapi.ModeAuto, live, fixture).Search(context.Background(), domain.Employee{})
	if err != nil || len(got) != 1 || got[0].Fact != "fixture" || live.calls != 1 || fixture.calls != 1 {
		t.Fatalf("auto got=%+v live=%d fixture=%d err=%v", got, live.calls, fixture.calls, err)
	}
	fixture.calls, live.calls = 0, 0
	_, err = WithEnrichmentFallback(openaiapi.ModeLiveRequired, live, fixture).Search(context.Background(), domain.Employee{})
	if !openaiapi.IsFallbackEligible(err) || fixture.calls != 0 {
		t.Fatalf("live required err=%v fixture=%d", err, fixture.calls)
	}
	fixture.calls, live.calls = 0, 0
	_, err = WithEnrichmentFallback(openaiapi.ModeFixture, live, fixture).Search(context.Background(), domain.Employee{})
	if err != nil || live.calls != 0 || fixture.calls != 1 {
		t.Fatalf("fixture err=%v live=%d fixture=%d", err, live.calls, fixture.calls)
	}
}

func TestEnrichmentFallbackDoesNotMaskCanceledContext(t *testing.T) {
	fixture := &testEnrichment{result: []ports.Evidence{{Fact: "fixture"}}}
	live := &testEnrichment{err: context.Canceled}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := WithEnrichmentFallback(openaiapi.ModeAuto, live, fixture).Search(ctx, domain.Employee{})
	if !errors.Is(err, context.Canceled) || fixture.calls != 0 {
		t.Fatalf("err=%v fixture=%d", err, fixture.calls)
	}
}
