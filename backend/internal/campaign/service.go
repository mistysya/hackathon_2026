package campaign

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

var requiredSafetyRules = map[string]struct{}{
	"no_real_credentials_requested":   {},
	"no_sensitive_personal_data":      {},
	"no_unsourced_personal_facts":     {},
	"no_prohibited_impersonation":     {},
	"cta_points_to_controlled_domain": {},
}

var allowedTemplateIDs = map[string]struct{}{
	"event_followup": {}, "training_reminder": {}, "benefit_update": {}, "saas_security_notice": {},
}

type Service struct {
	employeeRepository ports.EmployeeRepository
	campaignRepository ports.CampaignRepository
	agent              ports.ScenarioAgent
	validator          ports.StructuredValidator
	idGenerator        IDGenerator
}

func NewService(
	employeeRepository ports.EmployeeRepository,
	campaignRepository ports.CampaignRepository,
	agent ports.ScenarioAgent,
	validator ports.StructuredValidator,
	idGenerator IDGenerator,
) *Service {
	return &Service{
		employeeRepository: employeeRepository,
		campaignRepository: campaignRepository,
		agent:              agent,
		validator:          validator,
		idGenerator:        idGenerator,
	}
}

func (service *Service) Generate(ctx context.Context, employeeID string) (domain.GeneratedCampaign, error) {
	employeeDetails, err := service.employeeRepository.GetEmployee(ctx, employeeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.GeneratedCampaign{}, ErrEmployeeNotFound
		}
		return domain.GeneratedCampaign{}, fmt.Errorf("get employee: %w", err)
	}
	if employeeDetails.Profile == nil {
		return domain.GeneratedCampaign{}, ErrProfileRequired
	}
	if service.agent == nil || service.validator == nil || service.idGenerator == nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("generate campaign: %w", ErrGenerationFailed)
	}

	input := ports.ScenarioInput{Employee: employeeFromDetails(employeeDetails), Profile: *employeeDetails.Profile}
	raw, err := service.agent.Generate(ctx, input, nil)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("generate scenario: %w", ErrGenerationFailed)
	}
	if err := service.validator.ValidateScenario(raw); err != nil {
		raw, err = service.agent.Generate(ctx, input, &ports.ValidationFeedback{Message: err.Error()})
		if err != nil {
			return domain.GeneratedCampaign{}, fmt.Errorf("repair scenario: %w", ErrGenerationFailed)
		}
		if err := service.validator.ValidateScenario(raw); err != nil {
			return domain.GeneratedCampaign{}, fmt.Errorf("validate repaired scenario: %w", ErrGenerationFailed)
		}
	}

	generated, err := decodeScenario(raw)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("decode scenario: %w", ErrGenerationFailed)
	}
	if err := validateScenarioPolicy(generated); err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("validate scenario policy: %w", ErrGenerationFailed)
	}
	id, err := service.idGenerator()
	if err != nil || id == "" {
		return domain.GeneratedCampaign{}, fmt.Errorf("generate campaign id: %w", ErrGenerationFailed)
	}
	campaign := domain.GeneratedCampaign{
		CampaignID: id, EmployeeID: employeeDetails.EmployeeID,
		TemplateID: generated.TemplateID, Difficulty: generated.Difficulty,
		Subject: generated.Subject, EmailHTML: generated.EmailHTML,
		LandingConfig: generated.LandingConfig, DecisionReason: generated.DecisionReason,
		SafetyChecks: generated.SafetyChecks, Status: domain.CampaignStatusPendingReview,
		ApprovedBy: nil, ApprovedAt: nil, RejectionReason: nil,
	}
	if err := service.campaignRepository.CreateCampaign(ctx, campaign); err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("create campaign: %w", err)
	}
	persisted, err := service.campaignRepository.GetCampaign(ctx, campaign.CampaignID)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("read persisted campaign: %w", err)
	}
	return persisted, nil
}

func (service *Service) Get(ctx context.Context, campaignID string) (domain.GeneratedCampaign, error) {
	campaign, err := service.campaignRepository.GetCampaign(ctx, campaignID)
	if errors.Is(err, store.ErrNotFound) {
		return domain.GeneratedCampaign{}, ErrCampaignNotFound
	}
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("get campaign: %w", err)
	}
	return campaign, nil
}

func (service *Service) Approve(ctx context.Context, campaignID, approvedBy string) (domain.GeneratedCampaign, error) {
	campaign, err := service.Get(ctx, campaignID)
	if err != nil {
		return domain.GeneratedCampaign{}, err
	}
	if campaign.Status != domain.CampaignStatusPendingReview || validateSafetyChecks(campaign.SafetyChecks) != nil {
		return domain.GeneratedCampaign{}, ErrInvalidCampaignStatus
	}
	if err := service.campaignRepository.ApproveCampaign(ctx, campaignID, strings.TrimSpace(approvedBy)); err != nil {
		return domain.GeneratedCampaign{}, mapCampaignTransitionError("approve", err)
	}
	return service.Get(ctx, campaignID)
}

func (service *Service) Reject(ctx context.Context, campaignID, reason string) (domain.GeneratedCampaign, error) {
	campaign, err := service.Get(ctx, campaignID)
	if err != nil {
		return domain.GeneratedCampaign{}, err
	}
	if campaign.Status != domain.CampaignStatusPendingReview {
		return domain.GeneratedCampaign{}, ErrInvalidCampaignStatus
	}
	if err := service.campaignRepository.RejectCampaign(ctx, campaignID, strings.TrimSpace(reason)); err != nil {
		return domain.GeneratedCampaign{}, mapCampaignTransitionError("reject", err)
	}
	return service.Get(ctx, campaignID)
}

func mapCampaignTransitionError(action string, err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return ErrCampaignNotFound
	case errors.Is(err, store.ErrConflict):
		return ErrInvalidCampaignStatus
	default:
		return fmt.Errorf("%s campaign: %w", action, err)
	}
}

func employeeFromDetails(details domain.EmployeeDetails) domain.Employee {
	return domain.Employee{EmployeeID: details.EmployeeID, DisplayName: details.DisplayName, Email: details.Email, Department: details.Department, Title: details.Title, Company: details.Company}
}

func validateScenarioPolicy(output scenarioOutput) error {
	if _, ok := allowedTemplateIDs[output.TemplateID]; !ok {
		return fmt.Errorf("template is not allowlisted")
	}
	if !output.Difficulty.Valid() {
		return fmt.Errorf("difficulty is not allowlisted")
	}
	if strings.TrimSpace(output.Subject) == "" || strings.TrimSpace(output.DecisionReason) == "" ||
		strings.TrimSpace(output.LandingConfig.Title) == "" || strings.TrimSpace(output.LandingConfig.Description) == "" ||
		strings.TrimSpace(output.LandingConfig.CTALabel) == "" {
		return fmt.Errorf("scenario contains an empty required field")
	}
	if output.LandingConfig.Brand != testBrand {
		return fmt.Errorf("landing brand is not the test brand")
	}
	if !strings.Contains(output.EmailHTML, landingPlaceholder) {
		return fmt.Errorf("email lacks landing URL placeholder")
	}
	return validateSafetyChecks(output.SafetyChecks)
}

func validateSafetyChecks(checks []domain.SafetyCheck) error {
	passed := make(map[string]bool, len(checks))
	for _, check := range checks {
		if _, required := requiredSafetyRules[check.Rule]; required {
			if !check.Passed {
				return fmt.Errorf("required safety check %q did not pass", check.Rule)
			}
			passed[check.Rule] = check.Passed
		}
	}
	for rule := range requiredSafetyRules {
		if !passed[rule] {
			return fmt.Errorf("required safety check %q did not pass", rule)
		}
	}
	return nil
}
