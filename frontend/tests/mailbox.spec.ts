import { test, expect, type Page } from "@playwright/test";

const csv = "employee_id,display_name,email,department,title,company\nE001,Demo User,demo.user@example.test,Engineering,Software Engineer,Demo Corp\n";

async function generateCampaign(page: Page) {
  const response = page.waitForResponse(r => r.url().endsWith("/campaigns/generate"));
  await page.getByRole("button", { name: "Generate exercise" }).click();
  return (await response).json() as Promise<{ campaignId: string }>;
}

async function prepareCampaign(page: Page) {
  await page.request.post("/api/employees/import", { headers: { "Content-Type": "text/csv" }, data: csv });
  await page.request.post("/api/employees/E001/enrich", { data: {} });
  await page.goto("/");
  return generateCampaign(page);
}

async function openMailbox(page: Page) {
  const popup = page.waitForEvent("popup");
  await page.getByRole("button", { name: "Open employee mailbox ↗" }).click();
  return popup;
}

async function simulate(page: Page) {
  await page.getByRole("button", { name: "Approve campaign", exact: true }).click();
  await page.getByRole("button", { name: "Simulate delivery →", exact: true }).click();
  await expect(page.locator("#review .status")).toHaveText("Simulated delivery");
}

async function deliver(page: Page) {
  const campaign = await prepareCampaign(page);
  const mailbox = await openMailbox(page);
  await expect(mailbox.locator(".mail-row")).toHaveCount(13);
  await simulate(page);
  await expect(mailbox.locator(".mail-row")).toHaveCount(14);
  return { mailbox, campaignId: campaign.campaignId };
}

async function report(page: Page, campaignId: string) {
  return (await page.request.get(`/api/reports/${campaignId}`)).json();
}

test("mailbox completes the real API workflow and deduplicates reopened mail", async ({ page }) => {
  const { mailbox, campaignId } = await deliver(page);
  await mailbox.locator(".mail-row").first().click();
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "0", "0", "0"]);
  const popup = mailbox.waitForEvent("popup");
  await mailbox.frameLocator("#generatedCampaignFrame").getByRole("link").click();
  const landing = await popup;
  await expect(landing.locator("#dummy-form")).toBeVisible();
  const eventBodies: unknown[] = [];
  landing.on("request", req => { if (req.method() === "POST") eventBodies.push(req.postDataJSON()); });
  await landing.locator("#demo-user").fill("DO_NOT_TRANSMIT");
  await landing.locator("button[type=submit]").click();
  await expect(landing.locator("#reveal")).toBeVisible();
  await expect.poll(async () => (await report(page, campaignId)).funnel.trainingViewed).toBe(1);
  for (const body of eventBodies) expect(Object.keys(body as object).sort()).toEqual(["eventType", "token"]);
  expect(JSON.stringify(eventBodies)).not.toContain("DO_NOT_TRANSMIT");
  await mailbox.locator("#backButton").click();
  await mailbox.locator(".mail-row").first().click();
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "1", "1", "1"]);
  expect((await report(page, campaignId)).events).toHaveLength(4);
});

test("failed opened writes are retried until acknowledged", async ({ page }) => {
  const { mailbox, campaignId } = await deliver(page);
  let attempts = 0;
  await page.route("**/api/events", async route => {
    attempts++;
    if (attempts === 1) {
      await route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: { message: "transient failure" } }) });
    } else {
      await route.continue();
    }
  });
  await mailbox.locator(".mail-row").first().click();
  await expect(page.getByRole("alert")).toContainText("transient failure");
  await expect.poll(async () => (await report(page, campaignId)).funnel.opened).toBe(1);
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "0", "0", "0"]);
  // ACK stops the retry loop, rather than just relying on backend deduplication.
  await page.waitForTimeout(2300);
  expect(attempts).toBe(2);
  expect((await report(page, campaignId)).events).toHaveLength(1);
});

test("older campaign opens are persisted without changing the active campaign", async ({ page }) => {
  const { mailbox, campaignId } = await deliver(page);
  await generateCampaign(page);
  await expect(page.locator("#review .status")).toHaveText("Pending review");
  await mailbox.locator(".mail-row").first().click();
  await expect.poll(async () => (await report(page, campaignId)).funnel.opened).toBe(1);
  await expect(page.locator("#review .status")).toHaveText("Pending review");
  await expect(page.locator(".metric")).toHaveCount(0);
});

test("slow mailbox initialization waits for readiness and confirms delivery", async ({ page, context }) => {
  await prepareCampaign(page);
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await context.route("**/app.js", async route => { await gate; await route.continue(); });
  try {
    const mailbox = await openMailbox(page);
    await simulate(page);
    await page.waitForTimeout(1600);
    await expect(page.locator(".alert.info")).toContainText("awaiting delivery confirmation");
    release();
    await expect(mailbox.locator(".mail-row")).toHaveCount(14);
    await expect(page.locator(".alert.info")).toContainText("new email was delivered");
    await page.waitForTimeout(1200);
    await expect(mailbox.locator(".mail-row")).toHaveCount(14);
  } finally {
    release();
  }
});

test("missing delivery ACK retries the same message without duplicating it", async ({ page, context }) => {
  // Drop one ACK at the receiving window to exercise actual postMessage retries.
  await context.addInitScript(() => {
    window.addEventListener("message", event => {
      if (event.data?.type === "simsafe:delivery-ack" && !(window as any).__droppedAck) {
        (window as any).__droppedAck = true;
        event.stopImmediatePropagation();
      }
    });
  });
  const { mailbox } = await deliver(page);
  await expect(page.locator(".alert.info")).toContainText("new email was delivered");
  await expect(mailbox.locator(".mail-row")).toHaveCount(14);
  expect(await page.evaluate(() => (window as any).__droppedAck)).toBe(true);
});

test("late-opened, reloaded and reopened mailboxes replay existing targets without simulating again", async ({ page }) => {
  let simulations = 0;
  page.on("request", req => { if (req.url().endsWith("/simulate")) simulations++; });
  const { campaignId } = await prepareCampaign(page);
  await simulate(page);
  const mailbox = await openMailbox(page);
  await expect(mailbox.locator(".mail-row")).toHaveCount(14);
  const messageId = await mailbox.locator(".mail-row").first().getAttribute("data-id");
  await mailbox.locator(".mail-row").first().click();
  await expect.poll(async () => (await report(page, campaignId)).funnel.opened).toBe(1);
  // Clicking the admin entry again focuses the existing view instead of clearing it.
  await page.getByRole("button", { name: "Open employee mailbox ↗" }).click();
  await expect(mailbox.locator("#messagePanel")).toBeVisible();
  await mailbox.reload();
  await expect(mailbox.locator(".mail-row")).toHaveCount(14);
  await expect(mailbox.locator(".mail-row").first()).toHaveAttribute("data-id", messageId!);
  await mailbox.close();
  const reopened = await openMailbox(page);
  await expect(reopened.locator(".mail-row")).toHaveCount(14);
  await expect(reopened.locator(".mail-row").first()).toHaveAttribute("data-id", messageId!);
  expect(simulations).toBe(1);
  expect((await report(page, campaignId)).targetCount).toBe(1);
});
