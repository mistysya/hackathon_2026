package roleb

import (
	"errors"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidToken = errors.New("invalid tracking token")
	ErrInvalidEvent = errors.New("invalid event type")
	ErrConflict     = errors.New("state conflict")
)

type LandingPageData struct {
	Token                string
	Title                string
	Brand                string
	Description          string
	CTALabel             string
	EventEndpoint        string
	CampaignID           string
	EducationTitle       string
	Theme                string
	PortalLabel          string
	Audience             string
	FormHeading          string
	PrimaryLabel         string
	PrimaryPlaceholder   string
	SecondaryLabel       string
	SecondaryPlaceholder string
}

type SimulateResponse struct {
	CampaignID string                `json:"campaignId"`
	Status     domain.CampaignStatus `json:"status"`
	Targets    []SimulateTarget      `json:"targets"`
}

type SimulateTarget struct {
	EmployeeID string `json:"employeeId"`
	Token      string `json:"token"`
	LandingURL string `json:"landingUrl"`
}
