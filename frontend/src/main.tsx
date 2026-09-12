import { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { ApiError, api } from "./api";
import { demoCampaign, demoEmployee } from "./demo-data";
import type { CampaignReport, Employee, EmployeeSummary, EventType, GeneratedCampaign, PublicFact, Simulation } from "./types";
import "./styles.css";

const statusText = {
  pending_review: "等待人工審核",
  approved: "已核准",
  rejected: "已拒絕",
  simulated: "已模擬寄送",
} as const;
const eventText: Record<EventType, string> = {
  opened: "開啟模擬信件",
  clicked: "點擊 CTA",
  form_attempted: "嘗試提交 Dummy Form",
  training_viewed: "閱讀教育揭露頁",
};
const sampleCsv = "employee_id,display_name,email,department,title,company\nE002,Alex Chen,alex.chen@example.test,Product,Product Manager,Demo Corp\n";

function formatDate(value: string | null): string {
  return value ? new Intl.DateTimeFormat("zh-TW", { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(value)) + " UTC" : "—";
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
        : <span className="fact-source-missing">未提供來源</span>}
    </div>
    <div>
      <span className={`source ${source.className}`}>{source.label}</span>
      <small>可信度 {fact.confidence === null ? "—" : `${Math.round(fact.confidence * 100)}%`}</small>
    </div>
  </article>;
}

function App() {
  const inputRef = useRef<HTMLInputElement>(null);
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
  const [notice, setNotice] = useState("準備好開始安全演練。");
  const [error, setError] = useState<string | null>(null);

  const activeTarget = simulation?.targets[0] ?? null;
  const landingUrl = activeTarget?.landingUrl ?? "{{landingUrl}}";
  const emailDocument = useMemo(() => campaign?.emailHtml.replaceAll("{{landingUrl}}", landingUrl) ?? "", [campaign, landingUrl]);
  const campaignProfile = campaign && campaignEmployee?.employeeId === campaign.employeeId ? campaignEmployee.profile : null;
  const safetyChecksPassed = (campaign?.safetyChecks.length ?? 0) > 0 && (campaign?.safetyChecks.every((check) => check.passed) ?? false);
  const canApprove = Boolean(campaignProfile) && safetyChecksPassed;

  const showError = (reason: unknown) => {
    setError(reason instanceof ApiError ? reason.message : "發生未預期錯誤，請稍後再試。");
  };
  const run = async (name: string, work: () => Promise<void>) => {
    setBusy(name); setError(null);
    try { await work(); } catch (reason) { showError(reason); } finally { setBusy(null); }
  };

  const loadEmployee = async (employeeId: string) => {
    const employee = await api.getEmployee(employeeId);
    setSelected(employee);
  };
  const loadReport = async (campaignId: string) => {
    try { setReport(await api.getReport(campaignId)); } catch (reason) { if (!(reason instanceof ApiError && reason.status === 404)) throw reason; setReport(null); }
  };

  useEffect(() => {
    void run("initial", async () => {
      const list = await api.listEmployees();
      setEmployees(list);
      if (list[0]) await loadEmployee(list[0].employeeId);
    });
  }, []);

  const importCsv = async (file: File) => {
    await run("import", async () => {
      const result = await api.importCsv(await file.text());
      const list = await api.listEmployees();
      setEmployees(list);
      if (!selected && list[0]) await loadEmployee(list[0].employeeId);
      setNotice(`匯入完成：新增 ${result.imported} 位、略過 ${result.skipped} 位。`);
      if (result.errors.length) setError(result.errors.map((item) => `第 ${item.row} 列：${item.reason}`).join("；"));
    });
  };

  const enrich = () => selected && run("enrich", async () => {
    const profile = await api.enrich(selected.employeeId);
    setSelected({ ...selected, profile });
    setEmployees((items) => items.map((item) => item.employeeId === selected.employeeId ? { ...item, hasProfile: true } : item));
    setNotice("Profile 已更新；公開事實已標示來源狀態，未知來源會標記為 unverified。");
  });
  const generate = () => selected && run("generate", async () => {
    const targetEmployee = selected;
    const next = await api.generate(targetEmployee.employeeId);
    setCampaign(next); setCampaignEmployee(targetEmployee); setSimulation(null); setReport(null); setEmailOpened(false); setLandingOpen(false); setTrainingRevealed(false);
    setNotice("Agent 已建立受控情境，等待管理者人工審核。");
  });
  const approve = () => campaign && run("approve", async () => {
    if (!campaignProfile) throw new ApiError("無法確認 Campaign 對應的員工 Profile", 409);
    if (!safetyChecksPassed) throw new ApiError("安全規則尚未全部通過，Campaign 不可核准", 409);
    const next = await api.approve(campaign.campaignId, "hr@example.test");
    setCampaign(next); setNotice("Campaign 已記錄核准人與核准時間，可以模擬寄送。");
  });
  const reject = () => campaign && run("reject", async () => {
    const reason = window.prompt("請輸入拒絕原因：", "內容需調整")?.trim();
    if (!reason) return;
    setCampaign(await api.reject(campaign.campaignId, reason));
    setNotice("Campaign 已拒絕，不可進行模擬寄送。");
  });
  const simulate = () => campaign && run("simulate", async () => {
    const next = await api.simulate(campaign.campaignId);
    setSimulation(next); setCampaign({ ...campaign, status: "simulated" }); setEmailOpened(false); setLandingOpen(false); setTrainingRevealed(false);
    await loadReport(campaign.campaignId);
    setNotice("已建立不含個資的 tracking token，現在可切換至員工視角。");
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
    ["模擬寄送", report.funnel.simulated], ["開啟信件", report.funnel.opened], ["點擊 CTA", report.funnel.clicked],
    ["表單嘗試", report.funnel.formAttempted], ["教育頁閱讀", report.funnel.trainingViewed],
  ] : [];

  return <main className="app-shell">
    <aside className="sidebar">
      <div className="brand"><span className="brand-mark">S</span><div>SimSafe<small>Security training console</small></div></div>
      <nav><a href="#employees">員工與 Profile</a><a href="#review">審核工作台</a><a href="#simulation">模擬信件</a><a href="#dashboard">成效 Dashboard</a></nav>
      <div className="side-note"><span className="live-dot" />{api.usingFixtures ? "Fixture demo mode" : "Go API integration mode"}<p>資料僅用於授權的安全演練。</p></div>
    </aside>
    <section className="content">
      <header className="topbar"><div><p className="eyebrow">AI PHISHING SIMULATION</p><h1>安全演練管理台</h1></div><div className="mode-badge">{api.usingFixtures ? "展示資料" : "API 已串接"}</div></header>
      {error && <div className="alert error" role="alert">{error}<button onClick={() => setError(null)} aria-label="關閉">×</button></div>}
      <div className="alert info">{notice}</div>

      <section id="employees" className="panel employees-panel">
        <div className="panel-heading"><div><p className="eyebrow">01 / DATA INTAKE</p><h2>員工與可追溯 Profile</h2></div><div className="actions"><button className="secondary" onClick={() => { const blob = new Blob([sampleCsv], { type: "text/csv" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = "employees-template.csv"; a.click(); URL.revokeObjectURL(url); }}>下載 CSV 範本</button><button onClick={() => inputRef.current?.click()} disabled={busy === "import"}>{busy === "import" ? "匯入中…" : "上傳 CSV"}</button><input ref={inputRef} type="file" accept=".csv,text/csv" hidden onChange={(event) => { const file = event.target.files?.[0]; if (file) void importCsv(file); event.currentTarget.value = ""; }} /></div></div>
        <div className="employee-grid"><div className="employee-list">{employees.map((employee) => <button key={employee.employeeId} className={`employee-row ${selected?.employeeId === employee.employeeId ? "selected" : ""}`} onClick={() => void run("employee", () => loadEmployee(employee.employeeId))}><span className="avatar">{employee.displayName.slice(0, 1)}</span><span><b>{employee.displayName}</b><small>{employee.department} · {employee.title}</small></span>{employee.hasProfile && <span className="check">✓</span>}</button>)}{!employees.length && <p className="empty">尚未匯入員工。</p>}</div>
        <div className="profile-card">{selected ? <><div className="profile-title"><div><p className="eyebrow">EMPLOYEE {selected.employeeId}</p><h3>{selected.displayName}</h3><p>{selected.title} · {selected.department}</p></div><button className="secondary" onClick={enrich} disabled={busy === "enrich"}>{busy === "enrich" ? "分析中…" : selected.profile ? "重新 Enrich" : "建立 Profile"}</button></div>{selected.profile ? <><div className="facts"><h4>公開事實 <span>來源可核對</span></h4>{selected.profile.publicFacts.map((fact) => <FactRow key={`${fact.fact}-${fact.sourceUrl ?? "unknown"}`} fact={fact} />)}</div><div className="risk-box"><b>Agent 建議：{selected.profile.recommendedScenario}</b>{selected.profile.riskSignals.map((signal) => <p key={signal}>• {signal}</p>)}</div></> : <div className="empty large">尚無 Profile。建立後才可生成 Campaign。</div>}</> : <div className="empty large">請先選擇一位員工。</div>}</div></div>
        <div className="step-footer"><span>資料最小化：公開事實須保留來源；推論不會當作已確認事實。</span><button onClick={generate} disabled={!selected?.profile || busy === "generate"}>{busy === "generate" ? "生成中…" : "生成安全演練"} <span>→</span></button></div>
      </section>

      <section id="review" className="panel review-panel"><div className="panel-heading"><div><p className="eyebrow">02 / HUMAN REVIEW</p><h2>Agent 決策與內容審核</h2></div>{campaign && <span className={`status ${campaign.status}`}>{statusText[campaign.status]}</span>}</div>{campaign ? <div className="review-grid"><div><div className="decision"><span className="icon">✦</span><div><p className="eyebrow">RECOMMENDED SCENARIO</p><h3>{campaign.templateId} <span className="difficulty">{campaign.difficulty}</span></h3><p>{campaign.decisionReason}</p></div></div><div className="campaign-provenance"><p className="eyebrow">CAMPAIGN TARGET</p><h4>{campaignEmployee?.employeeId === campaign.employeeId ? `${campaignEmployee.displayName} · ${campaign.employeeId}` : campaign.employeeId}</h4><p className="provenance-label">生成內容使用的公開資訊與來源</p>{campaignProfile ? campaignProfile.publicFacts.length ? campaignProfile.publicFacts.map((fact) => <FactRow key={`${fact.fact}-${fact.sourceUrl ?? "unknown"}`} fact={fact} />) : <p className="empty">此 Profile 沒有可回報的公開事實。</p> : <p className="safety-blocker">無法確認此 Campaign 對應的 Profile，禁止核准。</p>}</div><div className="checks"><h4>安全規則檢查</h4>{campaign.safetyChecks.map((check) => <div key={check.rule} className={check.passed ? "passed" : "failed"}><span>{check.passed ? "✓" : "!"}</span><code>{check.rule}</code><small>{check.passed ? "passed" : check.detail ?? "failed"}</small></div>)}{!safetyChecksPassed && <p className="safety-blocker">安全規則尚未全部通過，必須拒絕或重新生成內容。</p>}</div><div className="review-actions">{campaign.status === "pending_review" && <><button className="danger-outline" onClick={reject} disabled={Boolean(busy)}>拒絕</button><button onClick={approve} disabled={Boolean(busy) || !canApprove} title={!canApprove ? "需確認 Profile 且所有安全規則通過" : undefined}>核准 Campaign</button></>}{campaign.status === "approved" && <button onClick={simulate} disabled={busy === "simulate"}>{busy === "simulate" ? "建立 token 中…" : "模擬寄送 →"}</button>}{campaign.status === "simulated" && <span className="success-text">✓ 已建立受控 Landing URL</span>}{campaign.status === "rejected" && <span className="rejected-text">拒絕原因：{campaign.rejectionReason}</span>}</div></div><div className="preview"><div className="preview-header"><span>信件預覽</span><b>Subject: {campaign.subject}</b></div><iframe title="安全信件預覽" sandbox="allow-popups" srcDoc={emailDocument} /></div><div className="landing-card"><p className="eyebrow">LANDING PAGE PREVIEW</p><h3>{campaign.landingConfig.brand}</h3><h4>{campaign.landingConfig.title}</h4><p>{campaign.landingConfig.description}</p><button disabled>{campaign.landingConfig.ctaLabel}</button><small>測試品牌 · Dummy Form · 不收集帳密</small></div></div> : <div className="empty large">先完成 Profile 後，建立可人工核准的 Campaign。</div>}</section>

      <section id="simulation" className="panel simulation-panel">
        <div className="panel-heading"><div><p className="eyebrow">03 / EMPLOYEE VIEW</p><h2>模擬信件與受控 Landing</h2></div>{activeTarget && <code className="token">token · {activeTarget.token.slice(0, 10)}…</code>}</div>
        {campaign?.status === "simulated" && activeTarget ? <div className="simulation-grid">
          <div>
            <p>以員工視角查看信件時才記錄 <code>opened</code>；其餘事件由 Landing Page 依互動階段記錄。</p>
            <button onClick={() => record("opened", () => setEmailOpened(true))} disabled={Boolean(busy)}>{emailOpened ? "✓ 已開啟信件" : "開啟模擬信件"}</button>
            {emailOpened && <div className="mail-preview">
              <b>{campaign.subject}</b>
              <iframe title="員工信件視角" sandbox="allow-popups" srcDoc={emailDocument} />
              <div className="actions">
                {api.usingFixtures
                  ? <button className="secondary" onClick={openFixtureLanding}>模擬點擊 CTA</button>
                  : <a href={activeTarget.landingUrl} target="_blank" rel="noreferrer">開啟受控 Landing ↗</a>}
              </div>
            </div>}
          </div>
          <div className={`training-card ${landingOpen || !api.usingFixtures ? "visible" : ""}`}>
            {api.usingFixtures ? landingOpen ? <>
              <p className="eyebrow">CONTROLLED LANDING</p>
              <h3>{campaign.landingConfig.title}</h3>
              {trainingRevealed ? <div className="notice">
                <h4>這是一場受控資安演練</h4>
                <p>輸入資料前，請確認寄件者、網址與需求是否合理，並避免在不受信任的頁面輸入密碼、MFA 或金融資料。</p>
                <p className="privacy">本次演練只記錄互動事件，不保存表單輸入值。</p>
              </div> : <>
                <p>{campaign.landingConfig.description}</p>
                <form onSubmit={(event) => { event.preventDefault(); void submitFixtureLanding(); }}>
                  <input aria-label="Dummy email" placeholder="這是 Dummy Form（內容不會送出）" />
                  <button type="submit" disabled={Boolean(busy)}>{campaign.landingConfig.ctaLabel}</button>
                </form>
                <p className="privacy">🔒 不傳送、不保存任何表單原始輸入值。</p>
              </>}
            </> : <div className="empty large">點擊 CTA 後顯示 Landing Page 與教育揭露。</div> : <>
              <p className="eyebrow">ROLE B LANDING</p>
              <h3>Landing 互動由 Go 頁面處理</h3>
              <p>受控頁面載入時記錄 <code>clicked</code>，提交 Dummy Form 後依序記錄 <code>form_attempted</code> 與 <code>training_viewed</code>。</p>
              <a href={activeTarget.landingUrl} target="_blank" rel="noreferrer">開啟受控 Landing ↗</a>
            </>}
          </div>
        </div> : <div className="empty large">需先由管理者核准並模擬寄送，才能切換員工視角。</div>}
      </section>

      <section id="dashboard" className="panel dashboard"><div className="panel-heading"><div><p className="eyebrow">04 / REPORTING</p><h2>演練成效 Dashboard</h2></div>{report && <button className="secondary" onClick={() => campaign && void run("report", () => loadReport(campaign.campaignId))}>重新整理</button>}</div>{report ? <><div className="metrics">{funnel.map(([label, value]) => <div key={label} className="metric"><span>{label}</span><b>{value}</b><small>{Math.round((Number(value) / maximum) * 100)}% of targets</small></div>)}</div><div className="report-grid"><div className="funnel-chart"><h3>轉換漏斗</h3>{funnel.map(([label, value]) => <div className="bar-row" key={label}><span>{label}</span><div><i style={{ width: `${Math.max((Number(value) / maximum) * 100, Number(value) ? 10 : 0)}%` }} /></div><b>{value}/{maximum}</b></div>)}</div><div className="timeline"><h3>事件時間軸</h3>{report.events.length ? report.events.map((event, index) => <div className="event" key={`${event.eventType}-${event.occurredAt}`}><span>{index + 1}</span><div><b>{eventText[event.eventType]}</b><small>{formatDate(event.occurredAt)}</small></div></div>) : <p className="empty">尚未有互動事件。</p>}</div></div><p className="dashboard-note">開信為參考指標；點擊與頁面互動更能反映演練成效。所有數字均為 distinct target 計數。</p></> : <div className="empty large">完成模擬寄送後，Dashboard 會顯示漏斗與事件時間軸。</div>}</section>
    </section>
  </main>;
}

createRoot(document.getElementById("root")!).render(<App />);
