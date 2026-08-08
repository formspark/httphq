import { test, expect } from "@playwright/test";

test.describe("Home screen", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test.describe("Page", () => {
    test("the title states what the product does", async ({ page }) => {
      await expect(page).toHaveTitle(
        "httphq: inspect HTTP requests in real time",
      );
    });

    test("the hero copy renders", async ({ page }) => {
      await expect(
        page.getByRole("heading", { name: /Inspect HTTP requests/ }),
      ).toBeVisible();
    });

    // The landing page is the one surface a visitor judges the product on
    // before trusting it with traffic, so it stays a pure document.
    test("the page ships no javascript", async ({ page }) => {
      await expect(page.locator("script")).toHaveCount(0);
    });
  });

  test.describe("Creating an endpoint", () => {
    test("the create button is visible", async ({ page }) => {
      await expect(
        page.locator('button[data-test="create-endpoint"]'),
      ).toBeVisible();
    });

    test("the create button lands on a working endpoint screen", async ({
      page,
    }) => {
      await page.locator('button[data-test="create-endpoint"]').click();
      await expect(page).toHaveURL(/\/[a-z0-9-]+$/);
      await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
    });

    // The facts a visitor needs before pointing live traffic at a public URL,
    // stated where the decision is made rather than after it.
    test("the retention and visibility terms are stated beside the button", async ({
      page,
    }) => {
      await expect(
        page.getByText("Requests are deleted after 4 hours"),
      ).toBeVisible();
      await expect(
        page.getByText("anyone with the URL can read them"),
      ).toBeVisible();
    });
  });

  test.describe("Supporting sections", () => {
    test("the use cases section is visible", async ({ page }) => {
      const section = page.locator('[data-test="use-cases"]');
      await expect(section).toBeVisible();
      await expect(section).toContainText("Test webhooks");
      await expect(section).toContainText("Inspect payloads");
    });

    test("the example capture shows a rendered request", async ({ page }) => {
      const example = page.locator('[data-test="example-capture"]');
      await expect(example).toBeVisible();
      await expect(example).toContainText("POST");
      await expect(example).toContainText("content-type");
      await expect(example).toContainText("payment_intent.succeeded");
    });

    test("the footer links to contact and to the repository", async ({
      page,
    }) => {
      await expect(
        page.getByRole("link", { name: "Send us feedback" }),
      ).toHaveAttribute("href", "/contact");
      await expect(
        page.getByRole("link", { name: "formspark/httphq" }),
      ).toHaveAttribute("href", "https://github.com/formspark/httphq");
    });
  });
});
