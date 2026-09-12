package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

func (repository *Repository) ImportEmployees(ctx context.Context, employees []domain.Employee) ([]bool, error) {
	created := make([]bool, len(employees))
	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin employee import: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	statement, err := transaction.PrepareContext(ctx, `
		INSERT OR IGNORE INTO employees (employee_id, display_name, email, department, title, company)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare employee import: %w", err)
	}
	defer statement.Close()

	for index, employee := range employees {
		result, err := statement.ExecContext(ctx,
			employee.EmployeeID,
			employee.DisplayName,
			employee.Email,
			employee.Department,
			employee.Title,
			employee.Company,
		)
		if err != nil {
			return nil, fmt.Errorf("insert employee %q: %w", employee.EmployeeID, mapError(err))
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("employee rows affected: %w", err)
		}
		created[index] = rows == 1
	}

	if err := transaction.Commit(); err != nil {
		return nil, fmt.Errorf("commit employee import: %w", mapError(err))
	}
	return created, nil
}

func (repository *Repository) ListEmployees(ctx context.Context) ([]domain.EmployeeSummary, error) {
	rows, err := repository.db.QueryContext(ctx, `
		SELECT e.employee_id, e.display_name, COALESCE(e.department, ''), COALESCE(e.title, ''),
		       EXISTS(SELECT 1 FROM employee_profiles p WHERE p.employee_id = e.employee_id)
		FROM employees e
		ORDER BY e.employee_id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	employees := []domain.EmployeeSummary{}
	for rows.Next() {
		var employee domain.EmployeeSummary
		var hasProfile int
		if err := rows.Scan(&employee.EmployeeID, &employee.DisplayName, &employee.Department, &employee.Title, &hasProfile); err != nil {
			return nil, fmt.Errorf("scan employee summary: %w", err)
		}
		employee.HasProfile = hasProfile != 0
		employees = append(employees, employee)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate employees: %w", err)
	}
	return employees, nil
}

func (repository *Repository) GetEmployee(ctx context.Context, employeeID string) (domain.EmployeeDetails, error) {
	var employee domain.EmployeeDetails
	err := repository.db.QueryRowContext(ctx, `
		SELECT employee_id, display_name, email, COALESCE(department, ''), COALESCE(title, ''), COALESCE(company, '')
		FROM employees
		WHERE employee_id = ?
	`, employeeID).Scan(
		&employee.EmployeeID,
		&employee.DisplayName,
		&employee.Email,
		&employee.Department,
		&employee.Title,
		&employee.Company,
	)
	if err != nil {
		return domain.EmployeeDetails{}, fmt.Errorf("get employee %q: %w", employeeID, mapError(err))
	}

	profile, err := repository.GetProfile(ctx, employeeID)
	if err == nil {
		employee.Profile = &profile
	} else if !errors.Is(err, ErrNotFound) {
		return domain.EmployeeDetails{}, err
	}
	return employee, nil
}

func (repository *Repository) GetProfile(ctx context.Context, employeeID string) (domain.EmployeeProfile, error) {
	var profile domain.EmployeeProfile
	var riskSignalsJSON string
	err := repository.db.QueryRowContext(ctx, `
		SELECT p.employee_id, e.display_name, COALESCE(e.department, ''), p.risk_signals_json,
		       COALESCE(p.recommended_scenario, '')
		FROM employee_profiles p
		JOIN employees e ON e.employee_id = p.employee_id
		WHERE p.employee_id = ?
	`, employeeID).Scan(
		&profile.EmployeeID,
		&profile.DisplayName,
		&profile.Department,
		&riskSignalsJSON,
		&profile.RecommendedScenario,
	)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("get profile %q: %w", employeeID, mapError(err))
	}
	if err := json.Unmarshal([]byte(riskSignalsJSON), &profile.RiskSignals); err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("decode profile risk signals: %w", err)
	}
	if profile.RiskSignals == nil {
		profile.RiskSignals = []string{}
	}

	rows, err := repository.db.QueryContext(ctx, `
		SELECT fact, source_url, confidence, source_type
		FROM public_facts
		WHERE employee_id = ?
		ORDER BY id ASC
	`, employeeID)
	if err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("list public facts: %w", err)
	}
	defer rows.Close()

	profile.PublicFacts = []domain.PublicFact{}
	for rows.Next() {
		var fact domain.PublicFact
		var sourceURL sql.NullString
		var confidence sql.NullFloat64
		if err := rows.Scan(&fact.Fact, &sourceURL, &confidence, &fact.SourceType); err != nil {
			return domain.EmployeeProfile{}, fmt.Errorf("scan public fact: %w", err)
		}
		if sourceURL.Valid {
			fact.SourceURL = &sourceURL.String
		}
		if confidence.Valid {
			fact.Confidence = &confidence.Float64
		}
		if !fact.SourceType.Valid() {
			return domain.EmployeeProfile{}, fmt.Errorf("invalid source type %q", fact.SourceType)
		}
		profile.PublicFacts = append(profile.PublicFacts, fact)
	}
	if err := rows.Err(); err != nil {
		return domain.EmployeeProfile{}, fmt.Errorf("iterate public facts: %w", err)
	}
	return profile, nil
}

func (repository *Repository) ReplaceProfile(ctx context.Context, profile domain.EmployeeProfile) error {
	for _, fact := range profile.PublicFacts {
		if !fact.SourceType.Valid() {
			return fmt.Errorf("invalid source type %q", fact.SourceType)
		}
		if fact.Confidence != nil && (*fact.Confidence < 0 || *fact.Confidence > 1) {
			return fmt.Errorf("confidence must be between 0 and 1")
		}
	}

	riskSignalsJSON, err := json.Marshal(nonNilStrings(profile.RiskSignals))
	if err != nil {
		return fmt.Errorf("encode profile risk signals: %w", err)
	}

	transaction, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile replacement: %w", err)
	}
	defer func() { _ = transaction.Rollback() }()

	var exists int
	if err := transaction.QueryRowContext(ctx, `SELECT 1 FROM employees WHERE employee_id = ?`, profile.EmployeeID).Scan(&exists); err != nil {
		return fmt.Errorf("find profile employee %q: %w", profile.EmployeeID, mapError(err))
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM public_facts WHERE employee_id = ?`, profile.EmployeeID); err != nil {
		return fmt.Errorf("delete existing public facts: %w", err)
	}
	for _, fact := range profile.PublicFacts {
		if _, err := transaction.ExecContext(ctx, `
			INSERT INTO public_facts (employee_id, fact, source_url, confidence, source_type)
			VALUES (?, ?, ?, ?, ?)
		`, profile.EmployeeID, fact.Fact, fact.SourceURL, fact.Confidence, fact.SourceType); err != nil {
			return fmt.Errorf("insert public fact: %w", mapError(err))
		}
	}
	if _, err := transaction.ExecContext(ctx, `
		INSERT INTO employee_profiles (employee_id, risk_signals_json, recommended_scenario)
		VALUES (?, ?, ?)
		ON CONFLICT(employee_id) DO UPDATE SET
			risk_signals_json = excluded.risk_signals_json,
			recommended_scenario = excluded.recommended_scenario,
			generated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
	`, profile.EmployeeID, string(riskSignalsJSON), profile.RecommendedScenario); err != nil {
		return fmt.Errorf("upsert employee profile: %w", mapError(err))
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit profile replacement: %w", mapError(err))
	}
	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
