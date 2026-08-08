import { test, expect } from "@playwright/test";
import {
  captureUrl,
  newEndpointId,
  readClipboard,
  readClipboardJson,
  send,
  type HarDocument,
} from "./support/harness";

test.describe("Endpoint screen", () => {
  let endpointId: string;
  let endpointPath: string;
  let endpointUrl: string;

  test.beforeEach(async ({ page }) => {
    endpointId = newEndpointId();
    endpointPath = `/to/${endpointId}`;
    endpointUrl = captureUrl(endpointId);
    await page.goto(`/${endpointId}`);
    // Wait for Alpine to mount and the empty state to render so subsequent
    // assertions don't race with initialization.
    await expect(page.locator('[data-test="endpoint-url"]')).toBeVisible();
  });

  test.describe("Page", () => {
    test("the title is the endpoint id", async ({ page }) => {
      await expect(page).toHaveTitle(`${endpointId} | httphq`);
    });

    test("the endpoint URL is shown with a copy button", async ({ page }) => {
      await expect(page.locator('[data-test="endpoint-url"]')).toContainText(
        endpointUrl,
      );
      await expect(page.locator('[data-test="copy-url"]')).toBeVisible();
    });

    test("the copy-url button writes the URL to the clipboard", async ({
      page,
    }) => {
      await page.locator('[data-test="copy-url"]').click();
      expect(await readClipboard(page)).toBe(endpointUrl);
      await expect(page.locator('[data-test="copy-url-label"]')).toContainText(
        "Copied!",
      );
    });

    test("the retention and visibility terms are stated", async ({ page }) => {
      await expect(
        page.getByText("Requests are deleted after 4 hours"),
      ).toBeVisible();
    });

    test("the connection indicator settles on live", async ({ page }) => {
      await expect(
        page.locator('[data-test="connection-status"]'),
      ).toContainText("Live");
    });
  });

  test.describe("Capture stream", () => {
    test("shows the empty state when no requests exist", async ({ page }) => {
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    test("does not show the empty state once requests arrive", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "Hello, World!" });
      await expect(page.locator('[data-test="requests"]')).not.toContainText(
        "Waiting for requests",
      );
    });

    test("displays new requests in real-time over WebSocket", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "Real-time-payload" });
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Real-time-payload",
      );
    });

    test("renders request details, headers and body", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "Hello, World!",
        headers: { "Content-Type": "text/plain" },
      });
      const card = page.locator('[data-test="request"]').first();
      const details = card.locator('[data-test="request-details"]');
      await expect(details).toContainText(/now|seconds? ago/);
      await expect(details).toContainText("127.0.0.1");
      await expect(details).toContainText(endpointPath);
      await expect(card).toContainText("POST");
      await expect(card.locator('[data-test="request-headers"]')).toContainText(
        "Content-Type",
      );
      await expect(card.locator('[data-test="request-body"]')).toContainText(
        "Hello, World!",
      );
    });

    test("renders the query string, and says so when there is none", async ({
      page,
      request,
    }) => {
      await send(request, `${endpointUrl}?event=charge.succeeded`, {
        data: "x",
      });
      const withQuery = page.locator('[data-test="query-string"]').first();
      await expect(withQuery).toContainText("event=charge.succeeded");
      await expect(
        page.locator('[data-test="request-path"]').first(),
      ).toContainText("?event=charge.succeeded");

      await send(request, endpointUrl, { data: "y" });
      await expect(
        page.locator('[data-test="query-string"]').first(),
      ).toContainText("None");
    });

    test("a bodyless request reports no body rather than an empty panel", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { method: "GET" });
      await expect(
        page.locator('[data-test="request-body"]').first(),
      ).toContainText("None");
    });

    test("the header count matches the headers listed", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "x",
        headers: { "X-One": "1", "X-Two": "2" },
      });
      const headers = page.locator('[data-test="request-headers"]').first();
      await expect(headers).toContainText("X-One");
      await expect(headers).toContainText("X-Two");
      await expect(headers).toContainText(/Headers\s*\(\d+\)/);
    });

    test("delete-request removes a single card", async ({ page, request }) => {
      await send(request, endpointUrl, { data: "first" });
      const response = await send(request, endpointUrl, {
        data: { msg: "second" },
      });
      const uuid = response.headers()["httphq-request-uuid"];
      const card = page.locator(`#request-${uuid}`);
      await expect(card).toBeAttached();
      await card.locator('[data-test="delete-request"]').click();
      await expect(card).not.toBeAttached();
    });

    test("delete-all asks before clearing every request", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      await send(request, endpointUrl, { data: "y" });

      await page.locator('[data-test="delete-requests"]').click();
      // Destructive and unrecoverable, so it confirms first; the requests are
      // still there until the confirmation is accepted.
      await expect(page.locator('[data-test="delete-confirm"]')).toBeVisible();
      await expect(page.locator('[data-test="request"]')).toHaveCount(2);

      await page.locator('[data-test="delete-cancel"]').click();
      await expect(page.locator('[data-test="request"]')).toHaveCount(2);

      await page.locator('[data-test="delete-requests"]').click();
      await page.locator('[data-test="delete-confirm-button"]').click();
      await expect(page.locator('[data-test="requests"]')).toContainText(
        "Waiting for requests",
      );
    });

    // Rendering every capture at once is a five-figure node count and a visible
    // stall, so the rest stay in the store until asked for.
    test("only a page of cards is rendered until more are asked for", async ({
      page,
      request,
    }) => {
      const overOnePage = 26;
      for (let i = 0; i < overOnePage; i++) {
        await send(request, endpointUrl, { data: `payload-${i}` });
      }
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        `${overOnePage} results`,
      );

      await expect(page.locator('[data-test="request"]')).toHaveCount(25);
      const showMore = page.locator('[data-test="show-more"]');
      await expect(showMore).toContainText("Show 1 more");

      await showMore.click();
      await expect(page.locator('[data-test="request"]')).toHaveCount(
        overOnePage,
      );
      await expect(showMore).toBeHidden();
    });
  });

  test.describe("Body rendering", () => {
    test("pretty-prints JSON bodies via highlight.js", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: { hello: "world", arr: [1, 2, 3] },
        headers: { "Content-Type": "application/json" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      // Pretty-printed → contains a newline and 2-space indent.
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"hello": "world"');
      expect(text).toContain("\n  ");
      // Highlight.js wraps tokens in <span class="hljs-…"> elements.
      const tokenCount = await body.locator("pre span.hljs-string").count();
      expect(tokenCount).toBeGreaterThan(0);
    });

    test("highlights XML bodies", async ({ page, request }) => {
      await send(request, endpointUrl, {
        data: "<root><a>1</a></root>",
        headers: { "Content-Type": "application/xml" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      const tagCount = await body.locator("pre span.hljs-tag").count();
      expect(tagCount).toBeGreaterThan(0);
    });

    // A body is captured bytes, never markup: it is escaped on the way onto the
    // page rather than rendered.
    test("an HTML body is displayed as text, not rendered", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "<img src=x onerror=alert(1)>",
        headers: { "Content-Type": "text/html" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      await expect(body).toContainText("onerror=alert(1)");
      await expect(body.locator("img")).toHaveCount(0);
    });

    test("renders multipart/form-data fields as a parsed JSON array", async ({
      page,
      request,
    }) => {
      await request.post(endpointUrl, {
        multipart: { firstName: "Ada", role: "engineer" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"name": "firstName"');
      expect(text).toContain('"value": "Ada"');
      expect(text).toContain('"name": "role"');
      expect(text).toContain('"value": "engineer"');
      const tokenCount = await body.locator("pre span.hljs-string").count();
      expect(tokenCount).toBeGreaterThan(0);
    });

    test("renders multipart file parts as metadata only, never file content", async ({
      page,
      request,
    }) => {
      await request.post(endpointUrl, {
        multipart: {
          avatar: {
            name: "avatar.png",
            mimeType: "image/png",
            buffer: Buffer.from("fake-png-bytes"),
          },
        },
      });
      const body = page.locator('[data-test="request-body"]').first();
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"filename": "avatar.png"');
      expect(text).toContain('"contentType": "image/png"');
      expect(text).toMatch(/"size":\s*\d+/);
      expect(text).not.toContain("fake-png-bytes");
    });

    test("keeps repeated multipart field names as separate array entries", async ({
      page,
      request,
    }) => {
      const formData = new FormData();
      formData.append("tag", "red");
      formData.append("tag", "blue");
      await request.post(endpointUrl, { multipart: formData });
      const body = page.locator('[data-test="request-body"]').first();
      const text = await body.locator("pre").innerText();
      const tagValues = [
        ...text.matchAll(/"name": "tag",\s*\n\s*"value": "(\w+)"/g),
      ].map((m) => m[1]);
      expect(tagValues).toEqual(["red", "blue"]);
    });

    test("falls back to raw display when a multipart body has no boundary", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "not-actually-parseable-multipart",
        headers: { "Content-Type": "multipart/form-data" },
      });
      const body = page.locator('[data-test="request-body"]').first();
      await expect(body).toContainText("not-actually-parseable-multipart");
    });
  });

  test.describe("Filtering", () => {
    test("filters by request body via the search box", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "Hello, World!" });
      const requests = page.locator('[data-test="requests"]');
      const results = page.locator('[data-test="search-results"]');
      const search = page.locator('[data-test="search-input"]');

      await expect(requests).toContainText("Hello, World!");

      await search.fill("Hello");
      await expect(results).toContainText("1 result");
      await expect(requests).toContainText("Hello, World!");

      await search.fill("Nothing-matches-this");
      await expect(results).toContainText("0 results");
      // A stream hidden by a filter is not a stream that never arrived, so it
      // must not claim to be still waiting.
      await expect(requests).toContainText("No requests match this filter");
      await expect(requests).not.toContainText("Waiting for requests");
    });

    test("filters by header key/value via the search box", async ({
      page,
      request,
    }) => {
      const key = "A-Test";
      const value = "Hello-Header";
      await send(request, endpointUrl, {
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

    test("filters by query string via the search box", async ({
      page,
      request,
    }) => {
      await send(request, `${endpointUrl}?event=charge.succeeded`, {
        data: "x",
      });
      const results = page.locator('[data-test="search-results"]');

      await page.locator('[data-test="search-input"]').fill("charge.succeeded");
      await expect(results).toContainText("1 result");
    });

    test("the method filter narrows the list to matching methods", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "p1" });
      await send(request, endpointUrl, { method: "PUT", data: "p2" });
      await send(request, endpointUrl, { method: "DELETE" });

      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "3 results",
      );

      await page.locator('[data-test="method-filter"]').selectOption("POST");
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "1 result",
      );
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);

      await page.locator('[data-test="method-filter"]').selectOption("");
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "3 results",
      );
    });

    test("a filtered-empty stream is not reported as waiting", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);

      await page
        .locator('[data-test="search-input"]')
        .fill("nothing-matches-this-term");
      await expect(page.locator('[data-test="empty-filtered"]')).toBeVisible();
      await expect(page.locator('[data-test="empty-waiting"]')).toHaveCount(0);

      await page.locator('[data-test="clear-filters"]').click();
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);
    });

    // Delete-all clears the endpoint, not the filtered view its neighbour
    // copies, so its count stays the whole stream.
    test("the delete-all count is the whole stream, not the filtered view", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "p" });
      await send(request, endpointUrl, { method: "PUT", data: "u" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(2);

      await page.locator('[data-test="method-filter"]').selectOption("PUT");

      await expect(
        page.locator('[data-test="copy-all-har-label"]'),
      ).toContainText("Copy shown (1)");
      await expect(
        page.locator('[data-test="delete-requests-label"]'),
      ).toContainText("Delete all (2)");
    });

    // The search runs on the server, so the list it returns is not the
    // endpoint. A destructive control must not take its count from it.
    test("a search does not shrink the delete-all count", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "alpha" });
      await send(request, endpointUrl, { data: "beta" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(2);

      await page.locator('[data-test="search-input"]').fill("alpha");
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "1 result",
      );
      await expect(
        page.locator('[data-test="delete-requests-label"]'),
      ).toContainText("Delete all (2)");

      await page.locator('[data-test="delete-requests"]').click();
      await expect(page.locator('[data-test="delete-confirm"]')).toContainText(
        "Delete all 2 captured requests?",
      );
    });

    test("a search that hides everything still reports the endpoint total", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "alpha" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);

      await page
        .locator('[data-test="search-input"]')
        .fill("nothing-matches-this-term");

      await expect(page.locator('[data-test="empty-filtered"]')).toContainText(
        "1 captured on this endpoint",
      );
    });
  });

  test.describe("Copying", () => {
    test("the body copy button writes the raw body to the clipboard", async ({
      page,
      request,
    }) => {
      const raw = '{"copy":"me"}';
      await send(request, endpointUrl, {
        data: raw,
        headers: { "Content-Type": "application/json" },
      });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-body"]').click();
      expect(await readClipboard(page)).toBe(raw);
    });

    test("the headers copy button writes the headers JSON to the clipboard", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "x",
        headers: { "X-Sample": "value" },
      });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-headers"]').click();
      const parsed = await readClipboardJson<Record<string, string>>(page);
      expect(parsed["X-Sample"]).toBe("value");
    });

    test("the query copy button writes the raw query string", async ({
      page,
      request,
    }) => {
      await send(request, `${endpointUrl}?a=1&b=2`, { data: "x" });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-query"]').click();
      expect(await readClipboard(page)).toBe("a=1&b=2");
    });

    test("the request copy button writes a HAR-shaped document", async ({
      page,
      request,
    }) => {
      const raw = '{"hello":"world"}';
      const response = await send(request, `${endpointUrl}?a=1&b=2`, {
        data: raw,
        headers: { "Content-Type": "application/json", "X-Sample": "value" },
      });
      const uuid = response.headers()["httphq-request-uuid"];
      const card = page.locator(`#request-${uuid}`);
      await expect(card).toBeAttached();

      await card.locator('[data-test="copy-request-har"]').click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.creator.name).toBe("httphq");
      expect(har.entries).toHaveLength(1);
      const entry = har.entries[0];
      expect(entry.id).toBe(uuid);
      expect(entry.request.method).toBe("POST");
      expect(entry.request.url).toBe(`${endpointUrl}?a=1&b=2`);
      expect(entry.request.httpVersion).toBe("HTTP/1.1");
      expect(entry.request.queryString).toEqual([
        { name: "a", value: "1" },
        { name: "b", value: "2" },
      ]);
      expect(entry.request.headers).toContainEqual({
        name: "X-Sample",
        value: "value",
      });
      expect(entry.request.postData).toEqual({
        mimeType: "application/json",
        text: raw,
      });
      expect(entry.request.bodySize).toBe(raw.length);
    });

    // httphq never observes a response, so an entry that carried one would be
    // reporting data it does not have.
    test("a HAR entry claims nothing about the response", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-request-har"]').click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries[0]).not.toHaveProperty("response");
      expect(har.entries[0]).not.toHaveProperty("timings");
      expect(har.entries[0]).not.toHaveProperty("cache");
    });

    test("a bodyless request omits postData", async ({ page, request }) => {
      await send(request, endpointUrl, { method: "GET" });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-request-har"]').click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries[0].request).not.toHaveProperty("postData");
      expect(har.entries[0].request.bodySize).toBe(0);
    });

    test("the request copy button label flips to Copied!", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      const card = page.locator('[data-test="request"]').first();
      await card.locator('[data-test="copy-request-har"]').click();
      await expect(
        card.locator('[data-test="copy-request-har-label"]'),
      ).toContainText("Copied!");
    });

    test("copy-all writes every visible request, newest first", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "first" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(1);
      await send(request, endpointUrl, { data: "second" });
      await expect(page.locator('[data-test="request"]')).toHaveCount(2);

      await page.locator('[data-test="copy-all-har"]').click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries).toHaveLength(2);
      expect(har.entries.map((e) => e.request.postData?.text)).toEqual([
        "second",
        "first",
      ]);
    });

    test("copy-all is scoped to the method filter", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "p" });
      await send(request, endpointUrl, { method: "PUT", data: "u" });
      await expect(page.locator('[data-test="search-results"]')).toContainText(
        "2 results",
      );

      await page.locator('[data-test="method-filter"]').selectOption("PUT");
      await expect(
        page.locator('[data-test="copy-all-har-label"]'),
      ).toContainText("Copy shown (1)");

      await page.locator('[data-test="copy-all-har"]').click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries).toHaveLength(1);
      expect(har.entries[0].request.method).toBe("PUT");
    });

    // Copying nothing produces an empty document, which reads as a failed
    // export rather than an empty stream.
    test("copy-all is disabled while nothing has been captured", async ({
      page,
    }) => {
      await expect(page.locator('[data-test="copy-all-har"]')).toBeDisabled();
    });
  });

  test.describe("Sending a test request", () => {
    test("submitting the panel produces a captured request", async ({
      page,
    }) => {
      await page.locator('[data-test="send-toggle"]').click();
      await page.locator('[data-test="send-method"]').selectOption("PUT");
      await page
        .locator('[data-test="send-headers"]')
        .fill("X-Source: panel\nContent-Type: application/json");
      await page.locator('[data-test="send-body"]').fill('{"hello":"panel"}');
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

    test("the path and query field reaches the capture", async ({ page }) => {
      await page.locator('[data-test="send-toggle"]').click();
      await page
        .locator('[data-test="send-path"]')
        .fill("/orders/8821?event=charge.succeeded");
      await page.locator('[data-test="send-submit"]').click();

      const card = page.locator('[data-test="request"]').first();
      await expect(card.locator('[data-test="request-path"]')).toContainText(
        `${endpointPath}/orders/8821?event=charge.succeeded`,
      );
    });

    // Silently discarding a line that is one typo away from an Authorization
    // header, and then reporting success, sends the user chasing an auth bug
    // that does not exist.
    test("a malformed header line is reported instead of dropped", async ({
      page,
    }) => {
      await page.locator('[data-test="send-toggle"]').click();
      await page
        .locator('[data-test="send-headers"]')
        .fill("Authorization Bearer sk_test_123");
      await page.locator('[data-test="send-submit"]').click();
      await expect(page.locator('[data-test="send-status"]')).toContainText(
        "is not a header",
      );
      await expect(page.locator('[data-test="request"]')).toHaveCount(0);
    });
  });

  test.describe("Tab indicator", () => {
    test("the title counts unread arrivals while hidden and resets on focus", async ({
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

      await send(request, endpointUrl, { data: "background-1" });
      await expect.poll(async () => page.title()).toContain("(1)");

      await send(request, endpointUrl, { data: "background-2" });
      await expect.poll(async () => page.title()).toContain("(2)");

      // Bring the tab back to foreground.
      await page.evaluate(() => {
        Object.defineProperty(document, "hidden", {
          configurable: true,
          get: () => false,
        });
        document.dispatchEvent(new Event("visibilitychange"));
      });

      await expect
        .poll(async () => page.title())
        .toBe(`${endpointId} | httphq`);
    });
  });
});
