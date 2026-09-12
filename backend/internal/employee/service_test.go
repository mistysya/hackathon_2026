package employee

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

func TestImportCSVParsesAllRowsBeforeWriting(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{imported: []bool{true, false, true}}
	service := NewService(repository)

	response, err := service.ImportCSV(context.Background(), strings.NewReader(
		"employee_id,display_name,email,department,title,company\r\n"+
			" E001 , Demo User , demo@example.test , Sales , Manager , Acme \r\n"+
			",Missing ID,missing-id@example.test,,,\r\n"+
			"E002,,missing-name@example.test,,,\r\n"+
			"E003,Missing Email,,,,\r\n"+
			"E001,Duplicate,duplicate@example.test,,,\r\n"+
			"E004,李小明,ming@example.test,,,\r\n"))
	if err != nil {
		t.Fatalf("ImportCSV() error = %v", err)
	}
	if response.Imported != 2 || response.Skipped != 4 {
		t.Fatalf("ImportCSV() response = %#v", response)
	}
	wantErrors := []RowError{
		{Row: 3, Reason: "missing employee_id"},
		{Row: 4, Reason: "missing display_name"},
		{Row: 5, Reason: "missing email"},
		{Row: 6, Reason: "duplicate employee_id"},
	}
	if !sameRowErrors(response.Errors, wantErrors) {
		t.Fatalf("ImportCSV() errors = %#v, want %#v", response.Errors, wantErrors)
	}
	if len(repository.importEmployeesInput) != 3 {
		t.Fatalf("ImportEmployees got %d rows, want 3", len(repository.importEmployeesInput))
	}
	if employee := repository.importEmployeesInput[0]; employee != (domain.Employee{
		EmployeeID: "E001", DisplayName: "Demo User", Email: "demo@example.test", Department: "Sales", Title: "Manager", Company: "Acme",
	}) {
		t.Fatalf("first imported employee = %#v", employee)
	}
}

func TestImportCSVRejectsInvalidSyntaxBeforeWriting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		csv  string
	}{
		{name: "empty file", csv: ""},
		{name: "wrong header", csv: "employee_id,email,display_name,department,title,company\n"},
		{name: "wrong field count", csv: "employee_id,display_name,email,department,title,company\nE001,Demo,demo@example.test\n"},
		{name: "malformed quotes", csv: "employee_id,display_name,email,department,title,company\nE001,\"Demo,demo@example.test,,,\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repository := &fakeRepository{}
			_, err := NewService(repository).ImportCSV(context.Background(), strings.NewReader(test.csv))
			if !errors.Is(err, ErrInvalidCSV) {
				t.Fatalf("ImportCSV() error = %v, want ErrInvalidCSV", err)
			}
			if repository.importCalls != 0 {
				t.Fatalf("ImportEmployees calls = %d, want 0", repository.importCalls)
			}
		})
	}
}

func TestImportCSVRepositoryFailureHasNoResponse(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{importErr: errors.New("rollback")}
	response, err := NewService(repository).ImportCSV(context.Background(), strings.NewReader(
		"employee_id,display_name,email,department,title,company\nE001,Demo,demo@example.test,,,\n"))
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("ImportCSV() error = %v, want ErrRepository", err)
	}
	if response.Imported != 0 || response.Skipped != 0 || response.Errors != nil {
		t.Fatalf("ImportCSV() response = %#v, want zero response", response)
	}
}

func TestImportCSVSortsDuplicateAndValidationErrorsByRow(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{imported: []bool{false, true}}
	response, err := NewService(repository).ImportCSV(context.Background(), strings.NewReader(
		"employee_id,display_name,email,department,title,company\n"+
			"E001,First,first@example.test,,,\n"+
			",Missing ID,missing@example.test,,,\n"+
			"E002,Second,second@example.test,,,\n"))
	if err != nil {
		t.Fatalf("ImportCSV() error = %v", err)
	}
	want := []RowError{
		{Row: 2, Reason: "duplicate employee_id"},
		{Row: 3, Reason: "missing employee_id"},
	}
	if !sameRowErrors(response.Errors, want) {
		t.Fatalf("ImportCSV() errors = %#v, want %#v", response.Errors, want)
	}
}

func TestListAndGetEmployees(t *testing.T) {
	t.Parallel()
	repository := &fakeRepository{
		list: nil,
		details: domain.EmployeeDetails{
			EmployeeID: "E001", DisplayName: "Demo User", Email: "demo@example.test",
		},
	}
	service := NewService(repository)
	employees, err := service.ListEmployees(context.Background())
	if err != nil || employees == nil || len(employees) != 0 {
		t.Fatalf("ListEmployees() = %#v, %v; want non-nil empty slice, nil", employees, err)
	}
	details, err := service.GetEmployee(context.Background(), "E001")
	if err != nil || details.EmployeeID != "E001" {
		t.Fatalf("GetEmployee() = %#v, %v", details, err)
	}

	repository.getErr = store.ErrNotFound
	_, err = service.GetEmployee(context.Background(), "unknown")
	if !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("GetEmployee() error = %v, want ErrEmployeeNotFound", err)
	}

	repository.getErr = errors.New("database unavailable")
	_, err = service.GetEmployee(context.Background(), "E001")
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("GetEmployee() error = %v, want ErrRepository", err)
	}
}

func sameRowErrors(actual, expected []RowError) bool {
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

type fakeRepository struct {
	imported             []bool
	importErr            error
	importCalls          int
	importEmployeesInput []domain.Employee
	list                 []domain.EmployeeSummary
	listErr              error
	details              domain.EmployeeDetails
	getErr               error
}

func (repository *fakeRepository) ImportEmployees(_ context.Context, employees []domain.Employee) ([]bool, error) {
	repository.importCalls++
	repository.importEmployeesInput = append([]domain.Employee(nil), employees...)
	return repository.imported, repository.importErr
}

func (repository *fakeRepository) ListEmployees(context.Context) ([]domain.EmployeeSummary, error) {
	return repository.list, repository.listErr
}

func (repository *fakeRepository) GetEmployee(context.Context, string) (domain.EmployeeDetails, error) {
	return repository.details, repository.getErr
}

func (repository *fakeRepository) GetProfile(context.Context, string) (domain.EmployeeProfile, error) {
	return domain.EmployeeProfile{}, store.ErrNotFound
}

func (repository *fakeRepository) ReplaceProfile(context.Context, domain.EmployeeProfile) error {
	return nil
}
