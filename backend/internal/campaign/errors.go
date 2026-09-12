// Package campaign generates reviewable training campaigns from enriched employee profiles.
package campaign

import "errors"

var (
	ErrEmployeeNotFound      = errors.New("campaign: employee not found")
	ErrProfileRequired       = errors.New("campaign: employee profile required")
	ErrGenerationFailed      = errors.New("campaign: generation failed")
	ErrCampaignNotFound      = errors.New("campaign: campaign not found")
	ErrInvalidCampaignStatus = errors.New("campaign: invalid campaign status")
)
