package profile

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

var (
	//go:embed fixtures/search_E001.json
	searchE001 []byte
	//go:embed fixtures/search_default.json
	searchDefault []byte
)

// FixtureAdapter provides deliberately small, source-backed public evidence for
// the MVP. It never attempts a network lookup or invents an unavailable source.
type FixtureAdapter struct {
	fixtures        map[string][]ports.Evidence
	defaultEvidence []ports.Evidence
}

// NewFixtureAdapter loads and validates the embedded evidence fixtures.
func NewFixtureAdapter() (*FixtureAdapter, error) {
	return newFixtureAdapter(map[string][]byte{"E001": searchE001}, searchDefault)
}

func newFixtureAdapter(fixtures map[string][]byte, defaultFixture []byte) (*FixtureAdapter, error) {
	defaultEvidence, err := decodeEvidence(defaultFixture)
	if err != nil {
		return nil, fmt.Errorf("decode default enrichment fixture: %w", err)
	}

	adapter := &FixtureAdapter{
		fixtures:        make(map[string][]ports.Evidence, len(fixtures)),
		defaultEvidence: defaultEvidence,
	}
	for employeeID, fixture := range fixtures {
		evidence, err := decodeEvidence(fixture)
		if err != nil {
			return nil, fmt.Errorf("decode enrichment fixture for %q: %w", employeeID, err)
		}
		adapter.fixtures[employeeID] = evidence
	}
	return adapter, nil
}

// Search returns the employee fixture, or the explicit empty default fixture
// where no employee-specific fixture exists.
func (adapter *FixtureAdapter) Search(_ context.Context, employee domain.Employee) ([]ports.Evidence, error) {
	if adapter == nil {
		return nil, fmt.Errorf("fixture adapter is nil")
	}
	evidence, found := adapter.fixtures[employee.EmployeeID]
	if !found {
		evidence = adapter.defaultEvidence
	}
	return cloneEvidence(evidence), nil
}

func decodeEvidence(data []byte) ([]ports.Evidence, error) {
	var records []struct {
		Fact       string            `json:"fact"`
		SourceURL  *string           `json:"sourceUrl"`
		Confidence *float64          `json:"confidence"`
		SourceType domain.SourceType `json:"sourceType"`
		Tags       []string          `json:"tags"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("trailing JSON value")
		}
		return nil, err
	}

	evidence := make([]ports.Evidence, 0, len(records))
	for index, record := range records {
		if strings.TrimSpace(record.Fact) == "" {
			return nil, fmt.Errorf("evidence %d has empty fact", index)
		}
		if !record.SourceType.Valid() {
			return nil, fmt.Errorf("evidence %d has invalid source type", index)
		}
		if record.Confidence != nil && (*record.Confidence < 0 || *record.Confidence > 1) {
			return nil, fmt.Errorf("evidence %d has invalid confidence", index)
		}
		evidence = append(evidence, ports.Evidence{
			Fact:       record.Fact,
			SourceURL:  cloneString(record.SourceURL),
			Confidence: cloneFloat(record.Confidence),
			SourceType: record.SourceType,
			Tags:       append([]string{}, record.Tags...),
		})
	}
	return evidence, nil
}

func cloneEvidence(evidence []ports.Evidence) []ports.Evidence {
	result := make([]ports.Evidence, 0, len(evidence))
	for _, item := range evidence {
		result = append(result, ports.Evidence{
			Fact:       item.Fact,
			SourceURL:  cloneString(item.SourceURL),
			Confidence: cloneFloat(item.Confidence),
			SourceType: item.SourceType,
			Tags:       append([]string{}, item.Tags...),
		})
	}
	return result
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
