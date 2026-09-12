package campaign

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/jsonutil"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

const (
	defaultTemplateID  = "training_reminder"
	landingSentinel    = "__LANDING_URL__"
	landingPlaceholder = "{{landingUrl}}"
	testBrand          = "Security Awareness Demo"
)

//go:embed fixtures/event_followup.json
var eventFollowupFixture []byte

//go:embed fixtures/training_reminder.json
var trainingReminderFixture []byte

//go:embed fixtures/benefit_update.json
var benefitUpdateFixture []byte

//go:embed fixtures/saas_security_notice.json
var saasSecurityNoticeFixture []byte

type scenarioOutput struct {
	TemplateID     string               `json:"templateId"`
	Difficulty     domain.Difficulty    `json:"difficulty"`
	Subject        string               `json:"subject"`
	EmailHTML      string               `json:"emailHtml"`
	LandingConfig  domain.LandingConfig `json:"landingConfig"`
	DecisionReason string               `json:"decisionReason"`
	SafetyChecks   []domain.SafetyCheck `json:"safetyChecks"`
}

// FixtureScenarioAgent is a deterministic, allowlisted implementation of the
// scenario port. It is deliberately the only component that reads fixtures.
type FixtureScenarioAgent struct {
	fixtures map[string]scenarioOutput
}

// NewFixtureScenarioAgent loads all four static, reviewable scenario fixtures.
func NewFixtureScenarioAgent() (*FixtureScenarioAgent, error) {
	files := map[string][]byte{
		"event_followup":       eventFollowupFixture,
		"training_reminder":    trainingReminderFixture,
		"benefit_update":       benefitUpdateFixture,
		"saas_security_notice": saasSecurityNoticeFixture,
	}
	fixtures := make(map[string]scenarioOutput, len(files))
	for expectedID, raw := range files {
		fixture, err := decodeScenario(raw)
		if err != nil {
			return nil, fmt.Errorf("decode scenario fixture %q: %w", expectedID, err)
		}
		if fixture.TemplateID != expectedID {
			return nil, fmt.Errorf("scenario fixture %q has templateId %q", expectedID, fixture.TemplateID)
		}
		if err := validateFixture(fixture); err != nil {
			return nil, fmt.Errorf("validate scenario fixture %q: %w", expectedID, err)
		}
		fixtures[expectedID] = fixture
	}
	return &FixtureScenarioAgent{fixtures: fixtures}, nil
}

// Generate renders a named fixture. Feedback is accepted to satisfy the port;
// fixture output is deterministic and already conforms to its policy shape.
func (agent *FixtureScenarioAgent) Generate(_ context.Context, input ports.ScenarioInput, _ *ports.ValidationFeedback) ([]byte, error) {
	if agent == nil {
		return nil, fmt.Errorf("fixture scenario agent is nil")
	}
	templateID := personalizeScenario(input).PreferredTemplate
	fixture, ok := agent.fixtures[templateID]
	if !ok {
		fixture = agent.fixtures[defaultTemplateID]
	}
	if fixture.TemplateID == "" {
		return nil, fmt.Errorf("default scenario fixture is unavailable")
	}

	rendered, err := renderEmail(fixture.EmailHTML, input.Employee.DisplayName)
	if err != nil {
		return nil, err
	}
	fixture.EmailHTML = rendered
	return json.Marshal(fixture)
}

func renderEmail(source, displayName string) (string, error) {
	tmpl, err := template.New("campaign-email").Parse(source)
	if err != nil {
		return "", fmt.Errorf("parse email template: %w", err)
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, struct{ DisplayName string }{DisplayName: displayName}); err != nil {
		return "", fmt.Errorf("render email template: %w", err)
	}
	return strings.ReplaceAll(rendered.String(), landingSentinel, landingPlaceholder), nil
}

func decodeScenario(raw []byte) (scenarioOutput, error) {
	var output scenarioOutput
	if err := jsonutil.DecodeStrict(bytes.NewReader(raw), &output); err != nil {
		return scenarioOutput{}, err
	}
	return output, nil
}

func validateFixture(output scenarioOutput) error {
	if output.TemplateID == "" || !output.Difficulty.Valid() || output.Subject == "" || output.EmailHTML == "" || output.DecisionReason == "" {
		return fmt.Errorf("missing required scenario field")
	}
	if output.LandingConfig.Title == "" || output.LandingConfig.Brand != testBrand || output.LandingConfig.Description == "" || output.LandingConfig.CTALabel == "" {
		return fmt.Errorf("invalid test landing configuration")
	}
	if !strings.Contains(output.EmailHTML, landingSentinel) {
		return fmt.Errorf("email template does not contain landing sentinel")
	}
	return validateSafetyChecks(output.SafetyChecks)
}

var _ ports.ScenarioAgent = (*FixtureScenarioAgent)(nil)
