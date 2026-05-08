import { test, expect, type Page, type APIRequestContext } from "@playwright/test";

const randomId = () => Math.random().toString(36).slice(2, 7);

const post = async (
  request: APIRequestContext,
  url: string,
  init: { data?: string | object; headers?: Record<string, string> } = {},
) => request.post(url, init);

test.describe("Endpoint screen", () => {
  let testId: string;
  let endpointPath: string;
  let endpointUrl: string;

  test.beforeEach(async ({ page }) => {
    testId = randomId();
    endpointPath = `/to/${testId}`;
    endpointUrl = `http://localhost:8080${endpointPath}`;
    await page.goto(`/${testId}`);
  });

  test("title should be correct", async ({ page }) => {
    await expect(page).toHaveTitle(`${testId} | httphq`);
  });

  test("endpoint URL should be visible", async ({ page }) => {
    await expect(page.locator('[data-test="endpoint-url"]')).toContainText(endpointUrl);
  });

  test("cURL example should be visible", async ({ page }) => {
    await expect(page.locator('[data-test="curl-example"]')).toContainText(
      `curl -X POST -d 'Hello, World!' ${endpointUrl}`,
    );
  });

  test("cURL example should be possible to send a request", async ({ page }) => {
    await page.locator('[data-test="curl-send"]').click();
    await expect(page.locator('[data-test="requests"]')).not.toContainText(
      "Waiting for requests",
    );
  });

  test.describe("Requests", () => {
    test("should show a waiting message if no requests are found", async ({ page }) => {
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    test("should not show a waiting message if some requests are found", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      await expect(page.locator('[data-test="requests"]')).not.toContainText(
        "Waiting for requests",
      );
    });

    test("should display new requests in real-time", async ({ page, request }) => {
      const body = "Hello, World!";
      await post(request, endpointUrl, { data: body });
      await expect(page.locator('[data-test="requests"]')).toContainText(body);
    });

    test("should display the request details", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      const details = page.locator('[data-test="request-details"]');
      await expect(details).toContainText(/now|seconds? ago/);
      await expect(details).toContainText("127.0.0.1");
      await expect(details).toContainText("POST");
      await expect(details).toContainText(endpointPath);
    });

    test("should display the request headers", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      await expect(page.locator('[data-test="request-headers"]')).toContainText(
        "Content-Type",
      );
    });

    test("should display the request body", async ({ page, request }) => {
      const body = "Hello, World!";
      await post(request, endpointUrl, { data: body });
      await expect(page.locator('[data-test="request-body"]')).toContainText(body);
    });

    test("should be possible to filter based on the request body", async ({
      page,
      request,
    }) => {
      const body = "Hello, World!";
      await post(request, endpointUrl, { data: body });
      const requests = page.locator('[data-test="requests"]');
      const results = page.locator('[data-test="search-results"]');
      const search = page.locator('[data-test="search-input"]');

      await expect(requests).toContainText(body);

      await search.fill("Hello");
      await expect(results).toContainText("1 result");
      await expect(requests).toContainText(body);

      await search.fill("Test");
      await expect(results).toContainText("0 results");
      await expect(requests).not.toContainText(body);
    });

    test("should be possible to filter based on the request headers", async ({
      page,
      request,
    }) => {
      const headerKey = "A-Test";
      const headerValue = "Hello, World!";
      await post(request, endpointUrl, {
        data: "",
        headers: { [headerKey]: headerValue },
      });

      const requests = page.locator('[data-test="requests"]');
      const results = page.locator('[data-test="search-results"]');
      const search = page.locator('[data-test="search-input"]');

      await expect(requests).toContainText(headerKey);
      await expect(requests).toContainText(headerValue);

      // Positive key search
      await search.fill("A-");
      await expect(results).toContainText("1 result");
      await expect(requests).toContainText(headerKey);
      await expect(requests).toContainText(headerValue);

      // Negative key search
      await search.fill("B-");
      await expect(results).toContainText("0 results");
      await expect(requests).not.toContainText(headerKey);
      await expect(requests).not.toContainText(headerValue);

      // Positive value search
      await search.fill("Hello");
      await expect(results).toContainText("1 result");
      await expect(requests).toContainText(headerKey);
      await expect(requests).toContainText(headerValue);

      // Negative value search
      await search.fill("Not");
      await expect(results).toContainText("0 results");
      await expect(requests).not.toContainText(headerKey);
      await expect(requests).not.toContainText(headerValue);
    });

    test("should show a deletion disclaimer", async ({ page }) => {
      await expect(page.getByText("⚠️️️ Requests are deleted after 4 hours")).toBeVisible();
    });

    test("should be possible to delete all requests", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      await post(request, endpointUrl, { data: "Hello, World!" });
      await page.locator('[data-test="delete-requests"]').click();
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    test("should be possible to delete a specific request", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      const response = await request.post(endpointUrl, { data: { message: "Hello, World!" } });
      const uuid = response.headers()["httphq-request-uuid"];
      const card = page.locator(`#request-${uuid}`);
      await expect(card).toBeAttached();
      await card.locator('[data-test="delete-request"]').click({ force: true });
      await expect(card).not.toBeAttached();
    });
  });
});
