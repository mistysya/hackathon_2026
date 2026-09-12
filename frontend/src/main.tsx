import { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { ApiError, api } from "./api";
import { demoCampaign, demoEmployee } from "./demo-data";
import type { CampaignReport, Employee, EmployeeSummary, EventType, GeneratedCampaign, PublicFact, Simulation } from "./types";
import "./styles.css";

const statusText = {
  pending_review: "Pending review",
  approved: "Approved",
  rejected: "Rejected",
  simulated: "Simulated delivery",
} as const;
const eventText: Record<EventType, string> = {
  opened: "Opened simulated email",
  clicked: "Clicked CTA",
  form_attempted: "Submitted dummy form",
  training_viewed: "Viewed training reveal",
};
const sampleCsv = "employee_id,display_name,email,department,title,company\nE001,Demo User,demo.user@example.test,Engineering,Software Engineer,Demo Corp\nE002,Alex Chen,alex.chen@example.test,Product,Product Manager,Demo Corp\n";
const mailboxUrl = import.meta.env.VITE_MAILBOX_URL || `${window.location.protocol}//${window.location.hostname}:4173/`;
const mailboxOrigin = new URL(mailboxUrl, window.location.href).origin;

type MailboxDelivery = {
  campaign: GeneratedCampaign;
  target: Simulation["targets"][number];
  acknowledged: boolean;
};

function formatDate(value: string | null): string {
  return value
    ? `${new Intl.DateTimeFormat("en-US", { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(value))} UTC`
    : "—";
}

function FactRow({ fact }: { fact: PublicFact }) {
  const source = fact.sourceType === "fixture"
    ? { label: "mock", className: "fixture" }
    : fact.sourceUrl
      ? { label: "source-backed", className: fact.sourceType }
      : { label: "unverified", className: "unknown" };

  return <article className="fact">
    <div>
      <b>{fact.fact}</b>
      {fact.sourceUrl
        ? <a href={fact.sourceUrl} target="_blank" rel="noreferrer">{fact.sourceUrl}</a>
        : <span className="fact-source-missing">No source provided</span>}
    </div>
    <div>
      <span className={`source ${source.className}`}>{source.label}</span>
      <small>Confidence {fact.confidence === null ? "—" : `${Math.round(fact.confidence * 100)}%`}</small>
    </div>
  </article>;
}

function App() {
  const inputRef = useRef<HTMLInputElement>(null);
  const mailboxWindowRef = useRef<Window | null>(null);
  const mailboxReadyRef = useRef(false);
  const mailboxDeliveriesRef = useRef(new Map<string, MailboxDelivery>());
  const pendingOpenedRef = useRef(new Set<string>());
  const currentCampaignIdRef = useRef<string | undefined>(undefined);
  const [employees, setEmployees] = useState<EmployeeSummary[]>([]);
  const [selected, setSelected] = useState<Employee | null>(null);
  const [campaign, setCampaign] = useState<GeneratedCampaign | null>(api.usingFixtures ? demoCampaign : null);
  const [campaignEmployee, setCampaignEmployee] = useState<Employee | null>(api.usingFixtures ? demoEmployee : null);
  const [simulation, setSimulation] = useState<Simulation | null>(null);
  const [report, setReport] = useState<CampaignReport | null>(null);
  const [emailOpened, setEmailOpened] = useState(false);
  const [landingOpen, setLandingOpen] = useState(false);
  const [trainingRevealed, setTrainingRevealed] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const [notice, setNotice] = useState("Ready to start a safe security exercise.");
  const [error, setError] = useState<string | null>(null);

  currentCampaignIdRef.current = campaign?.campaignId;
  const activeTarget = simulation?.targets[0] ?? null;
  const landingUrl = activeTarget?.landingUrl ?? "{{landingUrl}}";
  const emailDocument = useMemo(() => campaign?.emailHtml.replaceAll("{{landingUrl}}", landingUrl) ?? "", [campaign, landingUrl]);
  const campaignProfile = campaign && campaignEmployee?.employeeId === campaign.employeeId ? campaignEmployee.profile : null;
  const safetyChecksPassed = (campaign?.safetyChecks.length ?? 0) > 0 && (campaign?.safetyChecks.every((check) => check.passed) ?? false);
  const canApprove = Boolean(campaignProfile) && safetyChecksPassed;

  const showError = (reason: unknown) => {
    setError(reason instanceof ApiError ? reason.message : "An unexpected error occurred. Please try again.");
  };
  const run = async (name: string, work: () => Promise<void>) => {
    setBusy(name);
    setError(null);
    try {
      await work();
    } catch (reason) {
      showError(reason);
    } finally {
      setBusy(null);
    }
  };

  const loadEmployee = async (employeeId: string) => {
    setSelected(await api.getEmployee(employeeId));
  };
  const loadReport = async (campaignId: string) => {
    try {
      setReport(await api.getReport(campaignId));
    } catch (reason) {
      if (!(reason instanceof ApiError && reason.status === 404)) throw reason;
      setReport(null);
    }
  };
  const flushMailboxDeliveries = () => {
    const mailbox = mailboxWindowRef.current;
    if (!mailbox || mailbox.closed || !mailboxReadyRef.current) return;
    for (const delivery of mailboxDeliveriesRef.current.values()) {
      if (!delivery.acknowledged) {
        mailbox.postMessage({
          type: "simsafe:campaign-delivered",
          campaign: delivery.campaign,
          target: delivery.target,
        }, mailboxOrigin);
      }
    }
  };
  const openEmployeeMailbox = () => {
    if (api.usingFixtures) return;
    const existing = mailboxWindowRef.current;
    if (existing && !existing.closed) {
      existing.focus();
      flushMailboxDeliveries();
      return;
    }
    const mailbox = window.open(mailboxUrl, "simsafe-employee-mailbox");
    if (!mailbox) {
      setError("Your browser blocked the employee mailbox. Allow pop-ups and try again.");
      return;
    }
    mailboxWindowRef.current = mailbox;
    mailboxReadyRef.current = false;
    setNotice("Opening the employee mailbox; the simulated email will be delivered once it loads.");
  };
  const deliverToMailbox = (deliveredCampaign: GeneratedCampaign, deliveredSimulation: Simulation) => {
    if (api.usingFixtures) return;
    const target = deliveredSimulation.targets[0];
    if (!target) return;
    mailboxDeliveriesRef.current.set(deliveredCampaign.campaignId, {
      campaign: deliveredCampaign,
      target: { ...target, landingUrl: new URL(target.landingUrl, window.location.origin).href },
      acknowledged: false,
    });
    flushMailboxDeliveries();
  };

  useEffect(() => {
    void run("initial", async () => {
      const list = await api.listEmployees();
      setEmployees(list);
      if (list[0]) await loadEmployee(list[0].employeeId);
    });
  }, []);

  useEffect(() => {
    if (api.usingFixtures) return;
    const receiveMailboxEvent = (event: MessageEvent) => {
      const data = event.data as { type?: string; campaignId?: string; token?: string; eventType?: EventType } | null;
      const mailbox = mailboxWindowRef.current;
      if (!mailbox || event.origin !== mailboxOrigin || event.source !== mailbox || !data) return;
      if (data.type === "simsafe:mailbox-ready") {
        mailboxReadyRef.current = true;
        for (const delivery of mailboxDeliveriesRef.current.values()) delivery.acknowledged = false;
        flushMailboxDeliveries();
        return;
      }
      if (typeof data.campaignId !== "string" || typeof data.token !== "string") return;
      const delivery = mailboxDeliveriesRef.current.get(data.campaignId);
      if (!delivery || delivery.target.token !== data.token) return;
      if (data.type === "simsafe:delivery-ack") {
        const wasAcknowledged = delivery.acknowledged;
        delivery.acknowledged = true;
        if (!wasAcknowledged && currentCampaignIdRef.current === data.campaignId) {
          setNotice("Simulation complete; the new email was delivered to the employee mailbox.");
        }
        return;
      }
      if (data.type !== "simsafe:mailbox-event" || data.eventType !== "opened" || pendingOpenedRef.current.has(data.token)) return;
      const { campaignId, token } = data;
      pendingOpenedRef.current.add(token);
      void run("mailbox-opened", async () => {
        try {
          await api.recordEvent(token, "opened");
          mailbox.postMessage({ type: "simsafe:event-ack", campaignId, token, eventType: "opened" }, mailboxOrigin);
          if (currentCampaignIdRef.current !== campaignId) return;
          setReport(await api.getReport(campaignId));
          setEmailOpened(true);
          setNotice("The employee opened the new email; the opened event was recorded.");
        } finally {
          pendingOpenedRef.current.delete(token);
        }
      });
    };
    window.addEventListener("message", receiveMailboxEvent);
    const retryTimer = window.setInterval(flushMailboxDeliveries, 1000);
    return () => {
      window.removeEventListener("message", receiveMailboxEvent);
      window.clearInterval(retryTimer);
    };
  }, []);

  const importCsv = async (file: File) => {
    await run("import", async () => {
      const result = await api.importCsv(await file.text());
      const list = await api.listEmployees();
      setEmployees(list);
      if (!selected && list[0]) await loadEmployee(list[0].employeeId);
      setNotice(`Import complete: ${result.imported} added, ${result.skipped} skipped.`);
      if (result.errors.length) setError(result.errors.map((item) => `Row ${item.row}: ${item.reason}`).join("; "));
    });
  };
  const enrich = () => selected && run("enrich", async () => {
    const profile = await api.enrich(selected.employeeId);
    setSelected({ ...selected, profile });
    setEmployees((items) => items.map((item) => item.employeeId === selected.employeeId ? { ...item, hasProfile: true } : item));
    setNotice("Profile updated. Public facts include source status, and unknown sources are marked unverified.");
  });
  const generate = () => selected && run("generate", async () => {
    const targetEmployee = selected;
    const next = await api.generate(targetEmployee.employeeId);
    setCampaign(next);
    setCampaignEmployee(targetEmployee);
    setSimulation(null);
    setReport(null);
    setEmailOpened(false);
    setLandingOpen(false);
    setTrainingRevealed(false);
    setNotice("A controlled scenario was created and is ready for human review.");
  });
  const approve = () => campaign && run("approve", async () => {
    if (!campaignProfile) throw new ApiError("Unable to verify the employee profile for this campaign.", 409);
    if (!safetyChecksPassed) throw new ApiError("All safety checks must pass before this campaign can be approved.", 409);
    setCampaign(await api.approve(campaign.campaignId, "hr@example.test"));
    setNotice("Campaign approval and timestamp are recorded. It can now be simulated.");
  });
  const reject = () => campaign && run("reject", async () => {
    const reason = window.prompt("Enter a reason for rejection:", "Content needs revision")?.trim();
    if (!reason) return;
    setCampaign(await api.reject(campaign.campaignId, reason));
    setNotice("Campaign rejected; simulated delivery is unavailable.");
  });
  const simulate = () => campaign && run("simulate", async () => {
    const next = await api.simulate(campaign.campaignId);
    const deliveredCampaign: GeneratedCampaign = { ...campaign, status: "simulated" };
    setSimulation(next);
    setCampaign(deliveredCampaign);
    setEmailOpened(false);
    setLandingOpen(false);
    setTrainingRevealed(false);
    deliverToMailbox(deliveredCampaign, next);
    setNotice(api.usingFixtures
      ? "Simulation complete. Use the employee view below to complete the offline exercise."
      : "Simulation complete. The email is awaiting delivery confirmation; if it has not arrived, select Open employee mailbox.");
    await loadReport(campaign.campaignId);
  });
  const record = (eventType: EventType, after?: () => void) => activeTarget && run(`event-${eventType}`, async () => {
    await api.recordEvent(activeTarget.token, eventType);
    if (campaign) await loadReport(campaign.campaignId);
    after?.();
  });
  const openFixtureLanding = () => activeTarget && run("fixture-landing", async () => {
    await api.recordEvent(activeTarget.token, "clicked");
    if (campaign) await loadReport(campaign.campaignId);
    setLandingOpen(true);
    setTrainingRevealed(false);
  });
  const submitFixtureLanding = () => activeTarget && run("fixture-training", async () => {
    await api.recordEvent(activeTarget.token, "form_attempted");
    setTrainingRevealed(true);
    await api.recordEvent(activeTarget.token, "training_viewed");
    if (campaign) await loadReport(campaign.campaignId);
  });

  const maximum = report?.targetCount || 1;
  const funnel = report ? [
    ["Simulated delivery", report.funnel.simulated],
    ["Opened email", report.funnel.opened],
    ["Clicked CTA", report.funnel.clicked],
    ["Form submission", report.funnel.formAttempted],
    ["Training page viewed", report.funnel.trainingViewed],
  ] : [];

  return <main className="app-shell" id="top">
    <header className="app-header">
      <a className="brand" href="#top" aria-label="SimSafe home">
        <span className="brand-mark">S</span>
        <span>SimSafe<small>Security training</small></span>
      </a>
      <nav aria-label="Console sections">
        <a href="#employees">Employees</a>
        <a href="#review">Review</a>
        <a href="#simulation">Simulation</a>
        <a href="#dashboard">Results</a>
      </nav>
      <span className="mode-badge">{api.usingFixtures ? "Demo data" : "API connected"}</span>
    </header>

    <section className="content">
      <header className="page-heading">
        <div>
          <h1>Security Exercise Console</h1>
          <p>Plan, review, and measure authorized security training.</p>
        </div>
        <p className="environment"><span className="live-dot" />{api.usingFixtures ? "Fixture demo mode" : "Go API integration mode"}</p>
      </header>

      {error && <div className="alert error" role="alert">{error}<button onClick={() => setError(null)} aria-label="Close">×</button></div>}
      <div className="alert info" aria-live="polite">{notice}</div>

      <section id="employees" className="panel employees-panel">
        <div className="panel-heading">
          <div><p className="section-label">01 · Employees</p><h2>Employee profiles</h2></div>
          <div className="actions">
            <button className="secondary" onClick={() => {
              const blob = new Blob([sampleCsv], { type: "text/csv" });
              const url = URL.createObjectURL(blob);
              const anchor = document.createElement("a");
              anchor.href = url;
              anchor.download = "employees-template.csv";
              anchor.click();
              URL.revokeObjectURL(url);
            }}>Download CSV template</button>
            <button onClick={() => inputRef.current?.click()} disabled={Boolean(busy)}>{busy === "import" ? "Uploading…" : "Upload CSV"}</button>
            <input ref={inputRef} type="file" accept=".csv,text/csv" hidden onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) void importCsv(file);
              event.currentTarget.value = "";
            }} />
          </div>
        </div>
        <div className="employee-grid">
          <div className="employee-list">
            {employees.map((employee) => <button
              key={employee.employeeId}
              className={`employee-row ${selected?.employeeId === employee.employeeId ? "selected" : ""}`}
              disabled={Boolean(busy)}
              onClick={() => void run("employee", () => loadEmployee(employee.employeeId))}
            >
              <span className="avatar">{employee.displayName.slice(0, 1)}</span>
              <span><b>{employee.displayName}</b><small>{employee.department} · {employee.title}</small></span>
              {employee.hasProfile && <span className="check" aria-label="Profile available">✓</span>}
            </button>)}
            {!employees.length && <p className="empty">No employees imported.</p>}
          </div>
          <div className="profile-card">
            {selected ? <>
              <div className="profile-title">
                <div><p className="section-label">Employee {selected.employeeId}</p><h3>{selected.displayName}</h3><p>{selected.title} · {selected.department}</p></div>
                <button className="secondary" onClick={enrich} disabled={Boolean(busy)}>{busy === "enrich" ? "Analyzing…" : selected.profile ? "Refresh profile" : "Create profile"}</button>
              </div>
              {selected.profile ? <>
                <div className="facts"><h4>Public facts</h4>{selected.profile.publicFacts.map((fact) => <FactRow key={`${fact.fact}-${fact.sourceUrl ?? "unknown"}`} fact={fact} />)}</div>
                <div className="risk-box"><b>Recommended scenario: {selected.profile.recommendedScenario}</b>{selected.profile.riskSignals.map((signal) => <p key={signal}>{signal}</p>)}</div>
              </> : <div className="empty large">Create a profile before generating a campaign.</div>}
            </> : <div className="empty large">Select an employee to begin.</div>}
          </div>
        </div>
        <div className="step-footer"><span>Sources are retained for verification; inferences are not treated as confirmed facts.</span><button onClick={generate} disabled={!selected?.profile || Boolean(busy)}>{busy === "generate" ? "Generating…" : "Generate exercise"} <span>→</span></button></div>
      </section>

      <section id="review" className="panel review-panel">
        <div className="panel-heading">
          <div><p className="section-label">02 · Review</p><h2>Campaign review</h2></div>
          <div className="actions">
            {campaign && <span className={`status ${campaign.status}`}>{statusText[campaign.status]}</span>}
            <button className="secondary" onClick={openEmployeeMailbox} disabled={api.usingFixtures} title={api.usingFixtures ? "Employee mailbox is only available in Go API mode." : undefined}>Open employee mailbox ↗</button>
          </div>
        </div>
        {api.usingFixtures && <p className="helper-text">Employee mailbox requires the Go API. Use the employee view below for an offline demo.</p>}
        {campaign ? <div className="review-grid">
          <div className="review-summary">
            <div className="decision">
              <div><p className="section-label">Recommended scenario</p><h3>{campaign.templateId} <span className="difficulty">{campaign.difficulty}</span></h3><p>{campaign.decisionReason}</p></div>
            </div>
            <div className="campaign-provenance">
              <h4>Profile sources</h4>
              <p className="provenance-label">{campaignEmployee?.employeeId === campaign.employeeId ? `${campaignEmployee.displayName} · ${campaign.employeeId}` : campaign.employeeId}</p>
              {campaignProfile ? campaignProfile.publicFacts.length
                ? campaignProfile.publicFacts.map((fact) => <FactRow key={`${fact.fact}-${fact.sourceUrl ?? "unknown"}`} fact={fact} />)
                : <p className="empty">No public facts are available for this profile.</p>
                : <p className="safety-blocker">The employee profile cannot be verified, so approval is blocked.</p>}
            </div>
            <div className="checks">
              <h4>Safety checks</h4>
              {campaign.safetyChecks.map((check) => <div key={check.rule} className={check.passed ? "passed" : "failed"}><span>{check.passed ? "✓" : "!"}</span><code>{check.rule}</code><small>{check.passed ? "Passed" : check.detail ?? "Failed"}</small></div>)}
              {!safetyChecksPassed && <p className="safety-blocker">All safety checks must pass. Reject or regenerate the campaign.</p>}
            </div>
            <div className="review-actions">
              {campaign.status === "pending_review" && <><button className="danger-outline" onClick={reject} disabled={Boolean(busy)}>Reject</button><button onClick={approve} disabled={Boolean(busy) || !canApprove} title={!canApprove ? "A verified profile and passing safety checks are required" : undefined}>Approve campaign</button></>}
              {campaign.status === "approved" && <button onClick={simulate} disabled={Boolean(busy)}>{busy === "simulate" ? "Creating token…" : "Simulate delivery →"}</button>}
              {campaign.status === "simulated" && <span className="success-text">✓ Controlled landing URL created</span>}
              {campaign.status === "rejected" && <span className="rejected-text">Rejection reason: {campaign.rejectionReason}</span>}
            </div>
          </div>
          <div className="review-previews">
            <div className="preview"><div className="preview-header"><span>Email preview</span><b>{campaign.subject}</b></div><iframe title="Safe email preview" sandbox="" inert srcDoc={emailDocument} /></div>
            <div className="landing-card"><p className="section-label">Landing page preview</p><h3>{campaign.landingConfig.brand}</h3><h4>{campaign.landingConfig.title}</h4><p>{campaign.landingConfig.description}</p><button disabled>{campaign.landingConfig.ctaLabel}</button></div>
          </div>
        </div> : <div className="empty large">Create a profile, then generate a campaign for review.</div>}
      </section>

      <section id="simulation" className="panel simulation-panel">
        <div className="panel-heading"><div><p className="section-label">03 · Simulation</p><h2>Employee view</h2></div>{activeTarget && <code className="token">token · {activeTarget.token.slice(0, 10)}…</code>}</div>
        {campaign?.status === "simulated" && activeTarget ? <div className="simulation-grid">
          <div>
            <p className="helper-text">Opening the email records <code>opened</code>. Landing page interactions are recorded separately.</p>
            <button onClick={() => record("opened", () => setEmailOpened(true))} disabled={Boolean(busy)}>{emailOpened ? "✓ Email opened" : "Open simulated email"}</button>
            {emailOpened && <div className="mail-preview">
              <b>{campaign.subject}</b>
              <iframe title="Employee email view" sandbox="allow-popups allow-popups-to-escape-sandbox" srcDoc={`<base target="_blank">${emailDocument}`} inert={api.usingFixtures} />
              <div className="actions">{api.usingFixtures
                ? <button className="secondary" disabled={Boolean(busy)} onClick={openFixtureLanding}>Simulate CTA click</button>
                : <a href={activeTarget.landingUrl} target="_blank" rel="noreferrer">Open controlled landing ↗</a>}
              </div>
            </div>}
          </div>
          <div className={`training-card ${landingOpen || !api.usingFixtures ? "visible" : ""}`}>
            {api.usingFixtures ? landingOpen ? <>
              <p className="section-label">Controlled landing</p>
              <h3>{campaign.landingConfig.title}</h3>
              {trainingRevealed ? <div className="notice">
                <h4>This is a controlled security exercise</h4>
                <p>Before entering data, verify the sender, URL, and request. Never enter passwords, MFA codes, or financial data on an untrusted page.</p>
                <p className="privacy">This exercise records interaction events only; form values are never stored.</p>
              </div> : <>
                <p>{campaign.landingConfig.description}</p>
                <form onSubmit={(event) => { event.preventDefault(); void submitFixtureLanding(); }}>
                  <input aria-label="Dummy email" placeholder="This is a dummy form. Content is not submitted." />
                  <button type="submit" disabled={Boolean(busy)}>{campaign.landingConfig.ctaLabel}</button>
                </form>
                <p className="privacy">No form values are transmitted or stored.</p>
              </>}
            </> : <div className="empty large">Click the CTA to open the landing page and training reveal.</div> : <>
              <p className="section-label">Controlled landing</p>
              <h3>Landing interactions are handled by the Go page</h3>
              <p>The landing page records <code>clicked</code>, <code>form_attempted</code>, and <code>training_viewed</code>.</p>
              <a href={activeTarget.landingUrl} target="_blank" rel="noreferrer">Open controlled landing ↗</a>
            </>}
          </div>
        </div> : <div className="empty large">Approve and simulate a campaign to open the employee view.</div>}
      </section>

      <section id="dashboard" className="panel dashboard">
        <div className="panel-heading"><div><p className="section-label">04 · Results</p><h2>Exercise results</h2></div>{report && <button className="secondary" disabled={Boolean(busy)} onClick={() => campaign && void run("report", () => loadReport(campaign.campaignId))}>Refresh</button>}</div>
        {report ? <>
          <div className="metrics">{funnel.map(([label, value]) => <div key={label} className="metric"><span>{label}</span><b>{value}</b><small>{Math.round((Number(value) / maximum) * 100)}% of targets</small></div>)}</div>
          <div className="report-grid">
            <div className="funnel-chart"><h3>Conversion funnel</h3>{funnel.map(([label, value]) => <div className="bar-row" key={label}><span>{label}</span><div><i style={{ width: `${Math.max((Number(value) / maximum) * 100, Number(value) ? 10 : 0)}%` }} /></div><b>{value}/{maximum}</b></div>)}</div>
            <div className="timeline"><h3>Event timeline</h3>{report.events.length ? report.events.map((event, index) => <div className="event" key={`${event.eventType}-${event.occurredAt}`}><span>{index + 1}</span><div><b>{eventText[event.eventType]}</b><small>{formatDate(event.occurredAt)}</small></div></div>) : <p className="empty">No interaction events yet.</p>}</div>
          </div>
        </> : <div className="empty large">Results will appear after simulated delivery.</div>}
      </section>
    </section>
  </main>;
}

createRoot(document.getElementById("root")!).render(<App />);
