# Mailbox demo

## Startup and walkthrough

Run these commands from the repository root, in separate terminals:

1. Start API: `cd backend && docker compose up --build api`
2. Start admin: `cd frontend && npm install && VITE_USE_FIXTURES=false npm run dev`
3. Start mailbox: `python3 -m http.server 4173 --bind 127.0.0.1 --directory presentation`
4. In admin: import employee → enrich → generate → **開啟員工信箱 ↗**.
5. Click **核准 Campaign**, then **模擬寄送 →**.
6. Wait for **新郵件已送達員工信箱**. The generated campaign appears at the top of the inbox; open it to record `opened`.
7. Click the email CTA to open the Go landing page, submit the dummy form, then refresh the admin dashboard to see the complete funnel.

The default mailbox URL uses the admin's hostname and port `4173`; override it with `VITE_MAILBOX_URL` in `frontend/.env` if needed. Use localhost or `127.0.0.1` for the admin: cross-origin mailbox delivery only accepts local demo origins and the opening admin window.

## Modes and delivery guarantees

- The separate mailbox requires **Go API mode**. In the default fixture mode its entry is disabled with an explanation; use the admin's built-in employee view for the complete offline demo. Fixture tokens do not exist in the Go API.
- Mail is queued until the mailbox announces readiness. The admin retries unacknowledged deliveries and only displays “delivered” after the mailbox acknowledges receipt.
- You may open the mailbox after simulation, reload it, or close and reopen it. The admin replays the same campaign targets, without generating another campaign or calling simulate again. Clicking the entry while the mailbox is open focuses it instead of reloading it.
- Opening an email retries the `opened` event until the admin confirms a successful API write. Backend deduplication prevents inflated counts. Older delivered campaigns can still record events after the admin generates a new campaign; only the currently active dashboard is updated.
- The queue and window association are **in admin memory**, not persistent storage. Keep the admin tab open: reloading/closing it ends this mailbox session. This is a local demo, not a durable mail delivery service.

## Privacy and safety

The 13 background messages use synthetic addresses and bodies; no original bodies, recipient addresses, credentials, or tracking links are included. Generated campaign mail comes from the authorized admin workflow and may include its approved employee personalization. Email HTML runs in a script-disabled iframe; its CTA opens the controlled Go landing page. Dummy form values are not transmitted or stored. The standalone mailbox uses local assets only; the integrated flow communicates with the local admin and API.

## Regression tests

With Go, Node.js, Python 3 and Playwright Chromium installed:

```bash
cd frontend
npm ci
npx playwright install chromium
npm test
```

The suite starts an isolated API, fixture/API admin servers and a mailbox on port `14173`. It covers the real CTA-to-dashboard workflow, fixture-mode guard, delayed initialization, missing delivery ACKs, failed event writes, older campaigns, and mailbox replay without duplicate targets.
