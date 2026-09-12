package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func TestFixtureScenarioAgentRendersAllAllowlistedFixtures(t *testing.T) {
	agent, err := NewFixtureScenarioAgent()
	if err != nil {
		t.Fatalf("NewFixtureScenarioAgent() error = %v", err)
	}
	for _, templateID := range []string{"event_followup", "training_reminder", "benefit_update", "saas_security_notice"} {
		raw, err := agent.Generate(context.Background(), ports.ScenarioInput{
			Employee: domain.Employee{DisplayName: "Ava <Admin>"},
			Profile:  domain.EmployeeProfile{RecommendedScenario: templateID},
		}, nil)
		if err != nil {
			t.Fatalf("Generate(%q) error = %v", templateID, err)
		}
		output, err := decodeScenario(raw)
		if err != nil {
			t.Fatalf("decode %q: %v", templateID, err)
		}
		if output.TemplateID != templateID {
			t.Errorf("templateId = %q, want %q", output.TemplateID, templateID)
		}
		if err := validateScenarioPolicy(output); err != nil {
			t.Errorf("fixture %q policy = %v", templateID, err)
		}
		if strings.Contains(output.EmailHTML, "Ava <Admin>") || !strings.Contains(output.EmailHTML, "Ava &lt;Admin&gt;") {
			t.Errorf("email did not HTML-escape employee name: %q", output.EmailHTML)
		}
		if !strings.Contains(output.EmailHTML, landingPlaceholder) || strings.Contains(output.EmailHTML, landingSentinel) {
			t.Errorf("email placeholder = %q", output.EmailHTML)
		}
	}
}

func TestFixtureScenarioAgentFallsBackToTrainingReminder(t *testing.T) {
	agent, err := NewFixtureScenarioAgent()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := agent.Generate(context.Background(), ports.ScenarioInput{Profile: domain.EmployeeProfile{RecommendedScenario: "not_allowlisted"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	output, err := decodeScenario(raw)
	if err != nil {
		t.Fatal(err)
	}
	if output.TemplateID != defaultTemplateID {
		t.Fatalf("templateId = %q, want %q", output.TemplateID, defaultTemplateID)
	}
}

func TestCryptoIDGenerator(t *testing.T) {
	generate := NewCryptoIDGenerator()
	first, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^c_[0-9a-f]{32}$`).MatchString(first) {
		t.Errorf("ID = %q, want c_ plus 32 lowercase hex characters", first)
	}
	if first == second {
		t.Error("two crypto-generated IDs were equal")
	}
}

func TestServiceGenerate(t *testing.T) {
	valid := rawScenario(t)
	profile := domain.EmployeeProfile{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", RecommendedScenario: "training_reminder"}
	cases := []struct {
		name        string
		employeeErr error
		profile     *domain.EmployeeProfile
		agentRaw    [][]byte
		agentErr    error
		validation  []error
		id          IDGenerator
		createErr   error
		wantErr     error
		wantAnyErr  bool
		wantCalls   int
		wantCreates int
	}{
		{name: "success", profile: &profile, agentRaw: [][]byte{valid}, id: fixedID, wantCalls: 1, wantCreates: 1},
		{name: "missing employee", employeeErr: store.ErrNotFound, id: fixedID, wantErr: ErrEmployeeNotFound},
		{name: "profile required", profile: nil, id: fixedID, wantErr: ErrProfileRequired},
		{name: "validation repaired once", profile: &profile, agentRaw: [][]byte{valid, valid}, validation: []error{errors.New("missing field"), nil}, id: fixedID, wantCalls: 2, wantCreates: 1},
		{name: "validation fails twice", profile: &profile, agentRaw: [][]byte{valid, valid}, validation: []error{errors.New("bad"), errors.New("still bad")}, id: fixedID, wantErr: ErrGenerationFailed, wantCalls: 2},
		{name: "agent error", profile: &profile, agentErr: errors.New("agent unavailable"), id: fixedID, wantErr: ErrGenerationFailed, wantCalls: 1},
		{name: "strict decode failure", profile: &profile, agentRaw: [][]byte{[]byte(`{"templateId":"training_reminder","extra":true}`)}, id: fixedID, wantErr: ErrGenerationFailed, wantCalls: 1},
		{name: "policy failure", profile: &profile, agentRaw: [][]byte{[]byte(strings.Replace(string(valid), landingPlaceholder, "", 1))}, id: fixedID, wantErr: ErrGenerationFailed, wantCalls: 1},
		{name: "failed required safety check", profile: &profile, agentRaw: [][]byte{[]byte(strings.Replace(string(valid), `"passed":true`, `"passed":false`, 1))}, id: fixedID, wantErr: ErrGenerationFailed, wantCalls: 1},
		{name: "ID failure", profile: &profile, agentRaw: [][]byte{valid}, id: func() (string, error) { return "", errors.New("random unavailable") }, wantErr: ErrGenerationFailed, wantCalls: 1},
		{name: "repository failure", profile: &profile, agentRaw: [][]byte{valid}, id: fixedID, createErr: errors.New("db unavailable"), wantAnyErr: true, wantCalls: 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			employee := &fakeEmployeeRepository{employee: domain.EmployeeDetails{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", Profile: test.profile}, err: test.employeeErr}
			campaigns := &fakeCampaignRepository{createErr: test.createErr}
			agent := &fakeScenarioAgent{responses: test.agentRaw, err: test.agentErr}
			validator := &fakeValidator{scenarioErrors: test.validation}
			service := NewService(employee, campaigns, agent, validator, test.id)
			got, err := service.Generate(context.Background(), "E001")
			if test.wantAnyErr && err == nil {
				t.Fatal("Generate() error = nil, want repository error")
			}
			if !test.wantAnyErr && !errors.Is(err, test.wantErr) {
				t.Fatalf("Generate() error = %v, want errors.Is(..., %v)", err, test.wantErr)
			}
			if test.wantErr == nil && !test.wantAnyErr && err != nil {
				t.Fatalf("Generate() unexpected error = %v", err)
			}
			if agent.calls != test.wantCalls || len(campaigns.created) != test.wantCreates {
				t.Fatalf("agent calls, creates = %d, %d; want %d, %d", agent.calls, len(campaigns.created), test.wantCalls, test.wantCreates)
			}
			if test.wantCreates == 1 {
				if got.CampaignID != "c_fixed" || got.Status != domain.CampaignStatusPendingReview || got.ApprovedBy != nil || got.ApprovedAt != nil || got.RejectionReason != nil {
					t.Errorf("persisted campaign = %+v", got)
				}
			}
			if test.name == "validation repaired once" && (len(agent.feedback) != 2 || agent.feedback[1] == nil) {
				t.Error("validator failure did not produce exactly one retry with feedback")
			}
		})
	}
}

func TestServiceGetMapsNotFound(t *testing.T) {
	service := NewService(&fakeEmployeeRepository{}, &fakeCampaignRepository{getErr: store.ErrNotFound}, &fakeScenarioAgent{}, &fakeValidator{}, fixedID)
	_, err := service.Get(context.Background(), "missing")
	if !errors.Is(err, ErrCampaignNotFound) {
		t.Fatalf("Get() error = %v, want ErrCampaignNotFound", err)
	}
}

func TestRoutesFrozenHTTPContract(t *testing.T) {
	valid := rawScenario(t)
	profile := &domain.EmployeeProfile{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", RecommendedScenario: "training_reminder"}
	makeHandler := func(employeeErr error, campaignGetErr error, profile *domain.EmployeeProfile, createErr error) http.Handler {
		employees := &fakeEmployeeRepository{employee: domain.EmployeeDetails{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", Profile: profile}, err: employeeErr}
		campaigns := &fakeCampaignRepository{getErr: campaignGetErr, createErr: createErr}
		service := NewService(employees, campaigns, &fakeScenarioAgent{responses: [][]byte{valid}}, &fakeValidator{}, fixedID)
		return httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), NewRoutes(service, slog.New(slog.NewTextHandler(io.Discard, nil))))
	}
	cases := []struct {
		name, method, path, body, contentType string
		handler                               http.Handler
		status                                int
		code                                  string
	}{
		{name: "created", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001"}`, contentType: "application/json", handler: makeHandler(nil, nil, profile, nil), status: http.StatusCreated},
		{name: "invalid media", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001"}`, contentType: "text/plain", handler: makeHandler(nil, nil, profile, nil), status: http.StatusBadRequest, code: "invalid_content_type"},
		{name: "unknown field", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001","templateId":"x"}`, contentType: "application/json", handler: makeHandler(nil, nil, profile, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "trailing value", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001"} {}`, contentType: "application/json", handler: makeHandler(nil, nil, profile, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "empty employee", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":" "}`, contentType: "application/json", handler: makeHandler(nil, nil, profile, nil), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "employee missing", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E404"}`, contentType: "application/json", handler: makeHandler(store.ErrNotFound, nil, profile, nil), status: http.StatusNotFound, code: "employee_not_found"},
		{name: "profile required", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001"}`, contentType: "application/json", handler: makeHandler(nil, nil, nil, nil), status: http.StatusConflict, code: "profile_required"},
		{name: "repository error", method: http.MethodPost, path: "/campaigns/generate", body: `{"employeeId":"E001"}`, contentType: "application/json", handler: makeHandler(nil, nil, profile, errors.New("db down")), status: http.StatusInternalServerError, code: "internal_error"},
		{name: "campaign missing", method: http.MethodGet, path: "/campaigns/c_missing", handler: makeHandler(nil, store.ErrNotFound, profile, nil), status: http.StatusNotFound, code: "campaign_not_found"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response := httptest.NewRecorder()
			test.handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q", response.Header().Get("Content-Type"))
			}
			if test.code != "" {
				var envelope struct {
					Error struct{ Code, RequestID string }
				}
				if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
					t.Fatal(err)
				}
				if envelope.Error.Code != test.code || envelope.Error.RequestID == "" {
					t.Errorf("error envelope = %+v, want code=%q with requestId", envelope.Error, test.code)
				}
			}
			if test.status == http.StatusCreated {
				var campaign domain.GeneratedCampaign
				if err := json.NewDecoder(response.Body).Decode(&campaign); err != nil {
					t.Fatal(err)
				}
				if campaign.CampaignID != "c_fixed" || campaign.SafetyChecks == nil || campaign.Status != domain.CampaignStatusPendingReview {
					t.Errorf("campaign response = %+v", campaign)
				}
			}
		})
	}
}

func TestRoutesGetSuccessAndGenerationFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Run("get success", func(t *testing.T) {
		expected := domain.GeneratedCampaign{
			CampaignID: "c_existing", EmployeeID: "E001", TemplateID: "training_reminder",
			Difficulty: domain.DifficultyLow, Subject: "Training", EmailHTML: "<a>{{landingUrl}}</a>",
			LandingConfig:  domain.LandingConfig{Title: "Title", Brand: testBrand, Description: "Description", CTALabel: "Open"},
			DecisionReason: "Reason", SafetyChecks: []domain.SafetyCheck{}, Status: domain.CampaignStatusPendingReview,
		}
		repository := &fakeCampaignRepository{created: []domain.GeneratedCampaign{expected}}
		handler := httpapi.NewRouter(logger, NewRoutes(NewService(&fakeEmployeeRepository{}, repository, &fakeScenarioAgent{}, &fakeValidator{}, fixedID), logger))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/campaigns/c_existing", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
		}
		var got domain.GeneratedCampaign
		if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("response = %+v, want %+v", got, expected)
		}
	})
	t.Run("generation failure", func(t *testing.T) {
		profile := &domain.EmployeeProfile{EmployeeID: "E001"}
		service := NewService(
			&fakeEmployeeRepository{employee: domain.EmployeeDetails{EmployeeID: "E001", Profile: profile}},
			&fakeCampaignRepository{}, &fakeScenarioAgent{err: errors.New("agent down")}, &fakeValidator{}, fixedID,
		)
		handler := httpapi.NewRouter(logger, NewRoutes(service, logger))
		request := httptest.NewRequest(http.MethodPost, "/campaigns/generate", strings.NewReader(`{"employeeId":"E001"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), `"generation_failed"`) {
			t.Errorf("status/body = %d/%s, want 502 generation_failed", response.Code, response.Body.String())
		}
	})
}

func rawScenario(t *testing.T) []byte {
	t.Helper()
	agent, err := NewFixtureScenarioAgent()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := agent.Generate(context.Background(), ports.ScenarioInput{Employee: domain.Employee{DisplayName: "Demo User"}, Profile: domain.EmployeeProfile{RecommendedScenario: "training_reminder"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func fixedID() (string, error) { return "c_fixed", nil }

type fakeEmployeeRepository struct {
	employee domain.EmployeeDetails
	err      error
}

func (repository *fakeEmployeeRepository) ImportEmployees(context.Context, []domain.Employee) ([]bool, error) {
	return nil, errors.New("not implemented")
}
func (repository *fakeEmployeeRepository) ListEmployees(context.Context) ([]domain.EmployeeSummary, error) {
	return nil, errors.New("not implemented")
}
func (repository *fakeEmployeeRepository) GetEmployee(context.Context, string) (domain.EmployeeDetails, error) {
	return repository.employee, repository.err
}
func (repository *fakeEmployeeRepository) GetProfile(context.Context, string) (domain.EmployeeProfile, error) {
	return domain.EmployeeProfile{}, errors.New("not implemented")
}
func (repository *fakeEmployeeRepository) ReplaceProfile(context.Context, domain.EmployeeProfile) error {
	return errors.New("not implemented")
}

type fakeCampaignRepository struct {
	created   []domain.GeneratedCampaign
	createErr error
	getErr    error
}

func (repository *fakeCampaignRepository) CreateCampaign(_ context.Context, campaign domain.GeneratedCampaign) error {
	if repository.createErr != nil {
		return repository.createErr
	}
	repository.created = append(repository.created, campaign)
	return nil
}
func (repository *fakeCampaignRepository) GetCampaign(_ context.Context, campaignID string) (domain.GeneratedCampaign, error) {
	if repository.getErr != nil {
		return domain.GeneratedCampaign{}, repository.getErr
	}
	for _, campaign := range repository.created {
		if campaign.CampaignID == campaignID {
			return campaign, nil
		}
	}
	return domain.GeneratedCampaign{}, store.ErrNotFound
}

type fakeScenarioAgent struct {
	responses [][]byte
	err       error
	calls     int
	feedback  []*ports.ValidationFeedback
}

func (agent *fakeScenarioAgent) Generate(_ context.Context, _ ports.ScenarioInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	agent.calls++
	agent.feedback = append(agent.feedback, feedback)
	if agent.err != nil {
		return nil, agent.err
	}
	if len(agent.responses) == 0 {
		return nil, errors.New("no response")
	}
	index := agent.calls - 1
	if index >= len(agent.responses) {
		index = len(agent.responses) - 1
	}
	return agent.responses[index], nil
}

type fakeValidator struct {
	scenarioErrors []error
	calls          int
}

func (*fakeValidator) ValidateProfile([]byte) error { return nil }
func (validator *fakeValidator) ValidateScenario([]byte) error {
	index := validator.calls
	validator.calls++
	if index < len(validator.scenarioErrors) {
		return validator.scenarioErrors[index]
	}
	return nil
}

var _ ports.EmployeeRepository = (*fakeEmployeeRepository)(nil)
var _ ports.CampaignRepository = (*fakeCampaignRepository)(nil)
var _ ports.ScenarioAgent = (*fakeScenarioAgent)(nil)
var _ ports.StructuredValidator = (*fakeValidator)(nil)
