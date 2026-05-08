import { test, expect } from "@playwright/test";

test.describe("Home screen", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("title should be correct", async ({ page }) => {
    await expect(page).toHaveTitle("httphq");
  });

  test("create endpoint button should be visible", async ({ page }) => {
    await expect(page.locator('button[data-test="create-endpoint"]')).toBeVisible();
  });

  test("create endpoint button should redirect to the endpoint screen", async ({ page }) => {
    await page.locator('button[data-test="create-endpoint"]').click();
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
  });
});
