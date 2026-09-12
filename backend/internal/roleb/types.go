package roleb

import "errors"

const (
	EventOpened         EventType = "opened"
	EventClicked        EventType = "clicked"
	EventFormAttempted  EventType = "form_attempted"
	EventTrainingViewed EventType = "training_viewed"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidToken = errors.New("invalid tracking token")
	ErrInvalidEvent = errors.New("invalid event type")
	ErrConflict     = errors.New("state conflict")
)

type EventType string

func (e EventType) Valid() bool {
	switch e {
	case EventOpened, EventClicked, EventFormAttempted, EventTrainingViewed:
		return true
	default:
		return false
	}
}

type EventRequest struct {
	Token     string    `json:"token"`
	EventType EventType `json:"eventType"`
}

type LandingConfig struct {
	Title       string `json:"title"`
	Brand       string `json:"brand"`
	Description string `json:"description"`
	CTALabel    string `json:"ctaLabel"`
}

type LandingPageData struct {
	Token          string
	Title          string
	Brand          string
	Description    string
	CTALabel       string
	EventEndpoint  string
	CampaignID     string
	EducationTitle string
}

type SimulateResponse struct {
	CampaignID string           `json:"campaignId"`
	Status     string           `json:"status"`
	Targets    []SimulateTarget `json:"targets"`
}

type SimulateTarget struct {
	EmployeeID string `json:"employeeId"`
	Token      string `json:"token"`
	LandingURL string `json:"landingUrl"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}
