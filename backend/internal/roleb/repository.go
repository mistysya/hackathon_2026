package roleb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) LandingPage(ctx context.Context, token string) (LandingPageData, error) {
	const query = `
SELECT t.campaign_id, c.landing_config_json
FROM campaign_targets t
JOIN campaigns c ON c.id = t.campaign_id
WHERE t.token = ? AND c.status = 'simulated'`
	var campaignID string
	var rawConfig string
	if err := r.db.QueryRowContext(ctx, query, token).Scan(&campaignID, &rawConfig); err != nil {
		if err == sql.ErrNoRows {
			return LandingPageData{}, ErrNotFound
		}
		return LandingPageData{}, err
	}

	config := domain.LandingConfig{
		Title:       "安全演練頁面",
		Brand:       "Demo Corp Security Sandbox",
		Description: "這是受控演練頁面，不會傳送或保存表單欄位值。",
		CTALabel:    "送出測試表單",
	}
	if rawConfig != "" {
		_ = json.Unmarshal([]byte(rawConfig), &config)
	}
	return LandingPageData{
		Token:          token,
		Title:          config.Title,
		Brand:          config.Brand,
		Description:    config.Description,
		CTALabel:       config.CTALabel,
		EventEndpoint:  "/events",
		CampaignID:     campaignID,
		EducationTitle: "這是一場受控資安演練",
	}, nil
}

func (r *Repository) RecordEvent(ctx context.Context, req domain.EventRequest) error {
	if req.Token == "" {
		return ErrInvalidToken
	}
	if !req.EventType.Valid() {
		return ErrInvalidEvent
	}

	var campaignID, employeeID string
	if err := r.db.QueryRowContext(ctx, `SELECT campaign_id, employee_id FROM campaign_targets WHERE token = ?`, req.Token).Scan(&campaignID, &employeeID); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	_, err := r.db.ExecContext(ctx, `
INSERT OR IGNORE INTO tracking_events (campaign_id, employee_id, event_type, metadata_json)
VALUES (?, ?, ?, '{}')`, campaignID, employeeID, string(req.EventType))
	return err
}

func (r *Repository) SimulateCampaign(ctx context.Context, campaignID string) (SimulateResponse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return SimulateResponse{}, err
	}
	defer tx.Rollback()

	// Acquire the write lock before reading. A deferred read-then-write
	// transaction can fail with SQLITE_BUSY when two simulations race.
	result, err := tx.ExecContext(ctx, `UPDATE campaigns SET status = 'simulated' WHERE id = ? AND status = 'approved'`, campaignID)
	if err != nil {
		return SimulateResponse{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return SimulateResponse{}, err
	}
	var employeeID, subject, emailHTML, landingConfigJSON string
	if err := tx.QueryRowContext(ctx, `
		SELECT employee_id, COALESCE(subject, ''), COALESCE(email_html, ''), landing_config_json
		FROM campaigns WHERE id = ?
	`, campaignID).Scan(&employeeID, &subject, &emailHTML, &landingConfigJSON); err != nil {
		if err == sql.ErrNoRows {
			return SimulateResponse{}, ErrNotFound
		}
		return SimulateResponse{}, err
	}
	if changed != 1 {
		return SimulateResponse{}, ErrConflict
	}

	token, err := newToken()
	if err != nil {
		return SimulateResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO campaign_targets (campaign_id, employee_id, token)
VALUES (?, ?, ?)`, campaignID, employeeID, token); err != nil {
		return SimulateResponse{}, err
	}
	if err := r.persistMailboxDelivery(ctx, tx, campaignID, employeeID, subject, emailHTML, landingConfigJSON); err != nil {
		return SimulateResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return SimulateResponse{}, err
	}

	return SimulateResponse{
		CampaignID: campaignID,
		Status:     domain.CampaignStatusSimulated,
		Targets: []SimulateTarget{{
			EmployeeID: employeeID,
			Token:      token,
			LandingURL: fmt.Sprintf("/landing/%s", token),
		}},
	}, nil
}

// persistMailboxDelivery makes simulation the delivery boundary. INSERT OR
// IGNORE backfills campaigns made before mailbox persistence was introduced;
// current campaigns already have a row from store.CreateCampaign.
func (r *Repository) persistMailboxDelivery(ctx context.Context, tx *sql.Tx, campaignID, employeeID, subject, emailHTML, landingConfigJSON string) error {
	senderName := "Security Awareness Demo"
	var config domain.LandingConfig
	if json.Unmarshal([]byte(landingConfigJSON), &config) == nil && config.Brand != "" {
		senderName = config.Brand
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO mailbox_messages (
			campaign_id, employee_id, sender_name, sender_address, subject, email_html
		) VALUES (?, ?, ?, 'notification@campaign.example.test', ?, ?)
	`, campaignID, employeeID, senderName, subject, emailHTML); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE mailbox_messages
		SET delivered_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		WHERE campaign_id = ? AND delivered_at IS NULL
	`, campaignID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("mark mailbox delivery for campaign %q", campaignID)
	}
	return nil
}

func (r *Repository) CampaignReport(ctx context.Context, campaignID string) (domain.CampaignReport, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.CampaignReport{}, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM campaigns WHERE id = ?`, campaignID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return domain.CampaignReport{}, ErrNotFound
		}
		return domain.CampaignReport{}, err
	}

	report := domain.CampaignReport{CampaignID: campaignID, Events: []domain.CampaignEvent{}}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaign_targets WHERE campaign_id = ?`, campaignID).Scan(&report.TargetCount); err != nil {
		return domain.CampaignReport{}, err
	}
	report.Funnel.Simulated = report.TargetCount

	if err := tx.QueryRowContext(ctx, `
SELECT
	COUNT(DISTINCT CASE WHEN event_type = 'opened' THEN employee_id END),
	COUNT(DISTINCT CASE WHEN event_type = 'clicked' THEN employee_id END),
	COUNT(DISTINCT CASE WHEN event_type = 'form_attempted' THEN employee_id END),
	COUNT(DISTINCT CASE WHEN event_type = 'training_viewed' THEN employee_id END)
FROM tracking_events
WHERE campaign_id = ?`, campaignID).Scan(
		&report.Funnel.Opened,
		&report.Funnel.Clicked,
		&report.Funnel.FormAttempted,
		&report.Funnel.TrainingViewed,
	); err != nil {
		return domain.CampaignReport{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT event_type, occurred_at
FROM tracking_events
WHERE campaign_id = ?
ORDER BY occurred_at ASC, id ASC`, campaignID)
	if err != nil {
		return domain.CampaignReport{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var event domain.CampaignEvent
		if err := rows.Scan(&event.EventType, &event.OccurredAt); err != nil {
			return domain.CampaignReport{}, err
		}
		if !event.EventType.Valid() {
			return domain.CampaignReport{}, fmt.Errorf("invalid stored event type %q", event.EventType)
		}
		if err := validateRFC3339UTCTimestamp(event.OccurredAt); err != nil {
			return domain.CampaignReport{}, err
		}
		report.Events = append(report.Events, event)
	}
	if err := rows.Err(); err != nil {
		return domain.CampaignReport{}, err
	}
	if err := rows.Close(); err != nil {
		return domain.CampaignReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.CampaignReport{}, err
	}
	return report, nil
}

func validateRFC3339UTCTimestamp(value string) error {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("invalid stored event timestamp %q: %w", value, err)
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return fmt.Errorf("stored event timestamp is not UTC %q", value)
	}
	return nil
}

func newToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
