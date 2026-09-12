package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

func TestMigrateIsIdempotentAndForeignKeysAreEnabled(t *testing.T) {
	database, _ := newTestRepository(t)
	ctx := context.Background()

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("second migration: %v", err)
	}

	var foreignKeys int
	if err := database.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var employeesTable string
	if err := database.QueryRowContext(ctx, `
		SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'employees'
	`).Scan(&employeesTable); err != nil {
		t.Fatalf("find employees table: %v", err)
	}
}

func TestEmployeeImportListGetAndDuplicateFlags(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	employees := []domain.Employee{
		{EmployeeID: "E002", DisplayName: "Second", Email: "second@example.test", Department: "Sales", Title: "Manager", Company: "Demo Corp"},
		{EmployeeID: "E001", DisplayName: "First", Email: "first@example.test", Department: "Engineering", Title: "Engineer", Company: "Demo Corp"},
		{EmployeeID: "E001", DisplayName: "Duplicate", Email: "duplicate@example.test"},
	}

	created, err := repository.ImportEmployees(ctx, employees)
	if err != nil {
		t.Fatalf("import employees: %v", err)
	}
	if want := []bool{true, true, false}; !reflect.DeepEqual(created, want) {
		t.Fatalf("created = %v, want %v", created, want)
	}

	summaries, err := repository.ListEmployees(ctx)
	if err != nil {
		t.Fatalf("list employees: %v", err)
	}
	if len(summaries) != 2 || summaries[0].EmployeeID != "E001" || summaries[1].EmployeeID != "E002" {
		t.Fatalf("employees are not sorted by employee ID: %+v", summaries)
	}
	if summaries[0].HasProfile {
		t.Fatal("new employee unexpectedly has a profile")
	}

	employee, err := repository.GetEmployee(ctx, "E001")
	if err != nil {
		t.Fatalf("get employee: %v", err)
	}
	if employee.DisplayName != "First" || employee.Profile != nil {
		t.Fatalf("unexpected employee: %+v", employee)
	}

	if _, err := repository.GetEmployee(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing employee error = %v, want ErrNotFound", err)
	}
	if _, err := repository.GetProfile(ctx, "E001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing profile error = %v, want ErrNotFound", err)
	}
}

func TestReplaceProfileReplacesFactsAndSetsHasProfile(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	sourceURL := "https://example.test/event"
	confidence := 0.88
	initial := domain.EmployeeProfile{
		EmployeeID:          "E001",
		PublicFacts:         []domain.PublicFact{{Fact: "event", SourceURL: &sourceURL, Confidence: &confidence, SourceType: domain.SourceTypeFixture}},
		RiskSignals:         []string{"possible event interest"},
		RecommendedScenario: "event_followup",
	}
	if err := repository.ReplaceProfile(ctx, initial); err != nil {
		t.Fatalf("replace initial profile: %v", err)
	}

	replacement := domain.EmployeeProfile{
		EmployeeID:          "E001",
		PublicFacts:         []domain.PublicFact{{Fact: "training", SourceType: domain.SourceTypeManual}},
		RiskSignals:         nil,
		RecommendedScenario: "training_reminder",
	}
	if err := repository.ReplaceProfile(ctx, replacement); err != nil {
		t.Fatalf("replace profile: %v", err)
	}

	profile, err := repository.GetProfile(ctx, "E001")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.DisplayName != "Demo User" || profile.Department != "Engineering" {
		t.Fatalf("profile identity must come from employee: %+v", profile)
	}
	if len(profile.PublicFacts) != 1 || profile.PublicFacts[0].Fact != "training" {
		t.Fatalf("old facts were not replaced: %+v", profile.PublicFacts)
	}
	if profile.RiskSignals == nil || len(profile.RiskSignals) != 0 {
		t.Fatalf("risk signals = %#v, want non-nil empty slice", profile.RiskSignals)
	}

	summaries, err := repository.ListEmployees(ctx)
	if err != nil {
		t.Fatalf("list employees: %v", err)
	}
	if len(summaries) != 1 || !summaries[0].HasProfile {
		t.Fatalf("hasProfile was not set: %+v", summaries)
	}
}

func TestReplaceProfileRollsBackAfterFactInsertFailure(t *testing.T) {
	database, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	initial := domain.EmployeeProfile{
		EmployeeID:          "E001",
		PublicFacts:         []domain.PublicFact{{Fact: "original", SourceType: domain.SourceTypeFixture}},
		RiskSignals:         []string{"original signal"},
		RecommendedScenario: "event_followup",
	}
	if err := repository.ReplaceProfile(ctx, initial); err != nil {
		t.Fatalf("replace initial profile: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		CREATE TRIGGER reject_failed_fact
		BEFORE INSERT ON public_facts
		WHEN NEW.fact = 'FAIL'
		BEGIN
			SELECT RAISE(ABORT, 'forced failure');
		END
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	failing := domain.EmployeeProfile{
		EmployeeID: "E001",
		PublicFacts: []domain.PublicFact{
			{Fact: "new", SourceType: domain.SourceTypeFixture},
			{Fact: "FAIL", SourceType: domain.SourceTypeFixture},
		},
		RiskSignals:         []string{"new signal"},
		RecommendedScenario: "training_reminder",
	}
	if err := repository.ReplaceProfile(ctx, failing); err == nil {
		t.Fatal("ReplaceProfile unexpectedly succeeded")
	}

	profile, err := repository.GetProfile(ctx, "E001")
	if err != nil {
		t.Fatalf("get profile after rollback: %v", err)
	}
	if len(profile.PublicFacts) != 1 || profile.PublicFacts[0].Fact != "original" || profile.RecommendedScenario != "event_followup" {
		t.Fatalf("profile replacement was not rolled back: %+v", profile)
	}
}

func TestCampaignRoundTripConflictsAndNotFound(t *testing.T) {
	database, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	approvedBy := "hr@example.test"
	approvedAt := "2026-09-12T14:03:11Z"
	detail := "fixture-safe"
	campaign := domain.GeneratedCampaign{
		CampaignID: "c_1234",
		EmployeeID: "E001",
		TemplateID: "event_followup",
		Difficulty: domain.DifficultyMedium,
		Subject:    "Subject",
		EmailHTML:  `<a href="{{landingUrl}}">Open</a>`,
		LandingConfig: domain.LandingConfig{
			Title: "Title", Brand: "Demo Training", Description: "Description", CTALabel: "Continue",
		},
		DecisionReason: "fixture evidence",
		SafetyChecks:   []domain.SafetyCheck{{Rule: "fixture_only", Passed: true, Detail: &detail}},
		Status:         domain.CampaignStatusApproved,
		ApprovedBy:     &approvedBy,
		ApprovedAt:     &approvedAt,
	}
	if err := repository.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	var senderName, senderAddress, subject, emailHTML string
	var deliveredAt sql.NullString
	if err := database.QueryRowContext(ctx, `
		SELECT sender_name, sender_address, subject, email_html, delivered_at
		FROM mailbox_messages WHERE campaign_id = ?
	`, campaign.CampaignID).Scan(&senderName, &senderAddress, &subject, &emailHTML, &deliveredAt); err != nil {
		t.Fatalf("read persisted mailbox message: %v", err)
	}
	if senderName != campaign.LandingConfig.Brand || senderAddress != MailboxSenderAddress || subject != campaign.Subject || emailHTML != campaign.EmailHTML || deliveredAt.Valid {
		t.Fatalf("unexpected mailbox row sender=%q address=%q subject=%q html=%q delivered=%v", senderName, senderAddress, subject, emailHTML, deliveredAt)
	}
	got, err := repository.GetCampaign(ctx, campaign.CampaignID)
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if !reflect.DeepEqual(got, campaign) {
		t.Fatalf("campaign round trip mismatch:\n got: %+v\nwant: %+v", got, campaign)
	}

	if err := repository.CreateCampaign(ctx, campaign); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate campaign error = %v, want ErrConflict", err)
	}
	if _, err := repository.GetCampaign(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing campaign error = %v, want ErrNotFound", err)
	}

	campaign.CampaignID = "c_missing_employee"
	campaign.EmployeeID = "missing"
	if err := repository.CreateCampaign(ctx, campaign); !errors.Is(err, ErrConflict) {
		t.Fatalf("foreign key error = %v, want ErrConflict", err)
	}
}

func TestCampaignApproveAndRejectTransitions(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	newPendingCampaign := func(id string) domain.GeneratedCampaign {
		return domain.GeneratedCampaign{
			CampaignID: id, EmployeeID: "E001", TemplateID: "training_reminder",
			Difficulty: domain.DifficultyLow, Subject: "Subject", EmailHTML: `<a href="{{landingUrl}}">Open</a>`,
			LandingConfig:  domain.LandingConfig{Title: "Title", Brand: "Security Awareness Demo", Description: "Description", CTALabel: "Open"},
			DecisionReason: "fixture", SafetyChecks: []domain.SafetyCheck{}, Status: domain.CampaignStatusPendingReview,
		}
	}

	approved := newPendingCampaign("c_approved")
	if err := repository.CreateCampaign(ctx, approved); err != nil {
		t.Fatal(err)
	}
	if err := repository.ApproveCampaign(ctx, approved.CampaignID, "hr@example.test"); err != nil {
		t.Fatalf("approve campaign: %v", err)
	}
	gotApproved, err := repository.GetCampaign(ctx, approved.CampaignID)
	if err != nil {
		t.Fatal(err)
	}
	if gotApproved.Status != domain.CampaignStatusApproved || gotApproved.ApprovedBy == nil || *gotApproved.ApprovedBy != "hr@example.test" || gotApproved.ApprovedAt == nil {
		t.Fatalf("approved campaign = %+v", gotApproved)
	}
	if _, err := time.Parse(time.RFC3339, *gotApproved.ApprovedAt); err != nil {
		t.Fatalf("approvedAt is not RFC3339: %q", *gotApproved.ApprovedAt)
	}
	if err := repository.RejectCampaign(ctx, approved.CampaignID, "too late"); !errors.Is(err, ErrConflict) {
		t.Fatalf("reject approved campaign error = %v, want ErrConflict", err)
	}

	rejected := newPendingCampaign("c_rejected")
	if err := repository.CreateCampaign(ctx, rejected); err != nil {
		t.Fatal(err)
	}
	if err := repository.RejectCampaign(ctx, rejected.CampaignID, "內容需調整"); err != nil {
		t.Fatalf("reject campaign: %v", err)
	}
	gotRejected, err := repository.GetCampaign(ctx, rejected.CampaignID)
	if err != nil {
		t.Fatal(err)
	}
	if gotRejected.Status != domain.CampaignStatusRejected || gotRejected.RejectionReason == nil || *gotRejected.RejectionReason != "內容需調整" {
		t.Fatalf("rejected campaign = %+v", gotRejected)
	}
	if err := repository.ApproveCampaign(ctx, "missing", "hr@example.test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("approve missing campaign error = %v, want ErrNotFound", err)
	}
}

func newTestRepository(t *testing.T) (*sql.DB, *Repository) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(path))
	database, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := Migrate(context.Background(), database); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return database, New(database)
}

func importOneEmployee(t *testing.T, repository *Repository) {
	t.Helper()
	created, err := repository.ImportEmployees(context.Background(), []domain.Employee{{
		EmployeeID:  "E001",
		DisplayName: "Demo User",
		Email:       "demo.user@example.test",
		Department:  "Engineering",
		Title:       "Software Engineer",
		Company:     "Demo Corp",
	}})
	if err != nil {
		t.Fatalf("import employee: %v", err)
	}
	if len(created) != 1 || !created[0] {
		t.Fatalf("unexpected import result: %v", created)
	}
}
