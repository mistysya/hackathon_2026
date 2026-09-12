import { test, expect } from "@playwright/test";

test("offline fixture demo completes and deduplicates interactions", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", error => errors.push(error.message));
  page.on("request", req => { if (req.url().includes("/api/")) errors.push(`Unexpected API call: ${req.url()}`); });
  await page.goto("/");
  await expect(page.getByText("Fixture demo mode")).toBeVisible();
  await page.getByRole("button", { name: "生成安全演練" }).click();
  await page.getByRole("button", { name: "核准 Campaign", exact: true }).click();
  await page.getByRole("button", { name: "模擬寄送 →", exact: true }).click();
  await page.getByRole("button", { name: "開啟模擬信件", exact: true }).click();
  await page.getByRole("button", { name: "模擬點擊 CTA", exact: true }).click();
  await page.getByLabel("Dummy email").fill("NOT_SENT");
  await page.locator('button[type="submit"]').click();
  await expect(page.getByRole("heading", { name: "這是一場受控資安演練" })).toBeVisible();
  await page.getByRole("button", { name: "模擬點擊 CTA", exact: true }).click();
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
  await expect(page.getByText("匯入完成：新增 1 位、略過 0 位。")).toBeVisible();
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
