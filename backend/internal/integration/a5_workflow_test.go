package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/campaign"
	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/employee"
	"github.com/mistysya/hackathon_2026/backend/internal/httpapi"
	"github.com/mistysya/hackathon_2026/backend/internal/profile"
	"github.com/mistysya/hackathon_2026/backend/internal/roleb"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
	"github.com/mistysya/hackathon_2026/backend/internal/structured"
)

type errorEnvelope struct {
	Error struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
	} `json:"error"`
}

func TestFullWorkflowEndToEnd(t *testing.T) {
	database, repository := newSQLiteRepository(t)
	validator, err := structured.NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	fixtureAdapter, err := profile.NewFixtureAdapter()
	if err != nil {
		t.Fatalf("NewFixtureAdapter: %v", err)
	}
	fixtureScenarioAgent, err := campaign.NewFixtureScenarioAgent()
	if err != nil {
		t.Fatalf("NewFixtureScenarioAgent: %v", err)
	}

	employeeRoutes := employee.NewRoutes(employee.NewService(repository), slog.New(slog.NewTextHandler(io.Discard, nil)))
	profileRoutes := profile.NewRoutes(profile.NewService(
		repository,
		fixtureAdapter,
		profile.NewFixtureAgent(),
		validator,
	), slog.New(slog.NewTextHandler(io.Discard, nil)))
	campaignRoutes := campaign.NewRoutes(campaign.NewService(
		repository,
		repository,
		fixtureScenarioAgent,
		validator,
		campaign.NewCryptoIDGenerator(),
	), slog.New(slog.NewTextHandler(io.Discard, nil)))
	roleBRoutes := roleb.NewHandler(roleb.NewRepository(database))
	server := httptest.NewServer(httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		employeeRoutes,
		profileRoutes,
		campaignRoutes,
		roleBRoutes,
	))
	defer server.Close()

	importCSV := "employee_id,display_name,email,department,title,company\r\nE001,Demo User,demo.user@example.test,Engineering,Engineer,Demo Corp\r\n"
	importResult := doImportRequest(t, server, importCSV)
	if importResult.Imported != 1 || importResult.Skipped != 0 || len(importResult.Errors) != 0 {
		t.Fatalf("import result = %#v", importResult)
	}

	summaries := doListEmployees(t, server)
	if len(summaries) != 1 {
		t.Fatalf("list employees count = %d, want 1", len(summaries))
	}
	if summaries[0].EmployeeID != "E001" || summaries[0].HasProfile {
		t.Fatalf("employee summary = %#v, want hasProfile=false", summaries[0])
	}

	code, generateEnvelope := doGenerateRequest(t, server, "E001")
	if code != http.StatusConflict {
		t.Fatalf("generate before enrich status = %d, want 409", code)
	}
	if generateEnvelope.Error.Code != "profile_required" || generateEnvelope.Error.RequestID == "" {
		t.Fatalf("generate before enrich envelope = %#v", generateEnvelope)
	}

	profileResponse := doEnrichRequest(t, server, "E001")
	if profileResponse.EmployeeID != "E001" {
		t.Fatalf("enriched profile = %#v", profileResponse)
	}
	if len(profileResponse.PublicFacts) != 2 {
		t.Fatalf("enriched public facts = %#v", profileResponse.PublicFacts)
	}
	if !allSourceType(profileResponse.PublicFacts, domain.SourceTypeFixture) {
		t.Fatalf("enriched profile source types = %#v", profileResponse.PublicFacts)
	}

	summaries = doListEmployees(t, server)
	if !summaries[0].HasProfile {
		t.Fatalf("employee summary = %#v, want hasProfile=true", summaries[0])
	}
	details := doGetEmployeeDetails(t, server, "E001")
	if details.Profile == nil || details.Profile.EmployeeID != "E001" || details.Profile.DisplayName != "Demo User" || details.Profile.Department != "Engineering" {
		t.Fatalf("employee details = %#v", details)
	}
	if details.Profile.RecommendedScenario == "" || len(details.Profile.PublicFacts) != 2 || len(details.Profile.RiskSignals) == 0 {
		t.Fatalf("enriched profile shape = %#v", details.Profile)
	}

	generatedCampaign := doGenerateRequestOK(t, server, "E001")
	if generatedCampaign.Status != domain.CampaignStatusPendingReview {
		t.Fatalf("campaign status = %q, want %q", generatedCampaign.Status, domain.CampaignStatusPendingReview)
	}
	assertCampaignID(t, generatedCampaign.CampaignID)
	if len(generatedCampaign.SafetyChecks) != 5 || !allSafetyChecksPassed(generatedCampaign.SafetyChecks) {
		t.Fatalf("safety checks = %#v", generatedCampaign.SafetyChecks)
	}

	fetchedCampaign := doGetCampaign(t, server, generatedCampaign.CampaignID)
	if !reflect.DeepEqual(generatedCampaign, fetchedCampaign) {
		t.Fatalf("campaign mismatch\ngenerated=%#v\nfetched=%#v", generatedCampaign, fetchedCampaign)
	}
	if body, status := doRequest(t, server, http.MethodPost, "/campaigns/"+generatedCampaign.CampaignID+"/simulate", "", ""); status != http.StatusConflict {
		t.Fatalf("simulate before approval status = %d, body=%s", status, body)
	}

	approvedCampaign := doApproveCampaign(t, server, generatedCampaign.CampaignID, "hr@example.test")
	if approvedCampaign.Status != domain.CampaignStatusApproved || approvedCampaign.ApprovedBy == nil || *approvedCampaign.ApprovedBy != "hr@example.test" || approvedCampaign.ApprovedAt == nil {
		t.Fatalf("approved campaign = %#v", approvedCampaign)
	}
	duplicateApprovalBody := `{"approvedBy":"hr@example.test"}`
	if body, status := doRequest(t, server, http.MethodPost, "/campaigns/"+generatedCampaign.CampaignID+"/approve", duplicateApprovalBody, "application/json"); status != http.StatusConflict {
		t.Fatalf("duplicate approval status = %d, body=%s", status, body)
	}
	simulation := doSimulateCampaign(t, server, generatedCampaign.CampaignID)
	if simulation.Status != domain.CampaignStatusSimulated || len(simulation.Targets) != 1 || simulation.Targets[0].EmployeeID != "E001" {
		t.Fatalf("simulation = %#v", simulation)
	}
	for _, eventType := range []domain.EventType{domain.EventTypeOpened, domain.EventTypeClicked, domain.EventTypeFormAttempted, domain.EventTypeTrainingViewed, domain.EventTypeClicked} {
		doEventRequest(t, server, simulation.Targets[0].Token, eventType)
	}
	report := doCampaignReport(t, server, generatedCampaign.CampaignID)
	if report.TargetCount != 1 || report.Funnel != (domain.CampaignFunnel{Simulated: 1, Opened: 1, Clicked: 1, FormAttempted: 1, TrainingViewed: 1}) || len(report.Events) != 4 {
		t.Fatalf("campaign report = %#v", report)
	}

	rejectedCandidate := doGenerateRequestOK(t, server, "E001")
	rejectedCampaign := doRejectCampaign(t, server, rejectedCandidate.CampaignID, "內容需調整")
	if rejectedCampaign.Status != domain.CampaignStatusRejected || rejectedCampaign.RejectionReason == nil || *rejectedCampaign.RejectionReason != "內容需調整" {
		t.Fatalf("rejected campaign = %#v", rejectedCampaign)
	}

	duplicateImport := doImportRequest(t, server, importCSV)
	if duplicateImport.Imported != 0 || duplicateImport.Skipped != 1 {
		t.Fatalf("duplicate import result = %#v", duplicateImport)
	}
	summaries = doListEmployees(t, server)
	if len(summaries) != 1 {
		t.Fatalf("list employees after duplicate = %#v", summaries)
	}

	emptyDepartmentImport := doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE010,Jane,jane@example.test,,,\n")
	if emptyDepartmentImport.Imported != 1 || emptyDepartmentImport.Skipped != 0 {
		t.Fatalf("empty-department import result = %#v", emptyDepartmentImport)
	}
	emptyDepartmentProfile := doEnrichRequest(t, server, "E010")
	if emptyDepartmentProfile.Department != "" || emptyDepartmentProfile.RecommendedScenario != "training_reminder" {
		t.Fatalf("empty-department profile = %#v", emptyDepartmentProfile)
	}

	unknownGen := doGenerateRequestEnvelope(t, server, "E999")
	if unknownGen.Error.Code != "employee_not_found" || unknownGen.Error.RequestID == "" {
		t.Fatalf("unknown employee envelope = %#v", unknownGen)
	}
	unknownCampaign := doGetCampaignEnvelope(t, server, "c_unknown")
	if unknownCampaign.Error.Code != "campaign_not_found" || unknownCampaign.Error.RequestID == "" {
		t.Fatalf("unknown campaign envelope = %#v", unknownCampaign)
	}

}

func newSQLiteRepository(t *testing.T) (*sql.DB, *store.Repository) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "a5.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(path))
	database, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := store.Migrate(context.Background(), database); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return database, store.New(database)
}

func doImportRequest(t *testing.T, server *httptest.Server, body string) employee.ImportResponse {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodPost, "/employees/import", body, "text/csv")
	if status != http.StatusOK {
		t.Fatalf("import status = %d, body=%s", status, responseBody)
	}
	var parsed employee.ImportResponse
	if err := json.Unmarshal([]byte(responseBody), &parsed); err != nil {
		t.Fatalf("decode import response: %v", err)
	}
	return parsed
}

func doListEmployees(t *testing.T, server *httptest.Server) []domain.EmployeeSummary {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodGet, "/employees", "", "")
	if status != http.StatusOK {
		t.Fatalf("list employees status = %d, body=%s", status, responseBody)
	}
	var summaries []domain.EmployeeSummary
	if err := json.Unmarshal([]byte(responseBody), &summaries); err != nil {
		t.Fatalf("decode employee list: %v", err)
	}
	return summaries
}

func doGetEmployeeDetails(t *testing.T, server *httptest.Server, employeeID string) domain.EmployeeDetails {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodGet, "/employees/"+employeeID, "", "")
	if status != http.StatusOK {
		t.Fatalf("get employee status = %d, body=%s", status, responseBody)
	}
	var details domain.EmployeeDetails
	if err := json.Unmarshal([]byte(responseBody), &details); err != nil {
		t.Fatalf("decode employee details: %v", err)
	}
	return details
}

func doGenerateRequest(t *testing.T, server *httptest.Server, employeeID string) (int, errorEnvelope) {
	t.Helper()
	requestBody := `{"employeeId":` + toJSONString(employeeID) + `}`
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/generate", requestBody, "application/json")
	if status == http.StatusCreated {
		return status, errorEnvelope{}
	}
	var envelope errorEnvelope
	if err := json.Unmarshal([]byte(responseBody), &envelope); err != nil {
		t.Fatalf("decode generate envelope: %v, body=%s", err, responseBody)
	}
	return status, envelope
}

func doGenerateRequestOK(t *testing.T, server *httptest.Server, employeeID string) domain.GeneratedCampaign {
	t.Helper()
	requestBody := `{"employeeId":` + toJSONString(employeeID) + `}`
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/generate", requestBody, "application/json")
	if status != http.StatusCreated {
		t.Fatalf("generate status = %d, body=%s", status, responseBody)
	}
	var created domain.GeneratedCampaign
	if err := json.Unmarshal([]byte(responseBody), &created); err != nil {
		t.Fatalf("decode generated campaign: %v", err)
	}
	return created
}

func doGenerateRequestEnvelope(t *testing.T, server *httptest.Server, employeeID string) errorEnvelope {
	t.Helper()
	status, envelope := doGenerateRequest(t, server, employeeID)
	if status != http.StatusNotFound {
		t.Fatalf("generate status = %d, body envelope=%#v", status, envelope)
	}
	return envelope
}

func doEnrichRequest(t *testing.T, server *httptest.Server, employeeID string) domain.EmployeeProfile {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodPost, "/employees/"+employeeID+"/enrich", "{}", "application/json")
	if status != http.StatusOK {
		t.Fatalf("enrich status = %d, body=%s", status, responseBody)
	}
	var profileResponse domain.EmployeeProfile
	if err := json.Unmarshal([]byte(responseBody), &profileResponse); err != nil {
		t.Fatalf("decode enriched profile: %v", err)
	}
	return profileResponse
}

func doGetCampaign(t *testing.T, server *httptest.Server, campaignID string) domain.GeneratedCampaign {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodGet, "/campaigns/"+campaignID, "", "")
	if status != http.StatusOK {
		t.Fatalf("get campaign status = %d, body=%s", status, responseBody)
	}
	var got domain.GeneratedCampaign
	if err := json.Unmarshal([]byte(responseBody), &got); err != nil {
		t.Fatalf("decode campaign: %v", err)
	}
	return got
}

func doApproveCampaign(t *testing.T, server *httptest.Server, campaignID, approvedBy string) domain.GeneratedCampaign {
	t.Helper()
	body := `{"approvedBy":` + toJSONString(approvedBy) + `}`
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/"+campaignID+"/approve", body, "application/json")
	if status != http.StatusOK {
		t.Fatalf("approve campaign status = %d, body=%s", status, responseBody)
	}
	var approved domain.GeneratedCampaign
	if err := json.Unmarshal([]byte(responseBody), &approved); err != nil {
		t.Fatalf("decode approved campaign: %v", err)
	}
	return approved
}

func doRejectCampaign(t *testing.T, server *httptest.Server, campaignID, reason string) domain.GeneratedCampaign {
	t.Helper()
	body := `{"reason":` + toJSONString(reason) + `}`
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/"+campaignID+"/reject", body, "application/json")
	if status != http.StatusOK {
		t.Fatalf("reject campaign status = %d, body=%s", status, responseBody)
	}
	var rejected domain.GeneratedCampaign
	if err := json.Unmarshal([]byte(responseBody), &rejected); err != nil {
		t.Fatalf("decode rejected campaign: %v", err)
	}
	return rejected
}

func doSimulateCampaign(t *testing.T, server *httptest.Server, campaignID string) roleb.SimulateResponse {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/"+campaignID+"/simulate", "", "")
	if status != http.StatusOK {
		t.Fatalf("simulate campaign status = %d, body=%s", status, responseBody)
	}
	var simulation roleb.SimulateResponse
	if err := json.Unmarshal([]byte(responseBody), &simulation); err != nil {
		t.Fatalf("decode simulation: %v", err)
	}
	return simulation
}

func doEventRequest(t *testing.T, server *httptest.Server, token string, eventType domain.EventType) {
	t.Helper()
	body := `{"token":` + toJSONString(token) + `,"eventType":` + toJSONString(string(eventType)) + `}`
	responseBody, status := doRequest(t, server, http.MethodPost, "/events", body, "application/json")
	if status != http.StatusNoContent {
		t.Fatalf("event status = %d, body=%s", status, responseBody)
	}
}

func doCampaignReport(t *testing.T, server *httptest.Server, campaignID string) domain.CampaignReport {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodGet, "/reports/"+campaignID, "", "")
	if status != http.StatusOK {
		t.Fatalf("report status = %d, body=%s", status, responseBody)
	}
	var report domain.CampaignReport
	if err := json.Unmarshal([]byte(responseBody), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return report
}

func doGetCampaignEnvelope(t *testing.T, server *httptest.Server, campaignID string) errorEnvelope {
	t.Helper()
	responseBody, status := doRequest(t, server, http.MethodGet, "/campaigns/"+campaignID, "", "")
	if status != http.StatusNotFound {
		t.Fatalf("get campaign status = %d, body=%s", status, responseBody)
	}
	var envelope errorEnvelope
	if err := json.Unmarshal([]byte(responseBody), &envelope); err != nil {
		t.Fatalf("decode campaign envelope: %v, body=%s", err, responseBody)
	}
	return envelope
}

func doRequest(t *testing.T, server *httptest.Server, method string, path string, body string, contentType string) (string, int) {
	t.Helper()
	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()
	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(rawBody), response.StatusCode
}

func toJSONString(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func allSourceType(publicFacts []domain.PublicFact, expected domain.SourceType) bool {
	for _, fact := range publicFacts {
		if fact.SourceType != expected {
			return false
		}
	}
	return true
}

func allSafetyChecksPassed(checks []domain.SafetyCheck) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

func assertCampaignID(t *testing.T, campaignID string) {
	t.Helper()
	if matched, err := regexp.MatchString(`^c_[0-9a-f]{32}$`, campaignID); err != nil || !matched {
		t.Fatalf("campaign id = %q, want c_ + 32 lowercase hex chars", campaignID)
	}
}
