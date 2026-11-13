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

  // The narrowest phone, under touch. iOS zooms the page into any field set
  // under 16px when it takes focus, and a control under 44px is missed by a
  // thumb.
  test.describe("Phone", () => {
    test.use({
      viewport: { width: 320, height: 640 },
      isMobile: true,
      hasTouch: true,
    });

    test("every field is set at 16px and reaches 44px", async ({ page }) => {
      for (const label of ["Name", "Email", "Message"]) {
        const field = page.getByLabel(label, { exact: true });
        await expect(field).toHaveCSS("font-size", "16px");
        const box = await field.boundingBox();
        expect(box?.height).toBeGreaterThanOrEqual(44);
      }
    });

    test("the send button keeps its label on one line at 44px", async ({
      page,
    }) => {
      const box = await page.getByTestId("send-form").boundingBox();
      expect(box?.height).toBeGreaterThanOrEqual(44);
      expect(box?.height).toBeLessThan(50);
    });
  });
});
