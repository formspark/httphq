import { test, expect } from "@playwright/test";

test.describe("Contact screen", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/contact");
  });

  test("title should be correct", async ({ page }) => {
    await expect(page).toHaveTitle("Contact | httphq");
  });

  test("form should be functional", async ({ page }) => {
    await page.locator('input[name="name"]').fill("John Doe");
    await page.locator('input[name="email"]').fill("john@doe.test");
    await page.locator('textarea[name="message"]').fill("Hello, World!");
    await expect(page.locator('button[data-test="send-form"]')).toBeEnabled();
  });
});
