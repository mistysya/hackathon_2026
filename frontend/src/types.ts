export type SourceType = "live" | "fixture" | "manual";
export type EventType = "opened" | "clicked" | "form_attempted" | "training_viewed";
export type CampaignStatus = "pending_review" | "approved" | "rejected" | "simulated";

export type PublicFact = {
  fact: string;
  sourceUrl: string | null;
  confidence: number | null;
  sourceType: SourceType;
};

export type EmployeeProfile = {
  employeeId: string;
  displayName: string;
  department: string;
  publicFacts: PublicFact[];
  riskSignals: string[];
  recommendedScenario: string;
};

export type EmployeeSummary = {
  employeeId: string;
  displayName: string;
  department: string;
  title: string;
  hasProfile: boolean;
};

export type Employee = EmployeeSummary & {
  email: string;
  company: string;
  profile: EmployeeProfile | null;
};

export type SafetyCheck = { rule: string; passed: boolean; detail?: string };

export type GeneratedCampaign = {
  campaignId: string;
  employeeId: string;
  templateId: string;
  difficulty: "low" | "medium" | "high";
  subject: string;
  emailHtml: string;
  landingConfig: { title: string; brand: string; description: string; ctaLabel: string };
  decisionReason: string;
  safetyChecks: SafetyCheck[];
  status: CampaignStatus;
  approvedBy: string | null;
  approvedAt: string | null;
  rejectionReason: string | null;
};

export type Target = { employeeId: string; token: string; landingUrl: string };
export type Simulation = { campaignId: string; status: "simulated"; targets: Target[] };
export type CampaignReport = {
  campaignId: string;
  targetCount: number;
  funnel: Record<"simulated" | EventType, number>;
  events: Array<{ eventType: EventType; occurredAt: string }>;
};

export type ImportResult = {
  imported: number;
  skipped: number;
  errors: Array<{ row: number; reason: string }>;
};
