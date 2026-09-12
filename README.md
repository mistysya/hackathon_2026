# SimSafe — safe phishing-training demo

An authorized security-training MVP: employee CSV → fixture-backed profile → controlled scenario → human approval → simulated email → dummy landing page → event dashboard.

## Run the integrated demo

Requirements: **Go 1.27.1+**, **Node.js 22.12+** (or a newer supported release), npm, curl. No API keys, email service, or Docker required. Dependency downloads need internet on the first run; enrichment and scenario generation use embedded deterministic fixtures, not live AI/search.

```sh
# From the repository root
npm --prefix frontend ci
bash scripts/demo.sh
```

Open **http://127.0.0.1:5173**. The sidebar must say **Go API integration mode**. This launcher explicitly selects the real Go API, binds both servers to localhost, and stops both on Ctrl-C. Ports 5173 and 8080 must be free.

### Three-minute walkthrough

1. Upload `docs/fixtures/employees.csv` (or use **下載 CSV 範本** → **上傳 CSV**).
2. Select **Demo User / E001**, then **建立 Profile** (or **重新 Enrich**).
3. Show the public facts marked **mock**, source links, confidence, and scenario reason.
4. Click **生成安全演練**. Review the email, landing preview, and safety checks.
5. Click **核准 Campaign**, then **模擬寄送 →**. Unapproved/rejected campaigns cannot be simulated.
6. Click **開啟模擬信件**, then the CTA inside the email (or **開啟受控 Landing ↗**). Allow the local popup if prompted.
7. Submit the dummy form. The training explanation appears. Only token and event type are sent; no form values are transmitted.
8. Return to the console and click **重新整理** in Dashboard. Expected funnel: **1 → 1 → 1 → 1 → 1**, with four timeline events. Repeated landing visits/submissions do not inflate counts.

For a rejection demo, generate another campaign, click **拒絕**, enter a reason, and observe that sending is unavailable.

### Persistence and repeat demos

SQLite persists in `backend/app.db` by default. Re-importing employees skips duplicates; simply select E001 and generate a new campaign to repeat. The console's active campaign/target selection is in browser memory: keep that tab open during the walkthrough. Reloading the console clears its current view but does not delete backend records.

E001 has the prepared public-evidence fixture. Other employee IDs intentionally have no public facts in the Go backend; they can still use the safe default training scenario. This is not a live OSINT/LLM demo.

### Offline frontend-only fallback

```sh
cd frontend
VITE_USE_FIXTURES=true npm run dev -- --host 127.0.0.1
```

This mode uses in-memory mock state and requires no backend. Use **模擬點擊 CTA**, then submit the inline dummy form. Reload resets all fixture state. Do not present this mode as a real API integration.

## Verification

```sh
cd backend
go test -race ./...
go vet ./...
cd ../frontend
npm ci
npm run build
npx playwright install chromium
npm test
# Optional: repeat the browser/API regression suite three times
npm run test:repeat
```

Playwright launches the actual Go executable with an isolated in-memory SQLite database and two Vite servers. It uses ports **18080, 15173, 15174**, never the demo database. Tests cover real and fixture workflows, the actual email CTA, privacy of event payloads, event deduplication, CSV errors, missing resources, approval/rejection gates, and concurrent simulation requests. Failure traces are saved under `frontend/test-results/`.

An optional GitHub Actions template is available at `docs/ci/demo-regression.yml`. A maintainer with workflow permissions can enable it by copying it to `.github/workflows/test.yml`; CI is not enabled by this PR.

More details: [validation report](docs/demo-validation.md), [Docker/Colima validation](docs/docker-validation.md), [API contract](docs/api-contract.md), [backend setup](backend/README.md), [frontend setup](frontend/README.md).

## Safety / scope

Use fictional or explicitly authorized data only. The MVP has no authentication and must not be exposed publicly. It simulates delivery, uses clearly identified test branding, and must never collect real passwords, MFA codes, or financial data. Live search, live LLM generation, and real email delivery are not implemented or tested.
