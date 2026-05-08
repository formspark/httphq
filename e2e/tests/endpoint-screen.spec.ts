import {
  test,
  expect,
  type APIRequestContext,
} from "@playwright/test";

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
    // Wait for Alpine to mount and the empty state to render so subsequent
    // assertions don't race with initialization.
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
  });

  test("title is the endpoint id", async ({ page }) => {
    await expect(page).toHaveTitle(`${testId} | httphq`);
  });

  test("endpoint URL is shown with copy button", async ({ page }) => {
    await expect(page.locator('[data-test="endpoint-url"]')).toContainText(
      endpointUrl,
    );
    await expect(page.locator('[data-test="copy-url"]')).toBeVisible();
  });

  test("copy-url button writes the URL to the clipboard", async ({ page }) => {
    await page.locator('[data-test="copy-url"]').click();
    const clipboard = await page.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboard).toBe(endpointUrl);
    await expect(
      page.locator('[data-test="copy-url-label"]'),
    ).toContainText("Copied!");
  });

  test("disclaimer about 4-hour retention is visible", async ({ page }) => {
    await expect(
      page.getByText("Requests are deleted after 4 hours"),
    ).toBeVisible();
  });

  test.describe("Send a test request panel", () => {
    test("submitting the panel produces a captured request", async ({
      page,
    }) => {
      await page.locator('[data-test="send-toggle"]').click();
      await page
        .locator('[data-test="send-method"]')
        .selectOption("PUT");
      await page
        .locator('[data-test="send-headers"]')
        .fill("X-Source: panel\nContent-Type: application/json");
      await page
        .locator('[data-test="send-body"]')
        .fill('{"hello":"panel"}');
      await page.locator('[data-test="send-submit"]').click();

      const card = page.locator('[data-test="request"]').first();
      await expect(card).toContainText("PUT");
      await expect(card.locator('[data-test="request-headers"]')).toContainText(
        "X-Source",
      );
      await expect(card.locator('[data-test="request-body"]')).toContainText(
        "panel",
      );
    });
  });

  test.describe("Requests list", () => {
    test("shows the empty state when no requests exist", async ({ page }) => {
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    test("does not show the empty state once requests arrive", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      await expect(page.locator('[data-test="requests"]')).not.toContainText(
        "Waiting for requests",
      );
    });

    test("displays new requests in real-time over WebSocket", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, { data: "Real-time-payload" });
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Real-time-payload",
      );
    });

    test("renders request details, headers and body", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, {
        data: "Hello, World!",
        headers: { "Content-Type": "text/plain" },
      });
      const card = page.locator('[data-test="request"]').first();
      const details = card.locator('[data-test="request-details"]');
      await expect(details).toContainText(/now|seconds? ago/);
      await expect(details).toContainText("127.0.0.1");
      await expect(details).toContainText(endpointPath);
      await expect(card).toContainText("POST");
      await expect(
        card.locator('[data-test="request-headers"]'),
      ).toContainText("Content-Type");
      await expect(card.locator('[data-test="request-body"]')).toContainText(
        "Hello, World!",
      );
    });

    test("pretty-prints JSON bodies via highlight.js", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, {
        data: { hello: "world", arr: [1, 2, 3] },
        headers: { "Content-Type": "application/json" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      // Pretty-printed → contains a newline and 2-space indent.
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"hello": "world"');
      expect(text).toContain("\n  ");
      // Highlight.js wraps tokens in <span class="hljs-…"> elements.
      const tokenCount = await body
        .locator("pre span.hljs-string")
        .count();
      expect(tokenCount).toBeGreaterThan(0);
    });

    test("highlights XML bodies", async ({ page, request }) => {
      await post(request, endpointUrl, {
        data: "<root><a>1</a></root>",
        headers: { "Content-Type": "application/xml" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      const tagCount = await body.locator("pre span.hljs-tag").count();
      expect(tagCount).toBeGreaterThan(0);
    });

    test("body copy button writes raw body to clipboard", async ({
      page,
      request,
    }) => {
      const raw = '{"copy":"me"}';
      await post(request, endpointUrl, {
        data: raw,
        headers: { "Content-Type": "application/json" },
      });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-body"]').click();
      const clipboard = await page.evaluate(() =>
        navigator.clipboard.readText(),
      );
      expect(clipboard).toBe(raw);
    });

    test("headers copy button writes the headers JSON to clipboard", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, {
        data: "x",
        headers: { "X-Sample": "value" },
      });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-headers"]').click();
      const clipboard = await page.evaluate(() =>
        navigator.clipboard.readText(),
      );
      const parsed = JSON.parse(clipboard);
      expect(parsed["X-Sample"]).toBe("value");
    });

    test("filters by request body via the search box", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, { data: "Hello, World!" });
      const requests = page.locator('[data-test="requests"]');
      const results = page.locator('[data-test="search-results"]');
      const search = page.locator('[data-test="search-input"]');

      await expect(requests).toContainText("Hello, World!");

      await search.fill("Hello");
      await expect(results).toContainText("1 result");
      await expect(requests).toContainText("Hello, World!");

      await search.fill("Nothing-matches-this");
      await expect(results).toContainText("0 results");
      await expect(requests).toContainText("Waiting for requests");
    });

    test("filters by header key/value via the search box", async ({
      page,
      request,
    }) => {
      const key = "A-Test";
      const value = "Hello-Header";
      await post(request, endpointUrl, {
        data: "x",
        headers: { [key]: value },
      });
      const requests = page.locator('[data-test="requests"]');
      const results = page.locator('[data-test="search-results"]');
      const search = page.locator('[data-test="search-input"]');

      await expect(requests).toContainText(key);
      await expect(requests).toContainText(value);

      await search.fill("A-");
      await expect(results).toContainText("1 result");

      await search.fill("Hello-Header");
      await expect(results).toContainText("1 result");

      await search.fill("not-a-thing");
      await expect(results).toContainText("0 results");
    });

    test("method filter narrows the list to matching methods", async ({
      page,
      request,
    }) => {
      await post(request, endpointUrl, { data: "p1" });
      await request.fetch(endpointUrl, { method: "PUT", data: "p2" });
      await request.fetch(endpointUrl, { method: "DELETE" });

      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "3 results",
      );

      await page
        .locator('[data-test="method-filter"]')
        .selectOption("POST");
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "1 result",
      );
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);

      await page
        .locator('[data-test="method-filter"]')
        .selectOption("");
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "3 results",
      );
    });

    test("delete-all clears every request", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "x" });
      await post(request, endpointUrl, { data: "y" });
      await page.locator('[data-test="delete-requests"]').click();
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    test("delete-request removes a single card", async ({ page, request }) => {
      await post(request, endpointUrl, { data: "first" });
      const response = await request.post(endpointUrl, {
        data: { msg: "second" },
      });
      const uuid = response.headers()["httphq-request-uuid"];
      const card = page.locator(`#request-${uuid}`);
      await expect(card).toBeAttached();
      await card.locator('[data-test="delete-request"]').click();
      await expect(card).not.toBeAttached();
    });
  });

  test.describe("Tab indicator", () => {
    test("title shows unread counter when the tab is hidden and resets on focus", async ({
      page,
      request,
    }) => {
      // Simulate the tab going to background.
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", {
          configurable: true,
          get: () => true,
        });
        document.dispatchEvent(new Event("visibilitychange"));
      });

      await post(request, endpointUrl, { data: "background-1" });
      await expect.poll(async () => page.title()).toContain(`(1)`);

      await post(request, endpointUrl, { data: "background-2" });
      await expect.poll(async () => page.title()).toContain(`(2)`);

      // Bring the tab back to foreground.
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", {
          configurable: true,
          get: () => false,
        });
        document.dispatchEvent(new Event("visibilitychange"));
      });

      await expect.poll(async () => page.title()).toBe(`${testId} | httphq`);
    });
  });
});
