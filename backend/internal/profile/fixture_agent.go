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
		profile.RiskSignals = []string{"May be a suitable candidate for a follow-up security training scenario based on a public event."}
		profile.RecommendedScenario = "event_followup"
		return json.Marshal(profile)
	}

	// The offline demo may only use authorized work-role context. This keeps
	// fixture campaigns varied without inventing personal facts or requiring a
	// live provider.
	roleContext := strings.ToLower(input.Employee.Department + " " + input.Employee.Title)
	switch {
	case strings.Contains(roleContext, "people"), strings.Contains(roleContext, "human resources"), strings.Contains(roleContext, " hr "):
		profile.RecommendedScenario = "benefit_update"
	case strings.Contains(roleContext, "engineering"), strings.Contains(roleContext, "developer"), strings.Contains(roleContext, "devops"), strings.Contains(roleContext, "security"), strings.Contains(roleContext, " it "):
		profile.RecommendedScenario = "saas_security_notice"
	}
	return json.Marshal(profile)
}
