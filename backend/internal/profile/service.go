package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

type Service struct {
	repository ports.EmployeeRepository
	adapter    ports.EnrichmentAdapter
	agent      ports.ProfileAgent
	validator  ports.StructuredValidator
}

func NewService(repository ports.EmployeeRepository, adapter ports.EnrichmentAdapter, agent ports.ProfileAgent, validator ports.StructuredValidator) *Service {
	return &Service{repository: repository, adapter: adapter, agent: agent, validator: validator}
}

// Enrich creates a profile from fixture evidence, validates it once (with a
// single corrective retry), then returns the canonical persisted profile.
func (service *Service) Enrich(ctx context.Context, employeeID string) (domain.EmployeeProfile, error) {
	if service == nil || service.repository == nil || service.adapter == nil || service.agent == nil || service.validator == nil {
		return domain.EmployeeProfile{}, ErrEnrichmentFailed
	}

	employee, err := service.repository.GetEmployee(ctx, employeeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.EmployeeProfile{}, ErrEmployeeNotFound
		}
		return domain.EmployeeProfile{}, fmt.Errorf("get employee: %w", err)
	}
	inputEmployee := domain.Employee{
		EmployeeID:  employee.EmployeeID,
		DisplayName: employee.DisplayName,
		Email:       employee.Email,
		Department:  employee.Department,
		Title:       employee.Title,
		Company:     employee.Company,
	}
	evidence, err := service.adapter.Search(ctx, inputEmployee)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("%w: search evidence", ErrEnrichmentFailed)
	}

	input := ports.ProfileInput{Employee: inputEmployee, Evidence: evidence}
	rawProfile, err := service.agent.Generate(ctx, input, nil)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("%w: generate profile", ErrEnrichmentFailed)
	}
	if err := service.validator.ValidateProfile(rawProfile); err != nil {
		rawProfile, err = service.agent.Generate(ctx, input, &ports.ValidationFeedback{Message: err.Error()})
		if err != nil {
			return domain.EmployeeProfile{}, fmt.Errorf("%w: regenerate profile", ErrEnrichmentFailed)
		}
		if err := service.validator.ValidateProfile(rawProfile); err != nil {
			return domain.EmployeeProfile{}, fmt.Errorf("%w: profile validation", ErrEnrichmentFailed)
		}
	}

	profile, err := decodeProfile(rawProfile)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("%w: decode profile", ErrEnrichmentFailed)
	}
	if profile.EmployeeID != employee.EmployeeID || profile.DisplayName != employee.DisplayName || profile.Department != employee.Department {
		return domain.EmployeeProfile{}, fmt.Errorf("%w: profile identity does not match employee", ErrEnrichmentFailed)
	}
	if err := service.repository.ReplaceProfile(ctx, profile); err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("replace profile: %w", err)
	}
	persisted, err := service.repository.GetProfile(ctx, employeeID)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("get persisted profile: %w", err)
	}
	return persisted, nil
}

func decodeProfile(raw []byte) (domain.EmployeeProfile, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return domain.EmployeeProfile{}, fmt.Errorf("profile must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var profile domain.EmployeeProfile
	if err := decoder.Decode(&profile); err != nil {
		return domain.EmployeeProfile{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return domain.EmployeeProfile{}, fmt.Errorf("trailing JSON value")
		}
		return domain.EmployeeProfile{}, err
	}
	return profile, nil
}
