import type { CampaignReport, Employee, GeneratedCampaign } from "./types";

export const demoEmployee: Employee = {
  employeeId: "E001",
  displayName: "Demo User",
  email: "demo.user@example.test",
  department: "Engineering",
  title: "Software Engineer",
  company: "Demo Corp",
  hasProfile: true,
  profile: {
    employeeId: "E001",
    displayName: "Demo User",
    department: "Engineering",
    publicFacts: [
      { fact: "近期參加公開 AI 資安技術研討會", sourceUrl: "https://example.com/event/ai-security-workshop", confidence: 0.88, sourceType: "fixture" },
      { fact: "於公開技術部落格分享後端開發文章", sourceUrl: "https://example.com/blog/demo-user", confidence: 0.72, sourceType: "fixture" },
    ],
    riskSignals: ["可能較容易受到活動通知類演練情境影響", "可能會點擊與技術研討會相關的後續通知"],
    recommendedScenario: "event_followup",
  },
};

export const demoCampaign: GeneratedCampaign = {
  campaignId: "c_9f2c8a1b4d7e4f30a1b2c3d4e5f60718",
  employeeId: "E001",
  templateId: "event_followup",
  difficulty: "medium",
  subject: "【提醒】AI 資安研討會後續問卷與資料下載",
  emailHtml: '<html><body><p>Hi Demo User,</p><p>感謝參加本次 AI 資安研討會。請於下方連結填寫問卷並下載簡報。</p><p><a href="{{landingUrl}}">前往問卷與資料下載</a></p><p>Demo Corp 教育訓練小組（測試品牌）</p></body></html>',
  landingConfig: { title: "AI 資安研討會後續問卷", brand: "Demo Corp Training (測試品牌)", description: "請填寫以下欄位以取得研討會簡報。", ctaLabel: "送出並下載" },
  decisionReason: "員工公開資料顯示近期參加 AI 資安研討會，event_followup 情境與其職務及公開活動高度吻合。",
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

export const demoReport: CampaignReport = {
  campaignId: demoCampaign.campaignId,
  targetCount: 1,
  funnel: { simulated: 1, opened: 1, clicked: 1, formAttempted: 1, trainingViewed: 1 },
  events: [
    { eventType: "opened", occurredAt: "2026-09-12T14:03:11Z" },
    { eventType: "clicked", occurredAt: "2026-09-12T14:03:29Z" },
    { eventType: "form_attempted", occurredAt: "2026-09-12T14:03:52Z" },
    { eventType: "training_viewed", occurredAt: "2026-09-12T14:03:54Z" },
  ],
};
