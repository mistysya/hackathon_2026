import { test, expect } from "@playwright/test";

test("offline fixture demo completes and deduplicates interactions", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  page.on("request", req => { if (req.url().includes("/api/")) errors.push(`Unexpected API call: ${req.url()}`); });
  await page.goto("/");
  await expect(page.getByText("Fixture demo mode")).toBeVisible();
  await expect(page.getByRole("button", { name: "Open employee mailbox ↗" })).toBeDisabled();
  await expect(page.getByText("Employee mailbox requires the Go API. Use the employee view below for an offline demo.")).toBeVisible();
  await page.getByRole("button", { name: "Generate exercise" }).click();
  await page.getByRole("button", { name: "Approve campaign", exact: true }).click();
  await page.getByRole("button", { name: "Simulate delivery →", exact: true }).click();
  await page.getByRole("button", { name: "Open simulated email", exact: true }).click();
  await page.getByRole("button", { name: "Simulate CTA click", exact: true }).click();
  await page.getByLabel("Dummy email").fill("NOT_SENT");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByRole("heading", { name: "This is a controlled security exercise" })).toBeVisible();
  await page.getByRole("button", { name: "Simulate CTA click", exact: true }).click();
  await page.locator('button[type="submit"]').click();
  await expect(page.locator(".metric b")).toHaveText(["1", "1", "1", "1", "1"]);
  await expect(page.locator(".timeline .event")).toHaveCount(4);
  expect(errors).toEqual([]);
});

test("fixture CSV handles quoted commas, multiline fields, duplicates and invalid input atomically", async ({ page }) => {
  await page.goto("/");
  const header = "employee_id,display_name,email,department,title,company\n";
  const upload = async (csv: string) => {
    await page.locator('input[type="file"]').setInputFiles({ name: "employees.csv", mimeType: "text/csv", buffer: Buffer.from(csv) });
  };
  await upload(header + 'E002,"Chen, Alex",alex@example.test,Product,"Product\nManager",Demo Corp\n');
  await expect(page.getByText("Import complete: 1 added, 0 skipped.")).toBeVisible();
  await page.getByRole("button", { name: /Chen, Alex/ }).click();
  await expect(page.locator(".profile-title h3")).toHaveText("Chen, Alex");
  await expect(page.locator(".profile-title")).toContainText("Product\nManager");
  await upload(header + 'E002,Duplicate,d@example.test,,,,\n');
  await expect(page.getByRole("alert")).toContainText("CSV");
  await upload(header + 'E002,Duplicate,d@example.test,,,\nE003,,missing@example.test,,,\n');
  await expect(page.getByRole("alert")).toContainText("duplicate employee_id");
  await expect(page.getByRole("alert")).toContainText("missing display_name");
  await upload(header + 'E004,Valid,valid@example.test,,,\nE005,"unclosed quote');
  await expect(page.getByRole("alert")).toContainText("CSV");
  await expect(page.locator(".employee-row")).toHaveCount(2);
});
