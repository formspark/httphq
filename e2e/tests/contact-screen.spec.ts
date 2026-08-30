import { test, expect } from "@playwright/test";

test.describe("Contact screen", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/contact");
  });

  test.describe("Page", () => {
    test("the title names the page", async ({ page }) => {
      await expect(page).toHaveTitle("Contact | httphq");
    });

    test("the page ships no javascript", async ({ page }) => {
      await expect(page.locator("script")).toHaveCount(0);
    });
  });

  test.describe("Form", () => {
    test("a filled form is submittable", async ({ page }) => {
      await page.locator('input[name="name"]').fill("John Doe");
      await page.locator('input[name="email"]').fill("john@doe.test");
      await page.locator('textarea[name="message"]').fill("Hello, World!");
      await expect(page.getByTestId("send-form")).toBeEnabled();
    });

    test("every field is labelled and required", async ({ page }) => {
      for (const label of ["Name", "Email", "Message"]) {
        const field = page.getByLabel(label, { exact: true });
        await expect(field).toBeVisible();
        await expect(field).toHaveAttribute("required", "");
      }
    });

    test("an empty form does not submit", async ({ page }) => {
      await page.getByTestId("send-form").click();
      await expect(page).toHaveURL(/\/contact$/);
    });

    // The hidden checkbox is a honeypot: a real visitor never sees it, so a
    // submission that ticks it came from something filling fields blindly.
    test("the honeypot field stays out of the visible form", async ({
      page,
    }) => {
      const honeypot = page.locator('input[name="accept"]');
      await expect(honeypot).toBeHidden();
      await expect(honeypot).toHaveAttribute("tabindex", "-1");
    });
  });
});
