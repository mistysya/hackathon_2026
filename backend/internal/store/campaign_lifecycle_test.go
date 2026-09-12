package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

func TestCampaignLifecycleApproveAndRejectPersistCanonicalCampaign(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	approved := lifecycleCampaign("c_approved")
	if err := repository.CreateCampaign(ctx, approved); err != nil {
		t.Fatalf("create approval campaign: %v", err)
	}
	got, err := repository.ApproveCampaign(ctx, approved.CampaignID, "hr@example.test")
	if err != nil {
		t.Fatalf("ApproveCampaign: %v", err)
	}
	if got.Status != domain.CampaignStatusApproved || got.ApprovedBy == nil || *got.ApprovedBy != "hr@example.test" || got.ApprovedAt == nil || got.RejectionReason != nil {
		t.Fatalf("approved campaign = %+v", got)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal approved campaign: %v", err)
	}
	var frozen map[string]any
	if err := json.Unmarshal(data, &frozen); err != nil {
		t.Fatalf("unmarshal approved campaign: %v", err)
	}
	if frozen["rejectionReason"] != nil || frozen["approvedBy"] != "hr@example.test" || frozen["approvedAt"] == nil {
		t.Fatalf("frozen approval fields = %#v", frozen)
	}

	rejected := lifecycleCampaign("c_rejected")
	if err := repository.CreateCampaign(ctx, rejected); err != nil {
		t.Fatalf("create rejection campaign: %v", err)
	}
	got, err = repository.RejectCampaign(ctx, rejected.CampaignID, "needs a clearer call to action")
	if err != nil {
		t.Fatalf("RejectCampaign: %v", err)
	}
	if got.Status != domain.CampaignStatusRejected || got.ApprovedBy != nil || got.ApprovedAt != nil || got.RejectionReason == nil || *got.RejectionReason != "needs a clearer call to action" {
		t.Fatalf("rejected campaign = %+v", got)
	}
}

func TestCampaignLifecycleRejectsMissingInvalidOrUnsafeTransitions(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)

	if _, err := repository.ApproveCampaign(ctx, "c_missing", "hr@example.test"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing approve error = %v, want ErrNotFound", err)
	}
	for _, status := range []domain.CampaignStatus{domain.CampaignStatusApproved, domain.CampaignStatusRejected, domain.CampaignStatusSimulated} {
		campaign := lifecycleCampaign("c_" + string(status))
		campaign.Status = status
		if err := repository.CreateCampaign(ctx, campaign); err != nil {
			t.Fatalf("create %s campaign: %v", status, err)
		}
		if _, err := repository.RejectCampaign(ctx, campaign.CampaignID, "later"); !errors.Is(err, ErrConflict) {
			t.Errorf("reject %s error = %v, want ErrConflict", status, err)
		}
		if _, err := repository.ApproveCampaign(ctx, campaign.CampaignID, "hr@example.test"); !errors.Is(err, ErrConflict) {
			t.Errorf("approve %s error = %v, want ErrConflict", status, err)
		}
	}

	unsafe := lifecycleCampaign("c_unsafe")
	unsafe.SafetyChecks = unsafe.SafetyChecks[:4]
	if err := repository.CreateCampaign(ctx, unsafe); err != nil {
		t.Fatalf("create unsafe campaign: %v", err)
	}
	if _, err := repository.ApproveCampaign(ctx, unsafe.CampaignID, "hr@example.test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("missing safety check error = %v, want ErrConflict", err)
	}
	unsafe.SafetyChecks = lifecycleSafetyChecks()
	unsafe.SafetyChecks[0].Passed = false
	unsafe.CampaignID = "c_failed_check"
	if err := repository.CreateCampaign(ctx, unsafe); err != nil {
		t.Fatalf("create failed-check campaign: %v", err)
	}
	if _, err := repository.ApproveCampaign(ctx, unsafe.CampaignID, "hr@example.test"); !errors.Is(err, ErrConflict) {
		t.Fatalf("failed safety check error = %v, want ErrConflict", err)
	}
}

func TestCampaignLifecycleConcurrentTransitionsHaveAtMostOneSuccess(t *testing.T) {
	_, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)
	campaign := lifecycleCampaign("c_concurrent")
	if err := repository.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		_, err := repository.ApproveCampaign(ctx, campaign.CampaignID, "hr@example.test")
		results <- err
	}()
	go func() {
		defer group.Done()
		<-start
		_, err := repository.RejectCampaign(ctx, campaign.CampaignID, "not ready")
		results <- err
	}()
	close(start)
	group.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes > 1 {
		t.Fatalf("concurrent transitions had %d successes, want at most one", successes)
	}
	got, err := repository.GetCampaign(ctx, campaign.CampaignID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == domain.CampaignStatusPendingReview {
		t.Fatalf("campaign remained pending after concurrent transitions: %+v", got)
	}
}

func TestCampaignLifecycleUpdateFailureRollsBack(t *testing.T) {
	database, repository := newTestRepository(t)
	ctx := context.Background()
	importOneEmployee(t, repository)
	campaign := lifecycleCampaign("c_rollback")
	if err := repository.CreateCampaign(ctx, campaign); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
		CREATE TRIGGER fail_campaign_transition
		BEFORE UPDATE ON campaigns
		WHEN NEW.id = 'c_rollback'
		BEGIN
			SELECT RAISE(ABORT, 'forced transition failure');
		END
	`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	if _, err := repository.RejectCampaign(ctx, campaign.CampaignID, "not ready"); err == nil {
		t.Fatalf("RejectCampaign unexpectedly succeeded")
	}
	got, err := repository.GetCampaign(ctx, campaign.CampaignID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.CampaignStatusPendingReview || got.ApprovedBy != nil || got.ApprovedAt != nil || got.RejectionReason != nil {
		t.Fatalf("failed transition was not rolled back: %+v", got)
	}
}

func lifecycleCampaign(id string) domain.GeneratedCampaign {
	return domain.GeneratedCampaign{
		CampaignID: id, EmployeeID: "E001", TemplateID: "training_reminder", Difficulty: domain.DifficultyLow,
		Subject: "Training", EmailHTML: `<a href="{{landingUrl}}">Open</a>`,
		LandingConfig:  domain.LandingConfig{Title: "Training", Brand: "Security Awareness Demo", Description: "Description", CTALabel: "Open"},
		DecisionReason: "Fixture profile", SafetyChecks: lifecycleSafetyChecks(), Status: domain.CampaignStatusPendingReview,
	}
}

func lifecycleSafetyChecks() []domain.SafetyCheck {
	return []domain.SafetyCheck{
		{Rule: "no_real_credentials_requested", Passed: true},
		{Rule: "no_sensitive_personal_data", Passed: true},
		{Rule: "no_unsourced_personal_facts", Passed: true},
		{Rule: "no_prohibited_impersonation", Passed: true},
		{Rule: "cta_points_to_controlled_domain", Passed: true},
	}
}
