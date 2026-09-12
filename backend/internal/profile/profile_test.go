package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func TestFixtureAdapterUsesEmployeeFixtureAndDefault(t *testing.T) {
	adapter, err := NewFixtureAdapter()
	if err != nil {
		t.Fatalf("NewFixtureAdapter: %v", err)
	}

	e001, err := adapter.Search(context.Background(), domain.Employee{EmployeeID: "E001"})
	if err != nil {
		t.Fatalf("Search E001: %v", err)
	}
	if len(e001) != 2 || e001[0].SourceType != domain.SourceTypeFixture || e001[0].SourceURL == nil || e001[0].Confidence == nil {
		t.Fatalf("unexpected E001 evidence: %+v", e001)
	}
	defaultEvidence, err := adapter.Search(context.Background(), domain.Employee{EmployeeID: "E999"})
	if err != nil {
		t.Fatalf("Search default: %v", err)
	}
	if defaultEvidence == nil || len(defaultEvidence) != 0 {
		t.Fatalf("default evidence = %#v, want non-nil empty slice", defaultEvidence)
	}

	*e001[0].SourceURL = "mutated"
	again, err := adapter.Search(context.Background(), domain.Employee{EmployeeID: "E001"})
	if err != nil || *again[0].SourceURL == "mutated" {
		t.Fatalf("adapter returned mutable fixture data: %+v, %v", again, err)
	}
}

func TestFixtureAdapterRejectsCorruptAndMissingFixtureData(t *testing.T) {
	if _, err := newFixtureAdapter(map[string][]byte{"E001": []byte("{")}, []byte("[]")); err == nil {
		t.Fatal("corrupt employee fixture unexpectedly accepted")
	}
	if _, err := newFixtureAdapter(map[string][]byte{}, nil); err == nil {
		t.Fatal("missing default fixture unexpectedly accepted")
	}
	if _, err := decodeEvidence([]byte("[] []")); err == nil {
		t.Fatal("trailing fixture JSON unexpectedly accepted")
	}
}

func TestFixtureAgentPreservesIdentityAndSources(t *testing.T) {
	url := "https://example.test/source"
	confidence := 0.7
	agent := NewFixtureAgent()
	raw, err := agent.Generate(context.Background(), ports.ProfileInput{
		Employee: domain.Employee{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering"},
		Evidence: []ports.Evidence{{
			Fact: "Attended a public event", SourceURL: &url, Confidence: &confidence, SourceType: domain.SourceTypeFixture, Tags: []string{"event"},
		}},
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var profile domain.EmployeeProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("decode generated profile: %v", err)
	}
	if profile.EmployeeID != "E001" || profile.DisplayName != "Demo User" || profile.Department != "Engineering" || profile.RecommendedScenario != "event_followup" {
		t.Fatalf("unexpected generated profile: %+v", profile)
	}
	if len(profile.PublicFacts) != 1 || profile.PublicFacts[0].SourceURL == nil || *profile.PublicFacts[0].SourceURL != url || profile.PublicFacts[0].Confidence == nil || *profile.PublicFacts[0].Confidence != confidence {
		t.Fatalf("source metadata was not preserved: %+v", profile.PublicFacts)
	}
	if len(profile.RiskSignals) != 1 || !strings.Contains(profile.RiskSignals[0], "May") {
		t.Fatalf("event risk signal = %#v, want cautious language", profile.RiskSignals)
	}
}

func TestFixtureAgentUsesSafeEmptyEvidenceFallback(t *testing.T) {
	raw, err := NewFixtureAgent().Generate(context.Background(), ports.ProfileInput{
		Employee: domain.Employee{EmployeeID: "E002", DisplayName: "No Evidence", Department: "Sales"},
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var profile domain.EmployeeProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("decode generated profile: %v", err)
	}
	if len(profile.PublicFacts) != 0 || len(profile.RiskSignals) != 0 || profile.RecommendedScenario != "training_reminder" {
		t.Fatalf("unexpected empty-evidence profile: %+v", profile)
	}
}

func TestServiceFirstTrySuccessReturnsPersistedProfile(t *testing.T) {
	repository := newFakeRepository()
	validator := &fakeValidator{}
	service := NewService(repository, fixtureAdapterWithEvidence(nil), &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering")}}, validator)

	profile, err := service.Enrich(context.Background(), "E001")
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if validator.calls != 1 || repository.replaceCalls != 1 || repository.getProfileCalls != 1 || profile.RecommendedScenario != "persisted" {
		t.Fatalf("unexpected flow: validator=%d replace=%d getProfile=%d profile=%+v", validator.calls, repository.replaceCalls, repository.getProfileCalls, profile)
	}
}

func TestServiceRetriesOnceWithValidationFeedback(t *testing.T) {
	repository := newFakeRepository()
	validator := &fakeValidator{errors: []error{errors.New("missing recommendedScenario")}}
	agent := &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering"), profileJSON("E001", "Demo User", "Engineering")}}
	service := NewService(repository, fixtureAdapterWithEvidence(nil), agent, validator)

	if _, err := service.Enrich(context.Background(), "E001"); err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	if agent.feedback == nil || agent.feedback.Message != "missing recommendedScenario" || validator.calls != 2 || repository.replaceCalls != 1 {
		t.Fatalf("unexpected retry state: feedback=%+v calls=%d replace=%d", agent.feedback, validator.calls, repository.replaceCalls)
	}
}

func TestServiceDoesNotWriteOnRepeatedInvalidOrIdentityMismatch(t *testing.T) {
	for name, setup := range map[string]func(*fakeRepository) *Service{
		"two validation failures": func(repository *fakeRepository) *Service {
			return NewService(repository, fixtureAdapterWithEvidence(nil), &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering"), profileJSON("E001", "Demo User", "Engineering")}}, &fakeValidator{errors: []error{errors.New("first"), errors.New("second")}})
		},
		"identity mismatch": func(repository *fakeRepository) *Service {
			return NewService(repository, fixtureAdapterWithEvidence(nil), &scriptedAgent{outputs: [][]byte{profileJSON("other", "Demo User", "Engineering")}}, &fakeValidator{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			repository := newFakeRepository()
			_, err := setup(repository).Enrich(context.Background(), "E001")
			if !errors.Is(err, ErrEnrichmentFailed) || repository.replaceCalls != 0 {
				t.Fatalf("err=%v replace=%d, want enrichment failure with no write", err, repository.replaceCalls)
			}
		})
	}
}

func TestServiceDoesNotWriteWhenAdapterOrAgentFails(t *testing.T) {
	for name, service := range map[string]*Service{
		"adapter failure": NewService(newFakeRepository(), &fixedAdapter{err: errors.New("fixture unavailable")}, &scriptedAgent{}, &fakeValidator{}),
		"agent failure":   NewService(newFakeRepository(), fixtureAdapterWithEvidence(nil), &scriptedAgent{err: errors.New("agent unavailable")}, &fakeValidator{}),
	} {
		t.Run(name, func(t *testing.T) {
			repository := service.repository.(*fakeRepository)
			if _, err := service.Enrich(context.Background(), "E001"); !errors.Is(err, ErrEnrichmentFailed) || repository.replaceCalls != 0 {
				t.Fatalf("err=%v replace=%d, want enrichment failure with no write", err, repository.replaceCalls)
			}
		})
	}
}

func TestDecodeProfileRejectsUnknownAndTrailingContent(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"employeeId":"E001","displayName":"Demo User","department":"Engineering","publicFacts":[],"riskSignals":[],"recommendedScenario":"training_reminder","extra":true}`),
		append(profileJSON("E001", "Demo User", "Engineering"), []byte(` {}`)...),
		[]byte(`null`),
	} {
		if _, err := decodeProfile(raw); err == nil {
			t.Fatalf("decodeProfile(%s) unexpectedly succeeded", raw)
		}
	}
}

func TestServiceMapsNotFoundAndLeavesRepositoryFailuresInternal(t *testing.T) {
	repository := newFakeRepository()
	repository.getEmployeeErr = store.ErrNotFound
	service := NewService(repository, fixtureAdapterWithEvidence(nil), &scriptedAgent{}, &fakeValidator{})
	if _, err := service.Enrich(context.Background(), "none"); !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("not found error = %v", err)
	}

	repository = newFakeRepository()
	repository.replaceErr = errors.New("write failed")
	service = NewService(repository, fixtureAdapterWithEvidence(nil), &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering")}}, &fakeValidator{})
	if _, err := service.Enrich(context.Background(), "E001"); err == nil || errors.Is(err, ErrEnrichmentFailed) {
		t.Fatalf("write error = %v, want unclassified repository error", err)
	}
}

func TestRoutesValidateBodyAndMapErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		repository  *fakeRepository
		agent       ports.ProfileAgent
		validator   ports.StructuredValidator
		status      int
		code        string
	}{
		{name: "invalid content type", contentType: "text/plain", body: "{}", repository: newFakeRepository(), status: http.StatusBadRequest, code: "invalid_content_type"},
		{name: "unknown request field", contentType: "application/json", body: `{"mode":"live"}`, repository: newFakeRepository(), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "trailing JSON", contentType: "application/json", body: "{} {}", repository: newFakeRepository(), status: http.StatusBadRequest, code: "invalid_request"},
		{name: "not found", contentType: "application/json", body: "{}", repository: &fakeRepository{getEmployeeErr: store.ErrNotFound}, agent: &scriptedAgent{}, validator: &fakeValidator{}, status: http.StatusNotFound, code: "employee_not_found"},
		{name: "agent failure", contentType: "application/json", body: "{}", repository: newFakeRepository(), agent: &scriptedAgent{err: errors.New("agent down")}, validator: &fakeValidator{}, status: http.StatusBadGateway, code: "enrichment_failed"},
		{name: "repository failure", contentType: "application/json", body: "{}", repository: fakeRepositoryWithReplaceError(), agent: &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering")}}, validator: &fakeValidator{}, status: http.StatusInternalServerError, code: "internal_error"},
		{name: "success", contentType: "application/json; charset=utf-8", body: "{}", repository: newFakeRepository(), agent: &scriptedAgent{outputs: [][]byte{profileJSON("E001", "Demo User", "Engineering")}}, validator: &fakeValidator{}, status: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(test.repository, fixtureAdapterWithEvidence(nil), test.agent, test.validator)
			routes := NewRoutes(service, slog.New(slog.NewTextHandler(io.Discard, nil)))
			router := httpapi.NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), routes)
			request := httptest.NewRequest(http.MethodPost, "/employees/E001/enrich", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d body=%s", response.Code, test.status, response.Body.String())
			}
			if test.code != "" {
				var envelope struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil || envelope.Error.Code != test.code {
					t.Fatalf("error envelope=%+v decode=%v, want %q", envelope, err, test.code)
				}
			} else {
				var profile domain.EmployeeProfile
				if err := json.NewDecoder(response.Body).Decode(&profile); err != nil || profile.EmployeeID != "E001" || profile.PublicFacts == nil || profile.RiskSignals == nil {
					t.Fatalf("success profile=%+v decode=%v", profile, err)
				}
			}
		})
	}
}

type fakeRepository struct {
	employee        domain.EmployeeDetails
	getEmployeeErr  error
	replaceErr      error
	getProfileErr   error
	replaceCalls    int
	getProfileCalls int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{employee: domain.EmployeeDetails{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", Email: "demo@example.test"}}
}

func fakeRepositoryWithReplaceError() *fakeRepository {
	repository := newFakeRepository()
	repository.replaceErr = errors.New("sql detail")
	return repository
}

func (repository *fakeRepository) ImportEmployees(context.Context, []domain.Employee) ([]bool, error) {
	return nil, nil
}
func (repository *fakeRepository) ListEmployees(context.Context) ([]domain.EmployeeSummary, error) {
	return nil, nil
}
func (repository *fakeRepository) GetEmployee(_ context.Context, _ string) (domain.EmployeeDetails, error) {
	return repository.employee, repository.getEmployeeErr
}
func (repository *fakeRepository) GetProfile(_ context.Context, _ string) (domain.EmployeeProfile, error) {
	repository.getProfileCalls++
	if repository.getProfileErr != nil {
		return domain.EmployeeProfile{}, repository.getProfileErr
	}
	return domain.EmployeeProfile{EmployeeID: "E001", DisplayName: "Demo User", Department: "Engineering", PublicFacts: []domain.PublicFact{}, RiskSignals: []string{}, RecommendedScenario: "persisted"}, nil
}
func (repository *fakeRepository) ReplaceProfile(_ context.Context, _ domain.EmployeeProfile) error {
	repository.replaceCalls++
	return repository.replaceErr
}

type fakeValidator struct {
	errors []error
	calls  int
}

func (validator *fakeValidator) ValidateProfile([]byte) error {
	validator.calls++
	if len(validator.errors) == 0 {
		return nil
	}
	err := validator.errors[0]
	validator.errors = validator.errors[1:]
	return err
}
func (validator *fakeValidator) ValidateScenario([]byte) error { return nil }

type scriptedAgent struct {
	outputs  [][]byte
	calls    int
	feedback *ports.ValidationFeedback
	err      error
}

func (agent *scriptedAgent) Generate(_ context.Context, _ ports.ProfileInput, feedback *ports.ValidationFeedback) ([]byte, error) {
	agent.calls++
	agent.feedback = feedback
	if agent.err != nil {
		return nil, agent.err
	}
	if len(agent.outputs) == 0 {
		return nil, fmt.Errorf("no scripted output")
	}
	output := agent.outputs[0]
	agent.outputs = agent.outputs[1:]
	return output, nil
}

type fixedAdapter struct {
	evidence []ports.Evidence
	err      error
}

func fixtureAdapterWithEvidence(evidence []ports.Evidence) *fixedAdapter {
	return &fixedAdapter{evidence: evidence}
}
func (adapter *fixedAdapter) Search(context.Context, domain.Employee) ([]ports.Evidence, error) {
	return cloneEvidence(adapter.evidence), adapter.err
}

func profileJSON(employeeID, displayName, department string) []byte {
	return []byte(fmt.Sprintf(`{"employeeId":%q,"displayName":%q,"department":%q,"publicFacts":[],"riskSignals":[],"recommendedScenario":"training_reminder"}`, employeeID, displayName, department))
}
