package profile

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

// FixtureAgent produces deterministic, non-sensitive profile JSON from public
// evidence. Validation feedback is accepted to satisfy the agent contract; the
// deterministic fixture output itself does not need to invent a new fact.
type FixtureAgent struct{}

func NewFixtureAgent() *FixtureAgent {
	return &FixtureAgent{}
}

func (agent *FixtureAgent) Generate(_ context.Context, input ports.ProfileInput, _ *ports.ValidationFeedback) ([]byte, error) {
	profile := domain.EmployeeProfile{
		EmployeeID:          input.Employee.EmployeeID,
		DisplayName:         input.Employee.DisplayName,
		Department:          input.Employee.Department,
		PublicFacts:         make([]domain.PublicFact, 0, len(input.Evidence)),
		RiskSignals:         []string{},
		RecommendedScenario: "training_reminder",
	}

	hasEventEvidence := false
	for _, evidence := range input.Evidence {
		profile.PublicFacts = append(profile.PublicFacts, domain.PublicFact{
			Fact:       evidence.Fact,
			SourceURL:  cloneString(evidence.SourceURL),
			Confidence: cloneFloat(evidence.Confidence),
			SourceType: evidence.SourceType,
		})
		for _, tag := range evidence.Tags {
			if strings.EqualFold(strings.TrimSpace(tag), "event") {
				hasEventEvidence = true
			}
		}
	}
	if hasEventEvidence {
		profile.RiskSignals = []string{"可能適合以公開活動後的資安提醒作為後續訓練情境。"}
		profile.RecommendedScenario = "event_followup"
	}
	return json.Marshal(profile)
}
