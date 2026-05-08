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
      page.getByRole("heading", { name: /Capture and inspect HTTP requests/ }),
    ).toBeVisible();
  });

  test("create endpoint button is visible", async ({ page }) => {
    await expect(page.locator('button[data-test="create-endpoint"]')).toBeVisible();
  });

  test("create endpoint button redirects to the endpoint screen", async ({
    page,
  }) => {
    await page.locator('button[data-test="create-endpoint"]').click();
    await expect(page).toHaveURL(/\/[a-z0-9-]+$/);
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
  });

  test("recently created panel hides when localStorage is empty", async ({
    page,
  }) => {
    await page.evaluate(() => localStorage.clear());
    await page.reload();
    await expect(page.locator('[data-test="recent-endpoints"]')).toBeHidden();
  });

  test("recently created panel surfaces previously visited endpoints", async ({
    page,
  }) => {
    await page.evaluate(() => {
      localStorage.setItem(
        "httphq:recent-endpoints",
        JSON.stringify([
          { id: "ancient-fog-0001", createdAt: Date.now() - 60_000 },
        ]),
      );
    });
    await page.reload();
    const list = page.locator('[data-test="recent-endpoints"]');
    await expect(list).toBeVisible();
    const link = list.locator('a[data-test="recent-endpoint"]');
    await expect(link).toContainText("ancient-fog-0001");
    await expect(link).toHaveAttribute("href", "/ancient-fog-0001");
  });

  test("creating an endpoint adds it to the recent list", async ({ page }) => {
    await page.evaluate(() => localStorage.clear());
    await page.locator('button[data-test="create-endpoint"]').click();
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
    const id = await page.evaluate(() => location.pathname.slice(1));
    await page.goto("/");
    await expect(
      page.locator('[data-test="recent-endpoint"]', { hasText: id }),
    ).toBeVisible();
  });
});
