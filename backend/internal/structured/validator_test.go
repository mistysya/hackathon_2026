package structured

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestValidatorAcceptsFrozenShapes(t *testing.T) {
	validator := newValidator(t)
	if err := validator.ValidateProfile([]byte(validProfile)); err != nil {
		t.Fatalf("ValidateProfile: %v", err)
	}
	if err := validator.ValidateScenario([]byte(validScenario)); err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
}

func TestValidatorAcceptsAnEmptyOptionalDepartment(t *testing.T) {
	validator := newValidator(t)
	profileWithEmptyDepartment := strings.Replace(validProfile, `"department":"Engineering"`, `"department":""`, 1)
	if err := validator.ValidateProfile([]byte(profileWithEmptyDepartment)); err != nil {
		t.Fatalf("ValidateProfile() with empty department: %v", err)
	}
}

func TestValidatorRejectsInvalidProfileOutput(t *testing.T) {
	validator := newValidator(t)
	for name, raw := range map[string]string{
		"missing required":        strings.Replace(validProfile, "\n  \"department\":\"Engineering\",", "", 1),
		"unknown property":        strings.TrimSuffix(validProfile, "}") + `,"unexpected":true}`,
		"confidence out of range": strings.Replace(validProfile, `"confidence":0.75`, `"confidence":1.1`, 1),
		"null array":              strings.Replace(validProfile, `"riskSignals":["possible event interest"]`, `"riskSignals":null`, 1),
		"trailing document":       validProfile + " {}",
	} {
		t.Run(name, func(t *testing.T) {
			err := validator.ValidateProfile([]byte(raw))
			if err == nil {
				t.Fatal("ValidateProfile unexpectedly succeeded")
			}
			if strings.Contains(err.Error(), "possible event interest") {
				t.Fatalf("error exposed payload: %v", err)
			}
		})
	}
}

func TestValidatorRejectsInvalidScenarioOutput(t *testing.T) {
	validator := newValidator(t)
	for name, raw := range map[string]string{
		"unknown property":   strings.TrimSuffix(validScenario, "}") + `,"campaignId":"c_should_not_be_here"}`,
		"invalid difficulty": strings.Replace(validScenario, `"difficulty":"medium"`, `"difficulty":"extreme"`, 1),
		"blank required":     strings.Replace(validScenario, `"subject":"Demo training reminder"`, `"subject":"   "`, 1),
		"null array":         strings.Replace(validScenario, `"safetyChecks":[]`, `"safetyChecks":null`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if err := validator.ValidateScenario([]byte(raw)); err == nil {
				t.Fatal("ValidateScenario unexpectedly succeeded")
			}
		})
	}
}

func TestValidatorIsSafeForConcurrentReuse(t *testing.T) {
	validator := newValidator(t)
	var group sync.WaitGroup
	errors := make(chan error, 32)
	for index := 0; index < cap(errors); index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			errors <- validator.ValidateProfile([]byte(validProfile))
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent validation: %v", err)
		}
	}
}

func newValidator(t *testing.T) *Validator {
	t.Helper()
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	return validator
}

const validProfile = `{
  "employeeId":"E001",
  "displayName":"Demo User",
  "department":"Engineering",
  "publicFacts":[{"fact":"Presented at a public event","sourceUrl":"https://example.test/event","confidence":0.75,"sourceType":"fixture"}],
  "riskSignals":["possible event interest"],
  "recommendedScenario":"event_followup"
}`

const validScenario = `{
  "templateId":"event_followup",
  "difficulty":"medium",
  "subject":"Demo training reminder",
  "emailHtml":"<p>demo</p>",
  "landingConfig":{"title":"Training","brand":"Demo Training","description":"Learn safely","ctaLabel":"Continue"},
  "decisionReason":"Fixture evidence",
  "safetyChecks":[]
}`

func ExampleValidator_ValidateProfile() {
	validator, _ := NewValidator()
	fmt.Println(validator.ValidateProfile([]byte(validProfile)) == nil)
	// Output: true
}
