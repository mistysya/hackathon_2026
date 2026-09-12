package integration

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/openaiapi"
)

func TestNewRouterMountsFullFixtureWorkflow(t *testing.T) {
	database, _ := newSQLiteRepository(t)
	router, err := NewRouter(database, slog.New(slog.NewTextHandler(io.Discard, nil)), openaiapi.Config{Mode: openaiapi.ModeFixture}, nil)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	server := httptest.NewServer(router)
	defer server.Close()

	doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE001,Demo User,demo@example.test,Engineering,Engineer,Demo Corp\n")
	doEnrichRequest(t, server, "E001")
	campaign := doGenerateRequestOK(t, server, "E001")
	if body, status := doRequest(t, server, http.MethodPost, "/campaigns/"+campaign.CampaignID+"/approve", `{"approvedBy":"hr@example.test"}`, "application/json"); status != http.StatusOK {
		t.Fatalf("approve status=%d body=%s", status, body)
	}
	responseBody, status := doRequest(t, server, http.MethodPost, "/campaigns/"+campaign.CampaignID+"/simulate", "", "")
	if status != http.StatusOK {
		t.Fatalf("simulate status=%d body=%s", status, responseBody)
	}
	var simulated struct {
		Targets []struct {
			Token string `json:"token"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(responseBody), &simulated); err != nil || len(simulated.Targets) != 1 {
		t.Fatalf("decode simulation err=%v response=%s", err, responseBody)
	}
	if body, status := doRequest(t, server, http.MethodPost, "/events", `{"token":`+toJSONString(simulated.Targets[0].Token)+`,"eventType":"opened"}`, "application/json"); status != http.StatusNoContent {
		t.Fatalf("record event status=%d body=%s", status, body)
	}
	reportBody, status := doRequest(t, server, http.MethodGet, "/reports/"+campaign.CampaignID, "", "")
	if status != http.StatusOK {
		t.Fatalf("report status=%d body=%s", status, reportBody)
	}
	var report domain.CampaignReport
	if err := json.Unmarshal([]byte(reportBody), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.TargetCount != 1 || report.Funnel.Simulated != 1 || report.Funnel.Opened != 1 || len(report.Events) != 1 {
		t.Fatalf("report=%#v", report)
	}

	for _, path := range []string{"/campaigns/c_missing/approve", "/reports/c_missing"} {
		method, contentType, body := http.MethodGet, "", ""
		if strings.HasSuffix(path, "/approve") {
			method, contentType, body = http.MethodPost, "application/json", `{"approvedBy":"hr@example.test"}`
		}
		raw, code := doRequest(t, server, method, path, body, contentType)
		if code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, code, raw)
		}
		var envelope errorEnvelope
		if err := json.Unmarshal([]byte(raw), &envelope); err != nil || envelope.Error.Code != "campaign_not_found" || envelope.Error.RequestID == "" {
			t.Fatalf("%s envelope=%s err=%v", path, raw, err)
		}
	}
}

func TestNewRouterAutoWithoutLiveConfigUsesFixtures(t *testing.T) {
	database, _ := newSQLiteRepository(t)
	router, err := NewRouter(database, slog.New(slog.NewTextHandler(io.Discard, nil)), openaiapi.Config{Mode: openaiapi.ModeAuto}, nil)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	server := httptest.NewServer(router)
	defer server.Close()

	doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE001,Demo User,demo@example.test,Engineering,Engineer,Demo Corp\n")
	profile := doEnrichRequest(t, server, "E001")
	if len(profile.PublicFacts) == 0 || profile.PublicFacts[0].SourceType != domain.SourceTypeFixture {
		t.Fatalf("profile facts=%#v want fixture-backed facts", profile.PublicFacts)
	}
	if campaign := doGenerateRequestOK(t, server, "E001"); campaign.Status != domain.CampaignStatusPendingReview {
		t.Fatalf("campaign=%#v", campaign)
	}
}

func TestNewRouterAutoUsesLiveResponsesAndFallsBack(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		wantSource domain.SourceType
	}{
		{name: "live success", status: http.StatusOK, wantSource: domain.SourceTypeLive},
		{name: "authentication fallback", status: http.StatusUnauthorized, wantSource: domain.SourceTypeFixture},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Fatal("missing bearer authorization")
				}
				if test.status != http.StatusOK {
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(`{"error":{"code":"invalid_api_key"}}`))
					return
				}
				var request struct {
					Text struct {
						Format struct {
							Name string `json:"name"`
						} `json:"format"`
					} `json:"text"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				output := map[string]string{
					"public_evidence":  `{"evidence":[{"fact":"Speaker at a public event","sourceUrl":"https://example.test/event","tags":["event"]}]}`,
					"profile_analysis": `{"riskSignals":["Public event follow-up is suitable."],"recommendedScenario":"event_followup"}`,
					"campaign_copy":    `{"templateId":"event_followup","senderPersona":"demo_events","subject":"Demo follow-up","emailBody":["Please review this demo."],"ctaLabel":"Open demo","landingTitle":"Demo follow-up","landingDescription":"A controlled test.","decisionReason":"Public event reminder."}`,
				}[request.Text.Format.Name]
				_, _ = w.Write([]byte(`{"id":"resp_test","model":"test-model","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":` + toJSONString(output) + `}]}]}`))
			}))
			defer provider.Close()

			database, _ := newSQLiteRepository(t)
			router, err := NewRouter(database, slog.New(slog.NewTextHandler(io.Discard, nil)), openaiapi.Config{Mode: openaiapi.ModeAuto, APIKey: "test-key", Model: "test-model", BaseURL: provider.URL}, provider.Client())
			if err != nil {
				t.Fatalf("NewRouter: %v", err)
			}
			server := httptest.NewServer(router)
			defer server.Close()
			doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE001,Demo User,demo@example.test,Engineering,Engineer,Demo Corp\n")
			profile := doEnrichRequest(t, server, "E001")
			if len(profile.PublicFacts) == 0 || profile.PublicFacts[0].SourceType != test.wantSource {
				t.Fatalf("profile facts=%#v want source=%q", profile.PublicFacts, test.wantSource)
			}
			campaign := doGenerateRequestOK(t, server, "E001")
			if campaign.Status != domain.CampaignStatusPendingReview {
				t.Fatalf("campaign=%#v", campaign)
			}
			if calls.Load() != 3 {
				t.Fatalf("provider calls=%d want 3", calls.Load())
			}
		})
	}
}

func TestNewRouterLiveRequiredDoesNotFallback(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"invalid_api_key"}}`))
	}))
	defer provider.Close()
	database, _ := newSQLiteRepository(t)
	router, err := NewRouter(database, slog.New(slog.NewTextHandler(io.Discard, nil)), openaiapi.Config{Mode: openaiapi.ModeLiveRequired, APIKey: "test-key", Model: "test-model", BaseURL: provider.URL}, provider.Client())
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE001,Demo User,demo@example.test,Engineering,Engineer,Demo Corp\n")
	body, status := doRequest(t, server, http.MethodPost, "/employees/E001/enrich", "{}", "application/json")
	if status != http.StatusBadGateway || !strings.Contains(body, `"enrichment_failed"`) {
		t.Fatalf("live_required enrich status=%d body=%s", status, body)
	}
}

func TestNewRouterAutoWithoutKeyUsesFixture(t *testing.T) {
	database, _ := newSQLiteRepository(t)
	router, err := NewRouter(database, slog.New(slog.NewTextHandler(io.Discard, nil)), openaiapi.Config{Mode: openaiapi.ModeAuto}, nil)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	doImportRequest(t, server, "employee_id,display_name,email,department,title,company\nE001,Demo User,demo@example.test,Engineering,Engineer,Demo Corp\n")
	profile := doEnrichRequest(t, server, "E001")
	if len(profile.PublicFacts) == 0 || profile.PublicFacts[0].SourceType != domain.SourceTypeFixture {
		t.Fatalf("auto-without-key profile facts=%#v want fixture source", profile.PublicFacts)
	}
}
