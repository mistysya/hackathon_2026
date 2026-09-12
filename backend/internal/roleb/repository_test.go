package roleb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func newReportRepository(t *testing.T) (*sql.DB, *Repository) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "roleb-report.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(path))
	db, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db, NewRepository(db)
}

func seedReportCampaign(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO employees (employee_id, display_name, email) VALUES
  ('E001', 'First User', 'first@example.test'),
  ('E002', 'Second User', 'second@example.test');
INSERT INTO campaigns (id, employee_id, template_id, status)
VALUES ('campaign_report', 'E001', 'event_followup', 'simulated');`); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
}

func TestCampaignReportRepositoryNotFound(t *testing.T) {
	_, repository := newReportRepository(t)

	if _, err := repository.CampaignReport(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CampaignReport error = %v, want ErrNotFound", err)
	}
}

func TestCampaignReportRepositoryEmptyCampaignHasZeroCountsAndEvents(t *testing.T) {
	db, repository := newReportRepository(t)
	seedReportCampaign(t, db)

	report, err := repository.CampaignReport(context.Background(), "campaign_report")
	if err != nil {
		t.Fatalf("CampaignReport: %v", err)
	}
	if report.CampaignID != "campaign_report" || report.TargetCount != 0 || report.Funnel != (domain.CampaignFunnel{}) {
		t.Fatalf("unexpected empty report: %+v", report)
	}
	if report.Events == nil || len(report.Events) != 0 {
		t.Fatalf("events = %#v, want a non-nil empty slice", report.Events)
	}
}

func TestCampaignReportRepositoryUsesDistinctEmployeesAndStableTimeline(t *testing.T) {
	db, repository := newReportRepository(t)
	seedReportCampaign(t, db)
	if _, err := db.Exec(`
INSERT INTO campaign_targets (campaign_id, employee_id, token) VALUES
  ('campaign_report', 'E001', 'token_one'),
  ('campaign_report', 'E002', 'token_two');
INSERT INTO tracking_events (campaign_id, employee_id, event_type, occurred_at) VALUES
  ('campaign_report', 'E002', 'opened', '2026-09-12T10:00:00Z'),
  ('campaign_report', 'E001', 'opened', '2026-09-12T09:00:00Z'),
  ('campaign_report', 'E001', 'clicked', '2026-09-12T10:00:00Z'),
  ('campaign_report', 'E002', 'training_viewed', '2026-09-12T10:00:00Z');`); err != nil {
		t.Fatalf("seed report data: %v", err)
	}

	report, err := repository.CampaignReport(context.Background(), "campaign_report")
	if err != nil {
		t.Fatalf("CampaignReport: %v", err)
	}
	if report.TargetCount != 2 {
		t.Fatalf("target count = %d, want 2", report.TargetCount)
	}
	wantFunnel := domain.CampaignFunnel{Simulated: 2, Opened: 2, Clicked: 1, TrainingViewed: 1}
	if report.Funnel != wantFunnel {
		t.Fatalf("funnel = %+v, want %+v", report.Funnel, wantFunnel)
	}
	wantEvents := []domain.CampaignEvent{
		{EventType: domain.EventTypeOpened, OccurredAt: "2026-09-12T09:00:00Z"},
		{EventType: domain.EventTypeOpened, OccurredAt: "2026-09-12T10:00:00Z"},
		{EventType: domain.EventTypeClicked, OccurredAt: "2026-09-12T10:00:00Z"},
		{EventType: domain.EventTypeTrainingViewed, OccurredAt: "2026-09-12T10:00:00Z"},
	}
	if !reflect.DeepEqual(report.Events, wantEvents) {
		t.Fatalf("events = %+v, want %+v", report.Events, wantEvents)
	}
}

func TestCampaignReportRepositoryRejectsInvalidStoredEvents(t *testing.T) {
	for name, event := range map[string]struct {
		eventType  string
		occurredAt string
	}{
		"unknown event type":  {eventType: "credential_submitted", occurredAt: "2026-09-12T10:00:00Z"},
		"non UTC timestamp":   {eventType: "opened", occurredAt: "2026-09-12T18:00:00+08:00"},
		"malformed timestamp": {eventType: "opened", occurredAt: "not-a-timestamp"},
	} {
		t.Run(name, func(t *testing.T) {
			db, repository := newReportRepository(t)
			seedReportCampaign(t, db)
			if _, err := db.Exec(`
INSERT INTO tracking_events (campaign_id, employee_id, event_type, occurred_at)
VALUES (?, 'E001', ?, ?)`, "campaign_report", event.eventType, event.occurredAt); err != nil {
				t.Fatalf("seed invalid event: %v", err)
			}

			report, err := repository.CampaignReport(context.Background(), "campaign_report")
			if err == nil {
				t.Fatalf("CampaignReport unexpectedly succeeded: %+v", report)
			}
			if report.CampaignID != "" || report.TargetCount != 0 || report.Funnel != (domain.CampaignFunnel{}) || report.Events != nil {
				t.Fatalf("CampaignReport returned partial report: %+v", report)
			}
		})
	}
}
