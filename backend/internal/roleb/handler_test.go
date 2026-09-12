package roleb

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func testServer(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	db, err := store.Open(context.Background(), "file:rolebtest?mode=memory&cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	seedApprovedCampaign(t, db)
	return db, NewHandler(NewRepository(db)).Routes()
}

func seedApprovedCampaign(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
INSERT INTO employees (employee_id, display_name, email, department, title, company)
VALUES ('E001', 'Demo User', 'demo.user@example.test', 'Engineering', 'Software Engineer', 'Demo Corp');
INSERT INTO campaigns (id, employee_id, template_id, status, difficulty, decision_reason, subject, email_html, landing_config_json, safety_checks_json, approved_by, approved_at)
VALUES ('c_demo', 'E001', 'event_followup', 'approved', 'medium', 'fixture', 'subject', '<a href="{{landingUrl}}">CTA</a>', '{"title":"AI 資安研討會後續問卷","brand":"Demo Corp Training (測試品牌)","description":"請填寫以下欄位以取得研討會簡報。","ctaLabel":"送出並下載"}', '[]', 'hr@example.test', '2026-09-12T14:03:11Z');`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestSimulateCreatesOpaqueTokenAndLandingURL(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/campaigns/c_demo/simulate", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, `"status":"simulated"`) || !strings.Contains(body, `"landingUrl":"/landing/`) {
		t.Fatalf("unexpected simulate response: %s", body)
	}
	var token string
	if err := db.QueryRow(`SELECT token FROM campaign_targets WHERE campaign_id = 'c_demo'`).Scan(&token); err != nil {
		t.Fatalf("target token: %v", err)
	}
	if len(token) != 32 || strings.Contains(token, "E001") {
		t.Fatalf("token should be 32 hex chars and contain no employee id, got %q", token)
	}
	var subject, emailHTML string
	var deliveredAt sql.NullString
	if err := db.QueryRow(`SELECT subject, email_html, delivered_at FROM mailbox_messages WHERE campaign_id = 'c_demo'`).Scan(&subject, &emailHTML, &deliveredAt); err != nil {
		t.Fatalf("read delivered mailbox message: %v", err)
	}
	if subject != "subject" || emailHTML != `<a href="{{landingUrl}}">CTA</a>` || !deliveredAt.Valid {
		t.Fatalf("unexpected delivered mailbox message subject=%q html=%q delivered=%v", subject, emailHTML, deliveredAt)
	}
}

func TestPostEventsIsIdempotentAndDoesNotStoreFormValues(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()
	_, err := db.Exec(`UPDATE campaigns SET status = 'simulated' WHERE id = 'c_demo'; INSERT INTO campaign_targets (campaign_id, employee_id, token) VALUES ('c_demo', 'E001', 'tok_demo');`)
	if err != nil {
		t.Fatalf("seed target: %v", err)
	}

	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"token":"tok_demo","eventType":"form_attempted"}`))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusNoContent {
			t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
		}
	}

	var count int
	var metadata string
	if err := db.QueryRow(`SELECT COUNT(*), COALESCE(MAX(metadata_json), '') FROM tracking_events WHERE event_type = 'form_attempted'`).Scan(&count, &metadata); err != nil {
		t.Fatalf("query events: %v", err)
	}
	if count != 1 {
		t.Fatalf("duplicate form_attempted should be ignored, count=%d", count)
	}
	if metadata != "{}" {
		t.Fatalf("metadata should not contain form field values, got %s", metadata)
	}
}

func TestLandingPageRecordsClickedOnlyAndRevealsEducationOnSubmit(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()
	_, err := db.Exec(`UPDATE campaigns SET status = 'simulated' WHERE id = 'c_demo'; INSERT INTO campaign_targets (campaign_id, employee_id, token) VALUES ('c_demo', 'E001', 'tok_demo');`)
	if err != nil {
		t.Fatalf("seed target: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/landing/tok_demo", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	html := res.Body.String()
	checks := []string{
		`const token = "tok_demo";`,
		`const endpoint = "/events";`,
		"sendEvent(\"clicked\")",
		"sendEvent(\"form_attempted\")",
		"sendEvent(\"training_viewed\")",
		"表單欄位沒有 name 屬性",
		"這是一場受控資安演練",
	}
	for _, want := range checks {
		if !strings.Contains(html, want) {
			t.Fatalf("landing html missing %q", want)
		}
	}
	if strings.Contains(html, "sendEvent(\"opened\")") {
		t.Fatal("landing page must not record opened; email preview owns that event")
	}
}

func TestPostEventRejectsTrailingJSONWithoutWritingEvent(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()
	_, err := db.Exec(`UPDATE campaigns SET status = 'simulated' WHERE id = 'c_demo'; INSERT INTO campaign_targets (campaign_id, employee_id, token) VALUES ('c_demo', 'E001', 'tok_demo');`)
	if err != nil {
		t.Fatalf("seed target: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"token":"tok_demo","eventType":"clicked"}{}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tracking_events WHERE campaign_id = 'c_demo'`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("trailing JSON request wrote %d events", count)
	}
}

func TestCampaignReportReturnsDistinctFunnelAndTimeline(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()
	_, err := db.Exec(`
UPDATE campaigns SET status = 'simulated' WHERE id = 'c_demo';
INSERT INTO campaign_targets (campaign_id, employee_id, token) VALUES ('c_demo', 'E001', 'tok_demo');
INSERT INTO tracking_events (campaign_id, employee_id, event_type, occurred_at) VALUES
  ('c_demo', 'E001', 'opened', '2026-09-12T14:03:11Z'),
  ('c_demo', 'E001', 'clicked', '2026-09-12T14:03:29Z'),
  ('c_demo', 'E001', 'form_attempted', '2026-09-12T14:03:52Z'),
  ('c_demo', 'E001', 'training_viewed', '2026-09-12T14:03:54Z');`)
	if err != nil {
		t.Fatalf("seed report: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/c_demo", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}

	var report domain.CampaignReport
	if err := json.NewDecoder(res.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.CampaignID != "c_demo" || report.TargetCount != 1 {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if report.Funnel != (domain.CampaignFunnel{Simulated: 1, Opened: 1, Clicked: 1, FormAttempted: 1, TrainingViewed: 1}) {
		t.Fatalf("unexpected funnel: %+v", report.Funnel)
	}
	if len(report.Events) != 4 || report.Events[0].EventType != domain.EventTypeOpened || report.Events[3].EventType != domain.EventTypeTrainingViewed {
		t.Fatalf("unexpected timeline: %+v", report.Events)
	}
}

func TestCampaignReportReturnsNotFound(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/reports/missing", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestCampaignReportReturnsEmptyReportForCampaignWithoutTargetsOrEvents(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodGet, "/reports/c_demo", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}

	var report domain.CampaignReport
	if err := json.NewDecoder(res.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.TargetCount != 0 || report.Funnel != (domain.CampaignFunnel{}) {
		t.Fatalf("unexpected empty report counts: %+v", report)
	}
	if report.Events == nil || len(report.Events) != 0 {
		t.Fatalf("events = %#v, want a non-nil empty slice", report.Events)
	}
}

func TestCampaignReportRejectsInvalidStoredData(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()
	if _, err := db.Exec(`
INSERT INTO tracking_events (campaign_id, employee_id, event_type, occurred_at)
VALUES ('c_demo', 'E001', 'opened', '2026-09-12T18:00:00+08:00')`); err != nil {
		t.Fatalf("seed invalid event: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/reports/c_demo", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("unexpected error body: %s", res.Body.String())
	}
}

func TestPostEventRejectsMissingTokenAsNotFound(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"eventType":"clicked"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
}

func TestPostEventRejectsInvalidEventType(t *testing.T) {
	db, handler := testServer(t)
	defer db.Close()

	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"token":"tok_demo","eventType":"credential_submitted"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", res.Code, res.Body.String())
	}
}
