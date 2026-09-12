package roleb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

	config := LandingConfig{
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

func (r *Repository) RecordEvent(ctx context.Context, req EventRequest) error {
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

	var status, employeeID string
	if err := tx.QueryRowContext(ctx, `SELECT status, employee_id FROM campaigns WHERE id = ?`, campaignID).Scan(&status, &employeeID); err != nil {
		if err == sql.ErrNoRows {
			return SimulateResponse{}, ErrNotFound
		}
		return SimulateResponse{}, err
	}
	if status != "approved" {
		return SimulateResponse{}, ErrConflict
	}

	token, err := newToken()
	if err != nil {
		return SimulateResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE campaigns SET status = 'simulated' WHERE id = ?`, campaignID); err != nil {
		return SimulateResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO campaign_targets (campaign_id, employee_id, token)
VALUES (?, ?, ?)`, campaignID, employeeID, token); err != nil {
		return SimulateResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return SimulateResponse{}, err
	}

	return SimulateResponse{
		CampaignID: campaignID,
		Status:     "simulated",
		Targets: []SimulateTarget{{
			EmployeeID: employeeID,
			Token:      token,
			LandingURL: fmt.Sprintf("/landing/%s", token),
		}},
	}, nil
}

func newToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
