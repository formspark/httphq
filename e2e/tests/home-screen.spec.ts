import { test, expect } from "@playwright/test";

test.describe("Home screen", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("title should be correct", async ({ page }) => {
    await expect(page).toHaveTitle("httphq");
  });

  test("page renders the hero copy", async ({ page }) => {
    await expect(
      page.getByRole("heading", { name: /Inspect HTTP requests/ }),
    ).toBeVisible();
  });

  test("create endpoint button is visible", async ({ page }) => {
    await expect(
      page.locator('button[data-test="create-endpoint"]'),
    ).toBeVisible();
  });

  test("create endpoint button redirects to the endpoint screen", async ({
    page,
  }) => {
    await page.locator('button[data-test="create-endpoint"]').click();
    await expect(page).toHaveURL(/\/[a-z0-9-]+$/);
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
  });

  test("common use cases section is visible", async ({ page }) => {
    const section = page.locator('[data-test="use-cases"]');
    await expect(section).toBeVisible();
    await expect(section).toContainText("Test webhooks");
    await expect(section).toContainText("Inspect form payloads");
  });
});
