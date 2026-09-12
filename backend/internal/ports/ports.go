package ports

import (
	"context"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

type EmployeeRepository interface {
	ImportEmployees(context.Context, []domain.Employee) ([]bool, error)
	ListEmployees(context.Context) ([]domain.EmployeeSummary, error)
	GetEmployee(context.Context, string) (domain.EmployeeDetails, error)
	GetProfile(context.Context, string) (domain.EmployeeProfile, error)
	ReplaceProfile(context.Context, domain.EmployeeProfile) error
}

type CampaignRepository interface {
	CreateCampaign(context.Context, domain.GeneratedCampaign) error
	GetCampaign(context.Context, string) (domain.GeneratedCampaign, error)
	ApproveCampaign(context.Context, string, string) error
	RejectCampaign(context.Context, string, string) error
}

type Evidence struct {
	Fact       string
	SourceURL  *string
	Confidence *float64
	SourceType domain.SourceType
	Tags       []string
}

type ProfileInput struct {
	Employee domain.Employee
	Evidence []Evidence
}

type ScenarioInput struct {
	Employee domain.Employee
	Profile  domain.EmployeeProfile
}

type ValidationFeedback struct {
	Message string
}

type EnrichmentAdapter interface {
	Search(context.Context, domain.Employee) ([]Evidence, error)
}

type ProfileAgent interface {
	Generate(context.Context, ProfileInput, *ValidationFeedback) ([]byte, error)
}

type ScenarioAgent interface {
	Generate(context.Context, ScenarioInput, *ValidationFeedback) ([]byte, error)
}

type StructuredValidator interface {
	ValidateProfile([]byte) error
	ValidateScenario([]byte) error
}
