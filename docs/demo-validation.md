# Demo validation report

## Baseline and scope

- Repository baseline: `origin/main` at `8a48472`.
- Validation worktree: `/Users/angelica_wu/hackathon_2026-demo`, branch `fix/demo-validation`.
- The existing `sea_hack` feature checkout and its untracked files were left untouched.
- Main's Go executable registered no business routes. Existing tests still passed, and the frontend's default fixture mode concealed the integration gap.
- Fast-forwarded this validation branch to existing Role B integration commit `d05d9b5`, which includes the member A services and campaign lifecycle integration. Additional fixes below are in this worktree.

## Reproduced problems and fixes

| Problem | Fix / regression coverage |
|---|---|
| Real API returned 404 because routes were not wired on main | Integrated existing employee/profile/campaign/landing/event/report route composition; browser suite launches the actual executable, not a test-only router. |
| Landing HTML displayed but dashboard click count remained zero | Removed `printf "%q"` before Go's context-aware JavaScript template escaping. It had double-quoted both token and endpoint. Added rendered-JS assertions and actual browser event tests. |
| Email CTA navigated inside a script-disabled sandbox, breaking landing events | Employee email links open a new tab with popup sandbox escape; email itself still cannot run scripts. Review previews are inert. Browser test clicks the actual embedded email link. |
| Concurrent simulations intermittently returned HTTP 500 | Make the approved→simulated conditional update the first transactional operation, avoiding read-to-write SQLite lock upgrade races. Exactly one target is created; other callers get 409. Tested with 8 concurrent HTTP requests and 16 concurrent repository calls against file-backed WAL SQLite. |
| Fixture CSV split fields on every comma/newline | Use a CSV parser; support quoted commas, escaped quotes, multiline fields, required fields, duplicates, and atomic rejection of malformed CSV. |
| Fixture validation diverged on missing employees and blank approval/rejection inputs | Correct 404 and required-field checks; reject unknown fixture event types. |
| Overlapping UI actions could overwrite in-flight employee/campaign state | Disable data-changing controls during operations. |
| Downloadable CSV omitted E001's prepared evidence fixture | Include E001 and E002; add a ready-to-upload CSV under `docs/fixtures/`. |
| Demo setup was scattered and could silently use frontend mocks | Add localhost-only `scripts/demo.sh`, explicitly force real API mode, document full walkthrough and fallback mode. |

## Executed checks

Environment: macOS arm64, Go **1.27.1**, Node **26.8.2**, npm **11.19.1**, Playwright **1.63.0** / Chromium.

- `go test -race -count=1 -coverprofile=... ./...` — passed.
- `go test -json ./...` — **110 tests/subtests passed**, no failures.
- `go vet ./...` — passed.
- `go test -race -count=10 ./internal/roleb` — passed, including concurrent SQLite simulations.
- Go aggregate statement coverage: **76.6%** using default per-package instrumentation. This is not 100% function or branch coverage; the separate browser suite also executes the actual `cmd/api` entry point.
- Clean `npm ci` — passed, **0 reported dependency vulnerabilities** at validation time.
- `npm run build` — passed in both default fixture and `VITE_USE_FIXTURES=false` modes.
- Playwright regression suite — **4 scenarios, repeated three times: 12/12 passed**.
- Launcher smoke test — real Go API reached through Vite proxy; browser confirms integration mode; shutdown releases both ports.
- `bash -n scripts/demo.sh` and `git diff --check` — passed.

### Browser/API assertions

- CSV import, employee selection, enrichment, visible profile facts and email preview.
- Generation, approval, rejection, missing-resource error envelopes, and approval-before-simulation enforcement.
- Intermediate funnels `1/0/0/0/0` and `1/1/0/0/0`, followed by final `1/1/1/1/1`.
- Actual email CTA opens working Go landing page; submission reveals training.
- Event payloads contain only `token` and `eventType`; extra sensitive fields are rejected by the API.
- Reloading/re-submitting the landing does not inflate the four timeline events.
- Concurrent simulation requests preserve the lifecycle and target count.
- Frontend-only fallback completes without API calls.
- Quoted/multiline fixture CSV, duplicates, missing required values, and malformed-file atomicity.
- No uncaught JavaScript page errors on the tested happy paths.

## Limitations / not certified

- Docker commands could not be exercised: the local Docker daemon is not running. Native execution is verified.
- An optional GitHub Actions template is provided at `docs/ci/demo-regression.yml`; it is not enabled or verified on GitHub. The publishing token lacks `workflow` scope. A maintainer with workflow permissions can copy it to `.github/workflows/test.yml`.
- Browser execution was tested in Chromium, not every browser or device.
- Live LLM generation, external search, real email delivery, authentication, and production deployment are outside this fixture-backed MVP and were not tested.
- Source URLs are deliberately illustrative fixture URLs; they are not live evidence-fetch integrations.
- Active campaign selection is in console memory; a page reload clears that view. SQLite data remains. Keep the console tab open during the demo, or generate a new campaign to repeat.
- No finite test suite can certify that every function is free of all possible errors. The documented demo path and listed negative/concurrency cases are verified.

See [README](../README.md) for startup and the three-minute demo script.
