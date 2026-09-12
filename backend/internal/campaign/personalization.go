package campaign

import (
	"strings"

	"github.com/mistysya/hackathon_2026/backend/internal/ports"
)

type scenarioPersonalization struct {
	PreferredTemplate string `json:"preferredTemplate"`
	RoleFocus         string `json:"roleFocus"`
	SubjectPrefix     string `json:"-"`
}

func personalizeScenario(input ports.ScenarioInput) scenarioPersonalization {
	combined := strings.ToLower(strings.Join([]string{input.Employee.Department, input.Employee.Title}, " "))
	preferred := input.Profile.RecommendedScenario
	focus := "Use a plausible internal workplace workflow appropriate to the supplied department and title."

	switch {
	case containsAny(combined, "quant", "software", "engineer", "developer", "data", "platform", "devops", "sre"):
		preferred = "saas_security_notice"
		if strings.Contains(combined, "quant") {
			focus = "Use a fictional quantitative-engineering workflow such as a research-code workspace, model repository, data pipeline, or deployment access review."
		} else {
			focus = "Use a fictional software-engineering workflow such as a source-code workspace, build pipeline, dependency alert, or developer access review."
		}
	case containsAny(combined, "people", "human resources", " hr", "hr ", "benefits"):
		preferred = "benefit_update"
		focus = "Use a fictional people-operations workflow such as an employee benefit or policy update."
	}

	if _, ok := senderCatalog[preferred]; !ok {
		preferred = defaultTemplateID
	}
	subjectPrefix := ""
	if preferred == "saas_security_notice" {
		if strings.Contains(combined, "quant") {
			subjectPrefix = "Quant workspace"
		} else {
			subjectPrefix = "Developer workspace"
		}
	} else if preferred == "benefit_update" {
		subjectPrefix = "People operations"
	}
	return scenarioPersonalization{PreferredTemplate: preferred, RoleFocus: focus, SubjectPrefix: subjectPrefix}
}

func roleAwareSubject(subject string, personalization scenarioPersonalization) string {
	subject = strings.TrimSpace(subject)
	if personalization.SubjectPrefix == "" || strings.Contains(strings.ToLower(subject), strings.ToLower(personalization.SubjectPrefix)) {
		return subject
	}
	return personalization.SubjectPrefix + ": " + subject
}

func withoutModelGreeting(body []string) []string {
	if len(body) == 0 {
		return body
	}
	first := strings.ToLower(strings.TrimSpace(body[0]))
	for _, prefix := range []string{"hi ", "hello ", "dear "} {
		if strings.HasPrefix(first, prefix) {
			return body[1:]
		}
	}
	return body
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
