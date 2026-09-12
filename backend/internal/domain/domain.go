package domain

import "encoding/json"

type SourceType string

const (
	SourceTypeLive    SourceType = "live"
	SourceTypeFixture SourceType = "fixture"
	SourceTypeManual  SourceType = "manual"
)

func (value SourceType) Valid() bool {
	switch value {
	case SourceTypeLive, SourceTypeFixture, SourceTypeManual:
		return true
	default:
		return false
	}
}

type CampaignStatus string

const (
	CampaignStatusPendingReview CampaignStatus = "pending_review"
	CampaignStatusApproved      CampaignStatus = "approved"
	CampaignStatusRejected      CampaignStatus = "rejected"
	CampaignStatusSimulated     CampaignStatus = "simulated"
)

func (value CampaignStatus) Valid() bool {
	switch value {
	case CampaignStatusPendingReview, CampaignStatusApproved, CampaignStatusRejected, CampaignStatusSimulated:
		return true
	default:
		return false
	}
}

type Difficulty string

const (
	DifficultyLow    Difficulty = "low"
	DifficultyMedium Difficulty = "medium"
	DifficultyHigh   Difficulty = "high"
)

func (value Difficulty) Valid() bool {
	switch value {
	case DifficultyLow, DifficultyMedium, DifficultyHigh:
		return true
	default:
		return false
	}
}

// EventType is the deduplicated funnel event emitted by the landing experience.
// Its values are fixed by docs/api-contract.md.
type EventType string

const (
	EventTypeOpened         EventType = "opened"
	EventTypeClicked        EventType = "clicked"
	EventTypeFormAttempted  EventType = "form_attempted"
	EventTypeTrainingViewed EventType = "training_viewed"
)

func (value EventType) Valid() bool {
	switch value {
	case EventTypeOpened, EventTypeClicked, EventTypeFormAttempted, EventTypeTrainingViewed:
		return true
	default:
		return false
	}
}

type Employee struct {
	EmployeeID  string `json:"employeeId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Company     string `json:"company"`
}

type EmployeeSummary struct {
	EmployeeID  string `json:"employeeId"`
	DisplayName string `json:"displayName"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	HasProfile  bool   `json:"hasProfile"`
}

type EmployeeDetails struct {
	EmployeeID  string           `json:"employeeId"`
	DisplayName string           `json:"displayName"`
	Email       string           `json:"email"`
	Department  string           `json:"department"`
	Title       string           `json:"title"`
	Company     string           `json:"company"`
	Profile     *EmployeeProfile `json:"profile"`
}

type PublicFact struct {
	Fact       string     `json:"fact"`
	SourceURL  *string    `json:"sourceUrl"`
	Confidence *float64   `json:"confidence"`
	SourceType SourceType `json:"sourceType"`
}

type EmployeeProfile struct {
	EmployeeID          string       `json:"employeeId"`
	DisplayName         string       `json:"displayName"`
	Department          string       `json:"department"`
	PublicFacts         []PublicFact `json:"publicFacts"`
	RiskSignals         []string     `json:"riskSignals"`
	RecommendedScenario string       `json:"recommendedScenario"`
}

func (profile EmployeeProfile) MarshalJSON() ([]byte, error) {
	type alias EmployeeProfile
	if profile.PublicFacts == nil {
		profile.PublicFacts = []PublicFact{}
	}
	if profile.RiskSignals == nil {
		profile.RiskSignals = []string{}
	}
	return json.Marshal(alias(profile))
}

type LandingConfig struct {
	Title       string `json:"title"`
	Brand       string `json:"brand"`
	Description string `json:"description"`
	CTALabel    string `json:"ctaLabel"`
}

type SafetyCheck struct {
	Rule   string  `json:"rule"`
	Passed bool    `json:"passed"`
	Detail *string `json:"detail,omitempty"`
}

type GeneratedCampaign struct {
	CampaignID      string         `json:"campaignId"`
	EmployeeID      string         `json:"employeeId"`
	TemplateID      string         `json:"templateId"`
	Difficulty      Difficulty     `json:"difficulty"`
	Subject         string         `json:"subject"`
	EmailHTML       string         `json:"emailHtml"`
	LandingConfig   LandingConfig  `json:"landingConfig"`
	DecisionReason  string         `json:"decisionReason"`
	SafetyChecks    []SafetyCheck  `json:"safetyChecks"`
	Status          CampaignStatus `json:"status"`
	ApprovedBy      *string        `json:"approvedBy"`
	ApprovedAt      *string        `json:"approvedAt"`
	RejectionReason *string        `json:"rejectionReason"`
}

// EventRequest is intentionally free of employee and campaign identifiers:
// the event endpoint resolves those from the opaque target token.
type EventRequest struct {
	Token     string    `json:"token"`
	EventType EventType `json:"eventType"`
}

type CampaignFunnel struct {
	Simulated      int `json:"simulated"`
	Opened         int `json:"opened"`
	Clicked        int `json:"clicked"`
	FormAttempted  int `json:"formAttempted"`
	TrainingViewed int `json:"trainingViewed"`
}

type CampaignEvent struct {
	EventType  EventType `json:"eventType"`
	OccurredAt string    `json:"occurredAt"`
}

type CampaignReport struct {
	CampaignID  string          `json:"campaignId"`
	TargetCount int             `json:"targetCount"`
	Funnel      CampaignFunnel  `json:"funnel"`
	Events      []CampaignEvent `json:"events"`
}

func (report CampaignReport) MarshalJSON() ([]byte, error) {
	type alias CampaignReport
	if report.Events == nil {
		report.Events = []CampaignEvent{}
	}
	return json.Marshal(alias(report))
}

func (campaign GeneratedCampaign) MarshalJSON() ([]byte, error) {
	type alias GeneratedCampaign
	if campaign.SafetyChecks == nil {
		campaign.SafetyChecks = []SafetyCheck{}
	}
	return json.Marshal(alias(campaign))
}
