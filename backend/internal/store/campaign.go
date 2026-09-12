package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

func (repository *Repository) CreateCampaign(ctx context.Context, campaign domain.GeneratedCampaign) error {
	if !campaign.Status.Valid() {
		return fmt.Errorf("invalid campaign status %q", campaign.Status)
	}
	if !campaign.Difficulty.Valid() {
		return fmt.Errorf("invalid campaign difficulty %q", campaign.Difficulty)
	}
	landingConfigJSON, err := json.Marshal(campaign.LandingConfig)
	if err != nil {
		return fmt.Errorf("encode landing config: %w", err)
	}
	safetyChecks := campaign.SafetyChecks
	if safetyChecks == nil {
		safetyChecks = []domain.SafetyCheck{}
	}
	safetyChecksJSON, err := json.Marshal(safetyChecks)
	if err != nil {
		return fmt.Errorf("encode safety checks: %w", err)
	}

	_, err = repository.db.ExecContext(ctx, `
		INSERT INTO campaigns (
			id, employee_id, template_id, status, difficulty, decision_reason, subject, email_html,
			landing_config_json, safety_checks_json, approved_by, approved_at, rejection_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		campaign.CampaignID,
		campaign.EmployeeID,
		campaign.TemplateID,
		campaign.Status,
		campaign.Difficulty,
		campaign.DecisionReason,
		campaign.Subject,
		campaign.EmailHTML,
		string(landingConfigJSON),
		string(safetyChecksJSON),
		campaign.ApprovedBy,
		campaign.ApprovedAt,
		campaign.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("create campaign %q: %w", campaign.CampaignID, mapError(err))
	}
	return nil
}

func (repository *Repository) GetCampaign(ctx context.Context, campaignID string) (domain.GeneratedCampaign, error) {
	var campaign domain.GeneratedCampaign
	var landingConfigJSON string
	var safetyChecksJSON string
	var difficulty sql.NullString
	var approvedBy sql.NullString
	var approvedAt sql.NullString
	var rejectionReason sql.NullString
	err := repository.db.QueryRowContext(ctx, `
		SELECT id, employee_id, template_id, status, difficulty, COALESCE(decision_reason, ''),
		       COALESCE(subject, ''), COALESCE(email_html, ''), landing_config_json, safety_checks_json,
		       approved_by, approved_at, rejection_reason
		FROM campaigns
		WHERE id = ?
	`, campaignID).Scan(
		&campaign.CampaignID,
		&campaign.EmployeeID,
		&campaign.TemplateID,
		&campaign.Status,
		&difficulty,
		&campaign.DecisionReason,
		&campaign.Subject,
		&campaign.EmailHTML,
		&landingConfigJSON,
		&safetyChecksJSON,
		&approvedBy,
		&approvedAt,
		&rejectionReason,
	)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("get campaign %q: %w", campaignID, mapError(err))
	}
	if difficulty.Valid {
		campaign.Difficulty = domain.Difficulty(difficulty.String)
	}
	if !campaign.Status.Valid() {
		return domain.GeneratedCampaign{}, fmt.Errorf("invalid stored campaign status %q", campaign.Status)
	}
	if !campaign.Difficulty.Valid() {
		return domain.GeneratedCampaign{}, fmt.Errorf("invalid stored campaign difficulty %q", campaign.Difficulty)
	}
	if err := json.Unmarshal([]byte(landingConfigJSON), &campaign.LandingConfig); err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("decode landing config: %w", err)
	}
	if err := json.Unmarshal([]byte(safetyChecksJSON), &campaign.SafetyChecks); err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("decode safety checks: %w", err)
	}
	if campaign.SafetyChecks == nil {
		campaign.SafetyChecks = []domain.SafetyCheck{}
	}
	campaign.ApprovedBy = nullStringPointer(approvedBy)
	campaign.ApprovedAt = nullStringPointer(approvedAt)
	campaign.RejectionReason = nullStringPointer(rejectionReason)
	return campaign, nil
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
