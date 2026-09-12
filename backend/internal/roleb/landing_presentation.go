package roleb

import "strings"

type landingPresentation struct {
	Theme                string
	PortalLabel          string
	Audience             string
	FormHeading          string
	PrimaryLabel         string
	PrimaryPlaceholder   string
	SecondaryLabel       string
	SecondaryPlaceholder string
}

func presentationFor(templateID, department, title string) landingPresentation {
	audience := strings.Trim(strings.Join([]string{strings.TrimSpace(department), strings.TrimSpace(title)}, " · "), " ·")
	base := landingPresentation{
		Theme:                "learning",
		PortalLabel:          "Learning portal",
		Audience:             audience,
		FormHeading:          "Confirm this training request",
		PrimaryLabel:         "Demo work email",
		PrimaryPlaceholder:   "name@example.test",
		SecondaryLabel:       "Demo request reference",
		SecondaryPlaceholder: "TRAINING-0000",
	}

	switch templateID {
	case "saas_security_notice":
		base.Theme = "developer"
		base.PortalLabel = "Developer workspace"
		base.FormHeading = "Review project workspace access"
		base.SecondaryLabel = "Demo workspace reference"
		base.SecondaryPlaceholder = "DEV-ACCESS-0000"
	case "event_followup":
		base.Theme = "event"
		base.PortalLabel = "Event resources"
		base.FormHeading = "Request the follow-up materials"
		base.SecondaryLabel = "Demo registration reference"
		base.SecondaryPlaceholder = "EVENT-0000"
	case "benefit_update":
		base.Theme = "people"
		base.PortalLabel = "People operations"
		base.FormHeading = "Review the benefit update"
		base.SecondaryLabel = "Demo employee reference"
		base.SecondaryPlaceholder = "PEOPLE-0000"
	}
	return base
}
