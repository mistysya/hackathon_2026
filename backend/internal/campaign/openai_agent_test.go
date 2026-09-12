package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

func TestSafeCopyAllowsValidGeneratedCopy(t *testing.T) {
	output := struct {
		TemplateID, SenderPersona, Subject string
		EmailBody                          []string `json:"emailBody"`
		CTALabel                           string   `json:"ctaLabel"`
		LandingTitle                       string   `json:"landingTitle"`
		LandingDescription                 string   `json:"landingDescription"`
		DecisionReason                     string   `json:"decisionReason"`
	}{
		TemplateID:         "training_reminder",
		SenderPersona:      "demo_learning",
		Subject:            "Complete your security training",
		EmailBody:          []string{"A short training module is ready for you.", "It takes only a few minutes to complete."},
		CTALabel:           "Open training",
		LandingTitle:       "Security training reminder",
		LandingDescription: "Complete this demo training module.",
		DecisionReason:     "A short learning reminder suits this demo profile.",
	}

	if !safeCopy(output) {
		t.Fatal("safeCopy rejected valid generated copy")
	}
}

func TestOpenAIScenarioAgentProvidesTemplatePersonaMapping(t *testing.T) {
	client := &scenarioResponseStub{response: []byte(`{"templateId":"training_reminder","senderPersona":"demo_learning","subject":"Complete training","emailBody":["A short training module is ready."],"ctaLabel":"Open training","landingTitle":"Training reminder","landingDescription":"Complete this demo training module.","decisionReason":"A short learning reminder suits this demo profile."}`)}
	agent := NewOpenAIScenarioAgent(client)

	raw, err := agent.Generate(context.Background(), ports.ScenarioInput{
		Employee: domain.Employee{DisplayName: "Demo User"},
		Profile:  domain.EmployeeProfile{RecommendedScenario: "training_reminder"},
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Contains(client.request.Schema, []byte("senderPersona")) || !strings.Contains(client.request.Instructions, "event_followup=demo_events") || !strings.Contains(client.request.Instructions, "training_reminder=demo_learning") {
		t.Fatalf("scenario request is missing template/persona mapping: %#v", client.request)
	}
	var generated scenarioOutput
	if err := json.Unmarshal(raw, &generated); err != nil {
		t.Fatalf("decode generated scenario: %v", err)
	}
	if generated.TemplateID != "training_reminder" || generated.Difficulty != domain.DifficultyLow {
		t.Fatalf("generated scenario = %#v, want training_reminder difficulty", generated)
	}
}

type scenarioResponseStub struct {
	response []byte
	request  openaiapi.Request
}

func (stub *scenarioResponseStub) StructuredResponse(_ context.Context, request openaiapi.Request) ([]byte, openaiapi.Metadata, error) {
	stub.request = request
	return stub.response, openaiapi.Metadata{}, nil
}
