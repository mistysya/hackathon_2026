import { defineConfig } from "@playwright/test";

// Opt in to an already-running API (for example, Docker) instead of Go on the host.
// Use a disposable database: the tests import employees and create campaigns.
const externalAPI = process.env.E2E_API_URL;
const apiURL = externalAPI || "http://127.0.0.1:18080";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  workers: 1,
  use: { trace: "retain-on-failure" },
  projects: [
    { name: "real-api", testMatch: "demo.spec.ts", use: { baseURL: "http://127.0.0.1:15173" } },
    { name: "fixtures", testMatch: "fixtures.spec.ts", use: { baseURL: "http://127.0.0.1:15174" } },
    { name: "mailbox", testMatch: "mailbox.spec.ts", use: { baseURL: "http://127.0.0.1:15173" } },
  ],
  webServer: [
    ...(!externalAPI ? [{
      command: "cd ../backend && go run ./cmd/api",
      env: { HTTP_ADDR: "127.0.0.1:18080", DATABASE_DSN: "file:e2e?mode=memory&cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)" },
      url: "http://127.0.0.1:18080/employees",
      timeout: 120000,
    }] : []),
    {
      command: "npm run dev -- --host 127.0.0.1 --port 15173 --strictPort",
      env: { VITE_USE_FIXTURES: "false", API_PROXY_TARGET: apiURL, VITE_MAILBOX_URL: "http://127.0.0.1:14173/" },
      url: "http://127.0.0.1:15173",
    },
    {
      command: "npm run dev -- --host 127.0.0.1 --port 15174 --strictPort",
      env: { VITE_USE_FIXTURES: "true" },
      url: "http://127.0.0.1:15174",
    },
    {
      command: "python3 -m http.server 14173 --bind 127.0.0.1 --directory ../presentation",
      url: "http://127.0.0.1:14173",
    },
  ],
});
