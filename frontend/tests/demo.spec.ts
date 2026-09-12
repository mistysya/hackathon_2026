import { test, expect } from "@playwright/test";

const csv = "employee_id,display_name,email,department,title,company\nE001,Demo User,demo.user@example.test,Engineering,Software Engineer,Demo Corp\n";

test("real demo: CSV → profile → review → email CTA → landing → dashboard", async ({ page, context }, testInfo) => {
  const errors: string[] = [];
  context.on("page", p => p.on("pageerror", error => errors.push(error.message)));
  page.on("pageerror", error => errors.push(error.message));
  await page.goto("/");
  await expect(page.getByText("Go API integration mode")).toBeVisible();
  await page.locator('input[type="file"]').setInputFiles({ name: "employees.csv", mimeType: "text/csv", buffer: Buffer.from(csv) });
  await expect(page.locator(".profile-title")).toContainText("Demo User");
  await page.getByRole("button", { name: /^(建立 Profile|重新 Enrich)$/ }).click();
  await expect(page.locator(".facts")).toContainText("mock");
  const generated = page.waitForResponse(r => r.url().endsWith("/campaigns/generate"));
  await page.getByRole("button", { name: "生成安全演練" }).click();
  const campaign = await (await generated).json();
  await expect(page.frameLocator('iframe[title="安全信件預覽"]').getByText("Hello Demo User,")).toBeVisible();
  expect((await page.request.post(`/api/campaigns/${campaign.campaignId}/simulate`)).status()).toBe(409);
  await page.getByRole("button", { name: "核准 Campaign", exact: true }).click();
  await expect(page.locator("#review .status")).toHaveText("已核准");
  await page.getByRole("button", { name: "模擬寄送 →", exact: true }).click();
  await expect(page.locator(".metric b")).toHaveText(["1", "0", "0", "0", "0"]);
  await page.getByRole("button", { name: "開啟模擬信件", exact: true }).click();
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "0", "0", "0"]);
  const landingLink = page.locator(".mail-preview > .actions a");
  const landingUrl = await landingLink.getAttribute("href");
  const popup = page.waitForEvent("popup");
  await page.frameLocator('iframe[title="員工信件視角"]').getByRole("link").click();
  const landing = await popup;
  await expect(landing.locator("#dummy-form")).toBeVisible();
  const report = async () => (await page.request.get(`/api/reports/${campaign.campaignId}`)).json();
  await expect.poll(async () => (await report()).funnel.clicked).toBe(1);
  const eventBodies: unknown[] = [];
  landing.on("request", req => { if (req.method() === "POST") eventBodies.push(req.postDataJSON()); });
  await landing.locator("#demo-user").fill("PRIVATE_DUMMY_VALUE");
  await landing.locator("#demo-note").fill("DO_NOT_TRANSMIT");
  await landing.locator('button[type="submit"]').click();
  await expect(landing.locator("#reveal")).toBeVisible();
  await expect.poll(async () => (await report()).funnel.trainingViewed).toBe(1);
  expect(eventBodies).toHaveLength(2);
  expect((await page.request.post("/api/events", { data: { token: landingUrl!.split("/").pop(), eventType: "clicked", password: "must-not-be-accepted" } })).status()).toBe(400);
  for (const body of eventBodies) expect(Object.keys(body as object).sort()).toEqual(["eventType", "token"]);
  expect(JSON.stringify(eventBodies)).not.toContain("PRIVATE_DUMMY_VALUE");
  await landing.reload();
  await landing.locator('button[type="submit"]').click();
  await expect(landing.locator("#reveal")).toBeVisible();
  await page.getByRole("button", { name: "重新整理", exact: true }).click();
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "1", "1", "1"]);
  await expect(page.locator(".timeline .event")).toHaveCount(4);
  expect((await report()).events).toHaveLength(4);
  expect(landingUrl).toMatch(/^\/landing\/[a-f0-9]{32}$/);
  expect(errors).toEqual([]);
  await page.screenshot({ path: testInfo.outputPath("dashboard.png"), fullPage: true });
});

test("reject and invalid requests preserve lifecycle and error envelopes", async ({ request }) => {
  await request.post("/api/employees/import", { headers: { "Content-Type": "text/csv" }, data: csv });
  await request.post("/api/employees/E001/enrich", { data: {} });
  const generated = await request.post("/api/campaigns/generate", { data: { employeeId: "E001" } });
  expect(generated.status()).toBe(201);
  const { campaignId } = await generated.json();
  expect((await request.post(`/api/campaigns/${campaignId}/approve`, { data: { approvedBy: " " } })).status()).toBe(400);
  expect((await request.post(`/api/campaigns/${campaignId}/reject`, { data: { reason: "Needs revision" } })).status()).toBe(200);
  for (const action of ["approve", "simulate"]) {
    const response = await request.post(`/api/campaigns/${campaignId}/${action}`, { data: { approvedBy: "hr@example.test" } });
    expect(response.status()).toBe(409);
    expect((await response.json()).error.requestId).toBeTruthy();
  }
  const result = await request.post("/api/employees/import", { headers: { "Content-Type": "text/csv" }, data: csv });
  expect(await result.json()).toMatchObject({ imported: 0, skipped: 1 });
  const approved = await (await request.post("/api/campaigns/generate", { data: { employeeId: "E001" } })).json();
  await request.post(`/api/campaigns/${approved.campaignId}/approve`, { data: { approvedBy: "hr@example.test" } });
  const attempts = await Promise.all(Array.from({ length: 8 }, () => request.post(`/api/campaigns/${approved.campaignId}/simulate`)));
  expect(attempts.map(response => response.status()).sort()).toEqual([200, 409, 409, 409, 409, 409, 409, 409]);
  for (const path of ["/employees/missing", "/campaigns/missing", "/reports/missing", "/landing/missing"]) {
    const response = await request.get(`/api${path}`);
    expect(response.status()).toBe(404);
    expect((await response.json()).error.requestId).toBeTruthy();
  }
});
