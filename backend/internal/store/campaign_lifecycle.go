package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

var lifecycleRequiredSafetyRules = map[string]struct{}{
	"no_real_credentials_requested":   {},
	"no_sensitive_personal_data":      {},
	"no_unsourced_personal_facts":     {},
	"no_prohibited_impersonation":     {},
	"cta_points_to_controlled_domain": {},
}

var _ ports.CampaignLifecycleRepository = (*Repository)(nil)

// ApproveCampaign atomically verifies and approves a campaign that is pending
// review. The conditional update is the concurrency boundary: only one state
// transition can win.
func (repository *Repository) ApproveCampaign(ctx context.Context, campaignID, approvedBy string) (domain.GeneratedCampaign, error) {
	return repository.transitionCampaign(ctx, campaignID, func(tx *sql.Tx) (domain.GeneratedCampaign, error) {
		campaign, err := getCampaignInTransaction(ctx, tx, campaignID)
		if err != nil {
			return domain.GeneratedCampaign{}, err
		}
		if campaign.Status != domain.CampaignStatusPendingReview {
			return domain.GeneratedCampaign{}, ErrConflict
		}
		if err := validatePersistedSafetyChecks(campaign.SafetyChecks); err != nil {
			return domain.GeneratedCampaign{}, fmt.Errorf("approve campaign %q: %w", campaignID, ErrConflict)
		}

		result, err := tx.ExecContext(ctx, `
			UPDATE campaigns
			SET status = 'approved', approved_by = ?,
				approved_at = strftime('%Y-%m-%dT%H:%M:%SZ','now'), rejection_reason = NULL
			WHERE id = ? AND status = 'pending_review'
		`, approvedBy, campaignID)
		if err != nil {
			return domain.GeneratedCampaign{}, fmt.Errorf("approve campaign %q: %w", campaignID, mapError(err))
		}
		if err := requireTransitionRow(ctx, tx, campaignID, result); err != nil {
			return domain.GeneratedCampaign{}, err
		}
		return getCampaignInTransaction(ctx, tx, campaignID)
	})
}

// RejectCampaign atomically rejects a campaign that is pending review.
func (repository *Repository) RejectCampaign(ctx context.Context, campaignID, reason string) (domain.GeneratedCampaign, error) {
	return repository.transitionCampaign(ctx, campaignID, func(tx *sql.Tx) (domain.GeneratedCampaign, error) {
		campaign, err := getCampaignInTransaction(ctx, tx, campaignID)
		if err != nil {
			return domain.GeneratedCampaign{}, err
		}
		if campaign.Status != domain.CampaignStatusPendingReview {
			return domain.GeneratedCampaign{}, ErrConflict
		}

		result, err := tx.ExecContext(ctx, `
			UPDATE campaigns
			SET status = 'rejected', approved_by = NULL, approved_at = NULL, rejection_reason = ?
			WHERE id = ? AND status = 'pending_review'
		`, reason, campaignID)
		if err != nil {
			return domain.GeneratedCampaign{}, fmt.Errorf("reject campaign %q: %w", campaignID, mapError(err))
		}
		if err := requireTransitionRow(ctx, tx, campaignID, result); err != nil {
			return domain.GeneratedCampaign{}, err
		}
		return getCampaignInTransaction(ctx, tx, campaignID)
	})
}

func (repository *Repository) transitionCampaign(ctx context.Context, campaignID string, transition func(*sql.Tx) (domain.GeneratedCampaign, error)) (campaign domain.GeneratedCampaign, err error) {
	tx, err := repository.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("begin campaign transition: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	campaign, err = transition(tx)
	if err != nil {
		return domain.GeneratedCampaign{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("commit campaign transition: %w", err)
	}
	return campaign, nil
}

func requireTransitionRow(ctx context.Context, tx *sql.Tx, campaignID string, result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read campaign transition result: %w", err)
	}
	if affected != 0 {
		return nil
	}
	_, err = getCampaignInTransaction(ctx, tx, campaignID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return ErrConflict
}

func getCampaignInTransaction(ctx context.Context, tx *sql.Tx, campaignID string) (domain.GeneratedCampaign, error) {
	var campaign domain.GeneratedCampaign
	var landingConfigJSON, safetyChecksJSON string
	var difficulty, approvedBy, approvedAt, rejectionReason sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, employee_id, template_id, status, difficulty, COALESCE(decision_reason, ''),
		       COALESCE(subject, ''), COALESCE(email_html, ''), landing_config_json, safety_checks_json,
		       approved_by, approved_at, rejection_reason
		FROM campaigns WHERE id = ?
	`, campaignID).Scan(
		&campaign.CampaignID, &campaign.EmployeeID, &campaign.TemplateID, &campaign.Status, &difficulty,
		&campaign.DecisionReason, &campaign.Subject, &campaign.EmailHTML, &landingConfigJSON, &safetyChecksJSON,
		&approvedBy, &approvedAt, &rejectionReason,
	)
	if err != nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("get campaign %q: %w", campaignID, mapError(err))
	}
	if difficulty.Valid {
		campaign.Difficulty = domain.Difficulty(difficulty.String)
	}
	if !campaign.Status.Valid() || !campaign.Difficulty.Valid() {
		return domain.GeneratedCampaign{}, fmt.Errorf("campaign %q has invalid stored enum", campaignID)
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

func validatePersistedSafetyChecks(checks []domain.SafetyCheck) error {
	passed := make(map[string]bool, len(lifecycleRequiredSafetyRules))
	for _, check := range checks {
		if _, required := lifecycleRequiredSafetyRules[check.Rule]; !required {
			continue
		}
		if !check.Passed {
			return fmt.Errorf("required safety check %q did not pass", check.Rule)
		}
		passed[check.Rule] = true
	}
	for rule := range lifecycleRequiredSafetyRules {
		if !passed[rule] {
			return fmt.Errorf("required safety check %q is missing", rule)
		}
	}
	return nil
}
