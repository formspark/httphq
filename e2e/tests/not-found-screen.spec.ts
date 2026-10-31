import { test, expect } from "@playwright/test";

test.describe("Not found screen", () => {
  test("a missed address answers 404 with the page", async ({ page }) => {
    const response = await page.goto("/no/such/page");

    expect(response?.status()).toBe(404);
    await expect(page).toHaveTitle("Page not found | httphq");
    await expect(
      page.getByRole("heading", { level: 1, name: "There is no page here" }),
    ).toBeVisible();
    await expect(page.getByTestId("requested-path")).toHaveText(
      "/no/such/page",
    );
  });

  // A guessed page and a mistyped ID reach the endpoint route rather than the
  // fallthrough, and read the same.
  test("a refused endpoint ID shows the page", async ({ page }) => {
    const response = await page.goto("/Quiet-Sunset-4821");

    expect(response?.status()).toBe(404);
    await expect(page.getByTestId("requested-path")).toHaveText(
      "/Quiet-Sunset-4821",
    );
  });

  test("the page ships no javascript", async ({ page }) => {
    await page.goto("/no/such/page");
    await expect(page.locator("script")).toHaveCount(0);
  });

  test("the primary action creates an endpoint", async ({ page }) => {
    await page.goto("/no/such/page");
    await page.getByTestId("create-endpoint").click();
    await expect(page).toHaveURL(/\/[a-z0-9]+(-[a-z0-9]+)+$/);
  });
});
