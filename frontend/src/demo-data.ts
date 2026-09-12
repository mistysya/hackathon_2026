import type { Employee, GeneratedCampaign } from "./types";

export const demoEmployee: Employee = {
  employeeId: "E001",
  displayName: "Demo User",
  email: "demo.user@example.test",
  department: "Engineering",
  title: "Software Engineer",
  company: "Demo Corp",
  profile: {
    employeeId: "E001",
    displayName: "Demo User",
    department: "Engineering",
    publicFacts: [
      { fact: "Recently attended a public AI security workshop", sourceUrl: "https://example.com/event/ai-security-workshop", confidence: 0.88, sourceType: "fixture" },
      { fact: "Published a backend engineering article on a public technical blog", sourceUrl: "https://example.com/blog/demo-user", confidence: 0.72, sourceType: "fixture" },
    ],
    riskSignals: ["May be more responsive to an exercise themed around event notifications", "May click a follow-up notification related to a technical workshop"],
    recommendedScenario: "event_followup",
  },
};

export const demoCampaign: GeneratedCampaign = {
  campaignId: "c_9f2c8a1b4d7e4f30a1b2c3d4e5f60718",
  employeeId: "E001",
  templateId: "event_followup",
  difficulty: "medium",
  subject: "Reminder: AI Security Workshop Follow-up Survey and Materials",
  emailHtml: '<html><body><p>Hi Demo User,</p><p>Thank you for attending the AI security workshop. Complete the survey and download the materials using the link below.</p><p><a href="{{landingUrl}}">Open survey and materials</a></p><p>Demo Corp Training Team (test brand)</p></body></html>',
  landingConfig: { title: "AI Security Workshop Follow-up Survey", brand: "Demo Corp Training (test brand)", description: "Complete the form below to receive the workshop materials.", ctaLabel: "Submit and download" },
  decisionReason: "The employee's public profile shows recent AI security workshop participation, making the event_followup scenario relevant to their role and public activity.",
  safetyChecks: [
    { rule: "no_real_credentials_requested", passed: true },
    { rule: "no_sensitive_personal_data", passed: true },
    { rule: "no_unsourced_personal_facts", passed: true },
    { rule: "no_prohibited_impersonation", passed: true },
    { rule: "cta_points_to_controlled_domain", passed: true },
  ],
  status: "pending_review",
  approvedBy: null,
  approvedAt: null,
  rejectionReason: null,
};
