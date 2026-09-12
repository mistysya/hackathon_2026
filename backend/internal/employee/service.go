// Package employee provides employee CSV import and query use cases.
package employee

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

var (
	// ErrInvalidCSV means the request cannot be parsed as a complete employee CSV.
	ErrInvalidCSV = errors.New("invalid employee csv")
	// ErrEmployeeNotFound means no employee exists for the requested business ID.
	ErrEmployeeNotFound = errors.New("employee not found")
	// ErrRepository means a repository operation failed unexpectedly.
	ErrRepository = errors.New("employee repository failure")
)

var employeeCSVHeader = []string{
	"employee_id", "display_name", "email", "department", "title", "company",
}

// RowError describes one accepted CSV record that was not imported.
type RowError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// ImportResponse summarizes the result of a complete CSV import request.
type ImportResponse struct {
	Imported int        `json:"imported"`
	Skipped  int        `json:"skipped"`
	Errors   []RowError `json:"errors"`
}

// Service contains employee use cases. It depends only on the repository port.
type Service struct {
	repository ports.EmployeeRepository
}

// NewService creates an employee service using the supplied repository port.
func NewService(repository ports.EmployeeRepository) *Service {
	return &Service{repository: repository}
}

// ImportCSV first parses all input, then imports all valid rows in one repository
// call. CSV syntax errors never result in a partial repository write.
func (service *Service) ImportCSV(ctx context.Context, input io.Reader) (ImportResponse, error) {
	reader := csv.NewReader(input)

	header, err := reader.Read()
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	if err != nil || !sameFields(header, employeeCSVHeader) {
		return ImportResponse{}, invalidCSVError(err)
	}

	employees := make([]domain.Employee, 0)
	rowNumbers := make([]int, 0)
	errorsByRow := make([]RowError, 0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ImportResponse{}, invalidCSVError(err)
		}

		row, _ := reader.FieldPos(0)
		employee, reason := employeeFromRecord(record)
		if reason != "" {
			errorsByRow = append(errorsByRow, RowError{Row: row, Reason: reason})
			continue
		}
		employees = append(employees, employee)
		rowNumbers = append(rowNumbers, row)
	}

	response := ImportResponse{Errors: errorsByRow}
	if len(employees) == 0 {
		response.Skipped = len(response.Errors)
		return response, nil
	}

	created, err := service.repository.ImportEmployees(ctx, employees)
	if err != nil {
		return ImportResponse{}, fmt.Errorf("import employees: %w", ErrRepository)
	}
	if len(created) != len(employees) {
		return ImportResponse{}, fmt.Errorf("import employees: %w", ErrRepository)
	}
	for index, didCreate := range created {
		if didCreate {
			response.Imported++
			continue
		}
		response.Errors = append(response.Errors, RowError{
			Row:    rowNumbers[index],
			Reason: "duplicate employee_id",
		})
	}
	sort.Slice(response.Errors, func(left, right int) bool {
		return response.Errors[left].Row < response.Errors[right].Row
	})
	response.Skipped = len(response.Errors)
	return response, nil
}

// ListEmployees returns employee summaries. It always returns a non-nil slice.
func (service *Service) ListEmployees(ctx context.Context) ([]domain.EmployeeSummary, error) {
	employees, err := service.repository.ListEmployees(ctx)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", ErrRepository)
	}
	if employees == nil {
		employees = []domain.EmployeeSummary{}
	}
	return employees, nil
}

// GetEmployee returns an employee's details by business ID.
func (service *Service) GetEmployee(ctx context.Context, employeeID string) (domain.EmployeeDetails, error) {
	employee, err := service.repository.GetEmployee(ctx, employeeID)
	if err == nil {
		return employee, nil
	}
	if errors.Is(err, store.ErrNotFound) {
		return domain.EmployeeDetails{}, fmt.Errorf("get employee: %w", ErrEmployeeNotFound)
	}
	return domain.EmployeeDetails{}, fmt.Errorf("get employee: %w", ErrRepository)
}

func employeeFromRecord(record []string) (domain.Employee, string) {
	employee := domain.Employee{
		EmployeeID:  strings.TrimSpace(record[0]),
		DisplayName: strings.TrimSpace(record[1]),
		Email:       strings.TrimSpace(record[2]),
		Department:  strings.TrimSpace(record[3]),
		Title:       strings.TrimSpace(record[4]),
		Company:     strings.TrimSpace(record[5]),
	}
	switch {
	case employee.EmployeeID == "":
		return domain.Employee{}, "missing employee_id"
	case employee.DisplayName == "":
		return domain.Employee{}, "missing display_name"
	case employee.Email == "":
		return domain.Employee{}, "missing email"
	default:
		return employee, ""
	}
}

func sameFields(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range expected {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func invalidCSVError(cause error) error {
	if cause == nil {
		return ErrInvalidCSV
	}
	return fmt.Errorf("%w: %w", ErrInvalidCSV, cause)
}
