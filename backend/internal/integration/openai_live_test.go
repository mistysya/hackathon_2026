package integration

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mistysya/hackathon_2026/backend/internal/campaign"
	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
	"github.com/mistysya/hackathon_2026/backend/internal/structured"
)

type recordingScenarioAgent struct {
	delegate ports.ScenarioAgent
	lastErr  error
}

func (agent *recordingScenarioAgent) Generate(ctx context.Context, input ports.ScenarioInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	raw, err := agent.delegate.Generate(ctx, input, feedback)
	agent.lastErr = err
	return raw, err
}

// TestLiveOpenAIScenarioCreatesSafeMailboxContent verifies the real Responses
// integration with fake data only. It is deliberately opt-in because it sends
// one chargeable API request.
func TestLiveOpenAIScenarioCreatesSafeMailboxContent(t *testing.T) {
	if os.Getenv("OPENAI_LIVE_TEST") != "1" {
		t.Skip("set OPENAI_LIVE_TEST=1 to make a real OpenAI Responses API request")
	}

	config, err := OpenAIConfigFromEnvironment(os.Getenv)
	if err != nil {
		t.Fatalf("load OpenAI configuration: %v", err)
	}
	if strings.TrimSpace(config.APIKey) == "" {
		t.Fatal("OPENAI_API_KEY is empty; set it in backend/.env before running the live smoke test")
	}
	if strings.TrimSpace(config.Model) == "" {
		t.Fatal("OPENAI_MODEL is empty; set it in backend/.env before running the live smoke test")
	}
	config.Mode = openaiapi.ModeLiveRequired
	client, err := openaiapi.NewHTTPClient(config, nil)
	if err != nil {
		t.Fatalf("create OpenAI Responses client: %v", err)
	}

	database, repository := newSQLiteRepository(t)
	const employeeID = "LIVE_SMOKE_001"
	created, err := repository.ImportEmployees(context.Background(), []domain.Employee{{
		EmployeeID:  employeeID,
		DisplayName: "Demo User",
		Email:       "demo.user@example.test",
		Department:  "Engineering",
		Title:       "Engineer",
		Company:     "Security Awareness Demo",
	}})
	if err != nil {
		t.Fatalf("insert fake employee: %v", err)
	}
	if len(created) != 1 || !created[0] {
		t.Fatalf("insert fake employee result = %#v", created)
	}
	if err := repository.ReplaceProfile(context.Background(), domain.EmployeeProfile{
		EmployeeID:          employeeID,
		DisplayName:         "Demo User",
		Department:          "Engineering",
		RiskSignals:         []string{"training_reminder"},
		RecommendedScenario: "training_reminder",
	}); err != nil {
		t.Fatalf("insert fake employee profile: %v", err)
	}

	validator, err := structured.NewValidator()
	if err != nil {
		t.Fatalf("create structured validator: %v", err)
	}
	liveAgent := &recordingScenarioAgent{delegate: campaign.NewOpenAIScenarioAgent(client)}
	service := campaign.NewService(
		repository,
		repository,
		liveAgent,
		validator,
		campaign.NewCryptoIDGenerator(),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	generated, err := service.Generate(ctx, employeeID)
	if err != nil {
		if liveAgent.lastErr != nil {
			t.Fatalf("generate fake mailbox content through the Responses API (model %q): %v", config.Model, liveAgent.lastErr)
		}
		t.Fatalf("generate fake mailbox content through the Responses API (model %q): %v", config.Model, err)
	}

	var senderAddress, subject, emailHTML string
	var deliveredAt sql.NullString
	err = database.QueryRowContext(ctx, `
		SELECT sender_address, subject, email_html, delivered_at
		FROM mailbox_messages
		WHERE campaign_id = ? AND employee_id = ?
	`, generated.CampaignID, employeeID).Scan(&senderAddress, &subject, &emailHTML, &deliveredAt)
	if err != nil {
		t.Fatalf("read generated mailbox record: %v", err)
	}
	if senderAddress != store.MailboxSenderAddress {
		t.Fatalf("mailbox sender address = %q, want fixed demo sender", senderAddress)
	}
	if subject != generated.Subject || strings.TrimSpace(subject) == "" {
		t.Fatal("mailbox subject was not persisted from the generated campaign")
	}
	if emailHTML != generated.EmailHTML || !strings.Contains(emailHTML, "{{landingUrl}}") {
		t.Fatal("mailbox email HTML was not persisted with the controlled landing placeholder")
	}
	if deliveredAt.Valid {
		t.Fatalf("generated mailbox message was unexpectedly marked delivered at %q", deliveredAt.String)
	}

	t.Logf("live OpenAI mailbox smoke test passed: model=%q campaign=%q subjectLength=%d emailLength=%d", config.Model, generated.CampaignID, len(subject), len(emailHTML))
}
