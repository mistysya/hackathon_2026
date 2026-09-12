package ports

import (
	"context"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
)

// CampaignLifecycleRepository owns the atomic state transitions which move a
// generated campaign out of pending review.
type CampaignLifecycleRepository interface {
	ApproveCampaign(context.Context, string, string) (domain.GeneratedCampaign, error)
	RejectCampaign(context.Context, string, string) (domain.GeneratedCampaign, error)
}
