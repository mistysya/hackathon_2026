package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

// MailboxSenderAddress is a fixed demo-only sender. Generated content never
// controls an address, so no real sender can be persisted or impersonated.
const MailboxSenderAddress = "notification@campaign.example.test"

func insertMailboxMessage(ctx context.Context, tx *sql.Tx, campaign domain.GeneratedCampaign) error {
	senderName := strings.TrimSpace(campaign.LandingConfig.Brand)
	if senderName == "" {
		senderName = "Security Awareness Demo"
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO mailbox_messages (
			campaign_id, employee_id, sender_name, sender_address, subject, email_html
		) VALUES (?, ?, ?, ?, ?, ?)
	`, campaign.CampaignID, campaign.EmployeeID, senderName, MailboxSenderAddress, campaign.Subject, campaign.EmailHTML)
	return err
}
