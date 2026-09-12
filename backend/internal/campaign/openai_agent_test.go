package campaign

import "testing"

func TestSafeCopyAllowsValidGeneratedCopy(t *testing.T) {
	output := struct {
		TemplateID, SenderPersona, Subject string
		EmailBody                          []string `json:"emailBody"`
		CTALabel                           string   `json:"ctaLabel"`
		LandingTitle                       string   `json:"landingTitle"`
		LandingDescription                 string   `json:"landingDescription"`
		DecisionReason                     string   `json:"decisionReason"`
	}{
		TemplateID:         "training_reminder",
		SenderPersona:      "demo_learning",
		Subject:            "Complete your security training",
		EmailBody:          []string{"A short training module is ready for you.", "It takes only a few minutes to complete."},
		CTALabel:           "Open training",
		LandingTitle:       "Security training reminder",
		LandingDescription: "Complete this demo training module.",
		DecisionReason:     "A short learning reminder suits this demo profile.",
	}

	if !safeCopy(output) {
		t.Fatal("safeCopy rejected valid generated copy")
	}
}
