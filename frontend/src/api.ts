import { demoCampaign, demoEmployee } from "./demo-data";
import type { CampaignReport, Employee, EmployeeProfile, EmployeeSummary, EventType, GeneratedCampaign, ImportResult, Simulation } from "./types";

const useFixtures = import.meta.env.VITE_USE_FIXTURES !== "false";
const apiBase = "/api";

export class ApiError extends Error {
  constructor(message: string, readonly status = 0) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, init);
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const payload = await response.json() as { error?: { message?: string } };
      message = payload.error?.message ?? message;
    } catch { /* retain the status message */ }
    throw new ApiError(message, response.status);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value)) as T;
const funnelKeyByEvent = {
  opened: "opened",
  clicked: "clicked",
  form_attempted: "formAttempted",
  training_viewed: "trainingViewed",
} as const satisfies Record<EventType, keyof CampaignReport["funnel"]>;
let employees: Employee[] = [clone(demoEmployee)];
let campaign: GeneratedCampaign | null = clone(demoCampaign);
let report: CampaignReport | null = null;
let simulation: Simulation | null = null;

function summary(employee: Employee): EmployeeSummary {
  const { employeeId, displayName, department, title, hasProfile } = employee;
  return { employeeId, displayName, department, title, hasProfile };
}

function now(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>'"]/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "'": "&#39;",
    '"': "&quot;",
  })[character]!);
}

function fixtureProfile(employee: Employee): EmployeeProfile {
  const profile = clone(demoEmployee.profile!);
  if (employee.employeeId !== demoEmployee.employeeId) {
    profile.publicFacts[1] = {
      fact: `公開職務頁面顯示與 ${employee.department || "跨部門"} 團隊相關`,
      sourceUrl: "https://example.com/company/team-directory",
      confidence: 0.72,
      sourceType: "fixture",
    };
  }
  return {
    ...profile,
    employeeId: employee.employeeId,
    displayName: employee.displayName,
    department: employee.department,
  };
}

function fixtureCampaign(employee: Employee): GeneratedCampaign {
  const next = clone(demoCampaign);
  return {
    ...next,
    campaignId: `c_${crypto.randomUUID().replaceAll("-", "")}`,
    employeeId: employee.employeeId,
    templateId: employee.profile?.recommendedScenario ?? next.templateId,
    emailHtml: next.emailHtml.replaceAll("Demo User", () => escapeHtml(employee.displayName || "同仁")),
    status: "pending_review",
    approvedBy: null,
    approvedAt: null,
    rejectionReason: null,
  };
}

export const api = {
  usingFixtures: useFixtures,

  async listEmployees(): Promise<EmployeeSummary[]> {
    return useFixtures ? employees.map(summary) : request<EmployeeSummary[]>("/employees");
  },

  async getEmployee(employeeId: string): Promise<Employee> {
    if (!useFixtures) return request<Employee>(`/employees/${encodeURIComponent(employeeId)}`);
    const employee = employees.find((item) => item.employeeId === employeeId);
    if (!employee) throw new ApiError("找不到該員工", 404);
    return clone(employee);
  },

  async importCsv(csv: string): Promise<ImportResult> {
    if (!useFixtures) {
      return request<ImportResult>("/employees/import", { method: "POST", headers: { "Content-Type": "text/csv" }, body: csv });
    }
    const rows = csv.trim().split(/\r?\n/);
    if (rows.length < 2) throw new ApiError("CSV 至少需要標題列與一筆資料", 400);
    const headers = rows[0].split(",").map((item) => item.trim());
    const required = ["employee_id", "display_name", "email", "department", "title", "company"];
    if (required.some((field) => !headers.includes(field))) throw new ApiError("CSV 欄位不符合 API 契約", 400);
    const errors: ImportResult["errors"] = [];
    let imported = 0;
    rows.slice(1).filter(Boolean).forEach((row, index) => {
      const values = row.split(",").map((item) => item.trim());
      const value = (field: string) => values[headers.indexOf(field)] ?? "";
      const employeeId = value("employee_id");
      if (!employeeId || employees.some((item) => item.employeeId === employeeId)) {
        errors.push({ row: index + 2, reason: "duplicate employee_id" });
        return;
      }
      employees.push({
        employeeId, displayName: value("display_name"), email: value("email"), department: value("department"),
        title: value("title"), company: value("company"), hasProfile: false, profile: null,
      });
      imported += 1;
    });
    return { imported, skipped: errors.length, errors };
  },

  async enrich(employeeId: string): Promise<EmployeeProfile> {
    if (!useFixtures) return request<EmployeeProfile>(`/employees/${encodeURIComponent(employeeId)}/enrich`, { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
    const employee = employees.find((item) => item.employeeId === employeeId);
    if (!employee) throw new ApiError("找不到該員工", 404);
    employee.profile = fixtureProfile(employee);
    employee.hasProfile = true;
    return clone(employee.profile);
  },

  async generate(employeeId: string): Promise<GeneratedCampaign> {
    if (!useFixtures) return request<GeneratedCampaign>("/campaigns/generate", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ employeeId }) });
    const employee = employees.find((item) => item.employeeId === employeeId);
    if (!employee?.profile) throw new ApiError("請先建立 Profile", 409);
    campaign = fixtureCampaign(employee);
    report = null;
    simulation = null;
    return clone(campaign);
  },

  async approve(campaignId: string, approvedBy: string): Promise<GeneratedCampaign> {
    if (!useFixtures) return request<GeneratedCampaign>(`/campaigns/${encodeURIComponent(campaignId)}/approve`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ approvedBy }) });
    if (!campaign || campaign.campaignId !== campaignId || campaign.status !== "pending_review") throw new ApiError("Campaign 目前不可核准", 409);
    if (!campaign.safetyChecks.length || campaign.safetyChecks.some((check) => !check.passed)) throw new ApiError("安全規則尚未全部通過，Campaign 不可核准", 409);
    campaign = { ...campaign, status: "approved", approvedBy, approvedAt: now() };
    return clone(campaign);
  },

  async reject(campaignId: string, reason: string): Promise<GeneratedCampaign> {
    if (!useFixtures) return request<GeneratedCampaign>(`/campaigns/${encodeURIComponent(campaignId)}/reject`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ reason }) });
    if (!campaign || campaign.campaignId !== campaignId || campaign.status !== "pending_review") throw new ApiError("Campaign 目前不可拒絕", 409);
    campaign = { ...campaign, status: "rejected", rejectionReason: reason };
    return clone(campaign);
  },

  async simulate(campaignId: string): Promise<Simulation> {
    if (!useFixtures) return request<Simulation>(`/campaigns/${encodeURIComponent(campaignId)}/simulate`, { method: "POST" });
    if (!campaign || campaign.campaignId !== campaignId || campaign.status !== "approved") throw new ApiError("未核准的 Campaign 不可模擬寄送", 409);
    campaign = { ...campaign, status: "simulated" };
    const token = crypto.randomUUID().replaceAll("-", "");
    simulation = { campaignId, status: "simulated", targets: [{ employeeId: campaign.employeeId, token, landingUrl: `/landing/${token}` }] };
    report = { campaignId, targetCount: 1, funnel: { simulated: 1, opened: 0, clicked: 0, formAttempted: 0, trainingViewed: 0 }, events: [] };
    return clone(simulation);
  },

  async recordEvent(token: string, eventType: EventType): Promise<void> {
    if (!useFixtures) return request<void>("/events", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ token, eventType }) });
    if (!simulation || !report || !simulation.targets.some((item) => item.token === token)) throw new ApiError("無效 tracking token", 404);
    if (!report.events.some((event) => event.eventType === eventType)) {
      report.events.push({ eventType, occurredAt: now() });
      report.funnel[funnelKeyByEvent[eventType]] += 1;
    }
  },

  async getReport(campaignId: string): Promise<CampaignReport> {
    if (!useFixtures) return request<CampaignReport>(`/reports/${encodeURIComponent(campaignId)}`);
    if (!report || report.campaignId !== campaignId) throw new ApiError("尚無模擬報告", 404);
    return clone(report);
  },
};
