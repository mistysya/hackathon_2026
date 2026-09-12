package campaign

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/domain"
	"github.com/mistysya/hackathon_2026/backend/internal/ports"
	"github.com/mistysya/hackathon_2026/backend/internal/store"
)

var (
	ErrLifecycleInvalidInput = errors.New("campaign lifecycle: invalid input")
	ErrCampaignConflict      = errors.New("campaign lifecycle: campaign cannot transition")
)

type LifecycleService struct {
	repository ports.CampaignLifecycleRepository
}

func NewLifecycleService(repository ports.CampaignLifecycleRepository) *LifecycleService {
	return &LifecycleService{repository: repository}
}

func (service *LifecycleService) Approve(ctx context.Context, campaignID, approvedBy string) (domain.GeneratedCampaign, error) {
	campaignID, approvedBy = strings.TrimSpace(campaignID), strings.TrimSpace(approvedBy)
	if campaignID == "" || approvedBy == "" {
		return domain.GeneratedCampaign{}, ErrLifecycleInvalidInput
	}
	if service == nil || service.repository == nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("approve campaign: repository unavailable")
	}
	campaign, err := service.repository.ApproveCampaign(ctx, campaignID, approvedBy)
	return mapLifecycleError(campaign, err)
}

func (service *LifecycleService) Reject(ctx context.Context, campaignID, reason string) (domain.GeneratedCampaign, error) {
	campaignID, reason = strings.TrimSpace(campaignID), strings.TrimSpace(reason)
	if campaignID == "" || reason == "" {
		return domain.GeneratedCampaign{}, ErrLifecycleInvalidInput
	}
	if service == nil || service.repository == nil {
		return domain.GeneratedCampaign{}, fmt.Errorf("reject campaign: repository unavailable")
	}
	campaign, err := service.repository.RejectCampaign(ctx, campaignID, reason)
	return mapLifecycleError(campaign, err)
}

func mapLifecycleError(campaign domain.GeneratedCampaign, err error) (domain.GeneratedCampaign, error) {
	if errors.Is(err, store.ErrNotFound) {
		return domain.GeneratedCampaign{}, ErrCampaignNotFound
	}
	if errors.Is(err, store.ErrConflict) {
		return domain.GeneratedCampaign{}, ErrCampaignConflict
	}
	if err != nil {
		return domain.GeneratedCampaign{}, err
	}
	return campaign, nil
}
