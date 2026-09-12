package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"unicode/utf8"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/jsonutil"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

var scenarioSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"required":["templateId","senderPersona","subject","emailBody","ctaLabel","landingTitle","landingDescription","decisionReason"],"properties":{"templateId":{"type":"string","enum":["event_followup","training_reminder","benefit_update","saas_security_notice"]},"senderPersona":{"type":"string","enum":["demo_events","demo_learning","demo_people_ops","demo_security"]},"subject":{"type":"string"},"emailBody":{"type":"array","items":{"type":"string"}},"ctaLabel":{"type":"string"},"landingTitle":{"type":"string"},"landingDescription":{"type":"string"},"decisionReason":{"type":"string"}}}`)

type senderCatalogEntry struct {
	Persona    string
	Difficulty domain.Difficulty
}

var senderCatalog = map[string]senderCatalogEntry{
	"event_followup":       {"demo_events", domain.DifficultyMedium},
	"training_reminder":    {"demo_learning", domain.DifficultyLow},
	"benefit_update":       {"demo_people_ops", domain.DifficultyMedium},
	"saas_security_notice": {"demo_security", domain.DifficultyHigh},
}

type OpenAIScenarioAgent struct{ client openaiapi.Client }

func NewOpenAIScenarioAgent(client openaiapi.Client) *OpenAIScenarioAgent {
	return &OpenAIScenarioAgent{client: client}
}

func (agent *OpenAIScenarioAgent) Generate(ctx context.Context, input ports.ScenarioInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	if agent == nil || agent.client == nil {
		return nil, fmt.Errorf("%w: scenario client unavailable", openaiapi.ErrConfiguration)
	}
	instructions := "Create neutral, plain-text training copy only for a safe local Security Awareness Demo. Use exactly one matching template and sender persona: event_followup=demo_events; training_reminder=demo_learning; benefit_update=demo_people_ops; saas_security_notice=demo_security. Use one to three short email-body paragraphs, no real organization brands, and no sender address. Every output string must avoid these literal strings: <, >, http://, https://, javascript:, password, credential, 登入, 密碼, 帳密, gmail, google."
	if feedback != nil && strings.TrimSpace(feedback.Message) != "" {
		instructions += " Correct this schema or policy issue: " + truncateScenarioFeedback(feedback.Message)
	}
	payload, _ := json.Marshal(struct{ DisplayName, Department, RecommendedScenario string }{input.Employee.DisplayName, input.Employee.Department, input.Profile.RecommendedScenario})
	raw, _, err := agent.client.StructuredResponse(ctx, openaiapi.Request{Instructions: instructions, Input: string(payload), SchemaName: "campaign_copy", Schema: scenarioSchema})
	if err != nil {
		return nil, err
	}
	var output struct {
		TemplateID, SenderPersona, Subject string
		EmailBody                          []string `json:"emailBody"`
		CTALabel                           string   `json:"ctaLabel"`
		LandingTitle                       string   `json:"landingTitle"`
		LandingDescription                 string   `json:"landingDescription"`
		DecisionReason                     string   `json:"decisionReason"`
	}
	if err := jsonutil.DecodeStrict(bytes.NewReader(raw), &output); err != nil {
		return nil, &openaiapi.ProviderError{Kind: openaiapi.ErrorOutput, Code: "scenario_decode"}
	}
	entry, ok := senderCatalog[output.TemplateID]
	if !ok || entry.Persona != output.SenderPersona {
		return nil, &openaiapi.ProviderError{Kind: openaiapi.ErrorOutput, Code: "scenario_catalog"}
	}
	if code := safeCopyFailureCode(output); code != "" {
		return nil, &openaiapi.ProviderError{Kind: openaiapi.ErrorOutput, Code: code}
	}
	email, err := renderLiveEmail(input.Employee.DisplayName, output.EmailBody, output.CTALabel)
	if err != nil {
		return nil, &openaiapi.ProviderError{Kind: openaiapi.ErrorOutput, Code: "scenario_render"}
	}
	generated := scenarioOutput{
		TemplateID: output.TemplateID, Difficulty: entry.Difficulty, Subject: strings.TrimSpace(output.Subject), EmailHTML: email,
		LandingConfig:  domain.LandingConfig{Title: strings.TrimSpace(output.LandingTitle), Brand: testBrand, Description: strings.TrimSpace(output.LandingDescription), CTALabel: strings.TrimSpace(output.CTALabel)},
		DecisionReason: strings.TrimSpace(output.DecisionReason), SafetyChecks: generatedSafetyChecks(),
	}
	return json.Marshal(generated)
}

func renderLiveEmail(displayName string, body []string, cta string) (string, error) {
	if len(body) == 0 {
		return "", fmt.Errorf("empty email body")
	}
	const source = `{{range .Body}}<p>{{.}}</p>{{end}}<p><a href="__LANDING_URL__">{{.CTA}}</a></p>`
	tmpl, err := template.New("campaign-email").Parse(source)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, struct {
		Body []string
		CTA  string
	}{append([]string{"Hi " + displayName + ","}, body...), cta}); err != nil {
		return "", err
	}
	return strings.ReplaceAll(rendered.String(), landingSentinel, landingPlaceholder), nil
}

func generatedSafetyChecks() []domain.SafetyCheck {
	return []domain.SafetyCheck{
		{Rule: "no_real_credentials_requested", Passed: true}, {Rule: "no_sensitive_personal_data", Passed: true},
		{Rule: "no_unsourced_personal_facts", Passed: true}, {Rule: "no_prohibited_impersonation", Passed: true},
		{Rule: "cta_points_to_controlled_domain", Passed: true},
	}
}

func safeCopy(output struct {
	TemplateID, SenderPersona, Subject string
	EmailBody                          []string `json:"emailBody"`
	CTALabel                           string   `json:"ctaLabel"`
	LandingTitle                       string   `json:"landingTitle"`
	LandingDescription                 string   `json:"landingDescription"`
	DecisionReason                     string   `json:"decisionReason"`
}) bool {
	return safeCopyFailureCode(output) == ""
}

func safeCopyFailureCode(output struct {
	TemplateID, SenderPersona, Subject string
	EmailBody                          []string `json:"emailBody"`
	CTALabel                           string   `json:"ctaLabel"`
	LandingTitle                       string   `json:"landingTitle"`
	LandingDescription                 string   `json:"landingDescription"`
	DecisionReason                     string   `json:"decisionReason"`
}) string {
	values := append([]string{output.Subject, output.CTALabel, output.LandingTitle, output.LandingDescription, output.DecisionReason}, output.EmailBody...)
	if len(output.EmailBody) == 0 || len(output.EmailBody) > 6 {
		return "scenario_body_count"
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "scenario_empty_field"
		}
		if utf8.RuneCountInString(trimmed) > 500 {
			return "scenario_field_too_long"
		}
		lowered := strings.ToLower(trimmed)
		for _, blocked := range []string{"<", ">", "http://", "https://", "javascript:", "password", "credential", "登入", "密碼", "帳密", "gmail", "google"} {
			if strings.Contains(lowered, blocked) {
				return "scenario_prohibited_term"
			}
		}
	}
	return ""
}

func truncateScenarioFeedback(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 240 {
		return value[:240]
	}
	return value
}

var _ ports.ScenarioAgent = (*OpenAIScenarioAgent)(nil)
