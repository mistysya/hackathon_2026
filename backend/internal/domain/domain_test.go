package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnumsValidateFrozenValues(t *testing.T) {
	for _, value := range []SourceType{SourceTypeLive, SourceTypeFixture, SourceTypeManual} {
		if !value.Valid() {
			t.Fatalf("SourceType %q should be valid", value)
		}
	}
	if SourceType("other").Valid() {
		t.Fatal("unexpected valid SourceType")
	}

	for _, value := range []CampaignStatus{CampaignStatusPendingReview, CampaignStatusApproved, CampaignStatusRejected, CampaignStatusSimulated} {
		if !value.Valid() {
			t.Fatalf("CampaignStatus %q should be valid", value)
		}
	}
	if CampaignStatus("draft").Valid() {
		t.Fatal("draft must not be a valid frozen status")
	}

	for _, value := range []Difficulty{DifficultyLow, DifficultyMedium, DifficultyHigh} {
		if !value.Valid() {
			t.Fatalf("Difficulty %q should be valid", value)
		}
	}
	if Difficulty("extreme").Valid() {
		t.Fatal("unexpected valid Difficulty")
	}

	for _, value := range []EventType{EventTypeOpened, EventTypeClicked, EventTypeFormAttempted, EventTypeTrainingViewed} {
		if !value.Valid() {
			t.Fatalf("EventType %q should be valid", value)
		}
	}
	if EventType("submitted").Valid() {
		t.Fatal("unexpected valid EventType")
	}
}

func TestCampaignReportMarshalsFrozenFieldNamesAndEmptyEvents(t *testing.T) {
	data, err := json.Marshal(CampaignReport{CampaignID: "c_1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(data)
	for _, fragment := range []string{
		`"campaignId":"c_1"`,
		`"targetCount":0`,
		`"formAttempted":0`,
		`"trainingViewed":0`,
		`"events":[]`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("missing %s in %s", fragment, text)
		}
	}
}

func TestEmployeeProfileMarshalsEmptySlices(t *testing.T) {
	data, err := json.Marshal(EmployeeProfile{EmployeeID: "E001"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"publicFacts":[]`) || !strings.Contains(text, `"riskSignals":[]`) {
		t.Fatalf("empty slices must marshal as arrays: %s", text)
	}
}

func TestCampaignMarshalsFrozenFieldNamesAndNulls(t *testing.T) {
	data, err := json.Marshal(GeneratedCampaign{CampaignID: "c_1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := string(data)
	for _, fragment := range []string{
		`"campaignId":"c_1"`,
		`"landingConfig"`,
		`"safetyChecks":[]`,
		`"approvedBy":null`,
		`"approvedAt":null`,
		`"rejectionReason":null`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("missing %s in %s", fragment, text)
		}
	}
}
