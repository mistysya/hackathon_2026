# SimSafe presentation guide

## Part 1 — Live demo

### Start the demo

Requirements: Go, Node.js, npm, Python 3, and free ports `8080`, `5173`, `4173`.

```sh
# One time, from the repository root
npm --prefix frontend ci

# Terminal 1: Go API + admin UI
AGENT_MODE=fixture bash scripts/demo.sh

# Terminal 2: employee mailbox
python3 -m http.server 4173 --bind 127.0.0.1 --directory presentation
```

Open <http://127.0.0.1:5173>. Confirm the sidebar says **Go API integration mode**.

`AGENT_MODE=fixture` keeps the demo fast and repeatable. The frontend still uses the real Go API and SQLite.

### Demo flow (3 minutes)

1. **Import** — Upload `docs/fixtures/employees.csv`.
2. **Profile** — Select **Demo User / E001** and click **建立 Profile**.
   - Show the public facts, sources, confidence, and risk signals.
3. **Generate** — Click **生成安全演練**.
   - Show the selected scenario, email preview, landing preview, and safety checks.
4. **Open mailbox** — Click **開啟員工信箱 ↗** and keep the window open.
5. **Human review** — Click **核准 Campaign**, then **模擬寄送 →**.
   - Wait for **新郵件已送達員工信箱**.
6. **Employee view** — The generated email appears at the top of the 13 background emails.
   - Open it to record `opened`.
   - Click its CTA to record `clicked` and open the controlled landing page.
7. **Safe training** — Enter any dummy text and submit.
   - The input is not sent or stored.
   - The page records `form_attempted` and `training_viewed`.
8. **Result** — Return to admin and click **重新整理** in Dashboard.
   - Expected funnel: **1 → 1 → 1 → 1 → 1**.

### Short talk track

> SimSafe first builds a source-backed employee profile. The agent selects a relevant training scenario, but a human must approve it. After simulation, the employee receives the message in a realistic mailbox. We record only training events, never form values. The final dashboard shows the full learning journey.

### If something goes wrong

- **Mailbox button is disabled:** restart the admin with `VITE_USE_FIXTURES=false`, or use `scripts/demo.sh`.
- **Email does not appear:** keep the admin tab open, reopen the mailbox from admin, and wait for the delivery message.
- **Popup is blocked:** allow local popups for `127.0.0.1`.
- **Employee already exists:** select E001 and continue; duplicate imports are safely skipped.

## Part 2 — Slide plan

### Slide 1 — The problem

**On slide**
- Real phishing is personal.
- Most training is generic.
- Generic training is easy to ignore.

**Say**
> Employees need relevant practice, but personalization must be safe and controlled.

### Slide 2 — Our solution

**On slide**
- Source-backed employee profile
- AI-assisted scenario selection
- Human approval before delivery
- Safe landing page and clear training

**Say**
> SimSafe creates relevant exercises while keeping a human in control.

### Slide 3 — How it works

**On slide**

```text
Employee data → Profile → Generate → Human review → Simulate
              → Mailbox → Safe landing → Training dashboard
```

**Say**
> The system turns approved employee data into a complete and measurable training flow.

### Slide 4 — Safety by design

**On slide**
- Authorized or fictional data only
- Sources and confidence are visible
- No real passwords, MFA codes, or financial data
- Dummy form values are never transmitted
- Email HTML runs in a sandbox
- Duplicate events do not increase results

**Say**
> Safety is part of every step, not an extra check at the end.

### Slide 5 — Architecture

**On slide**
- React admin console
- Go API and controlled landing page
- SQLite for campaigns, mailbox messages, and events
- Deterministic agent fixtures for this demo
- Optional OpenAI agent mode

**Say**
> The demo is repeatable, but the architecture can also support live agent generation.

### Slide 6 — Live demo

Use the eight steps in **Part 1**. Keep the admin and mailbox windows side by side.

**Say before the demo**
> I will show the full journey from employee data to measurable training results.

### Slide 7 — Result and next step

**On slide**
- Relevant training
- Human-controlled delivery
- Privacy-safe measurement
- Next: authentication and durable mailbox retrieval

**Say**
> SimSafe makes security training more realistic without turning personalization into a privacy risk.
