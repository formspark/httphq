import { test, expect, type APIResponse, type Page } from "@playwright/test";
import {
  captureUrl,
  newEndpointId,
  newestBody,
  newestBodyText,
  pruneExpiredCaptures,
  readClipboard,
  readClipboardJson,
  requestsUrl,
  send,
  type HarDocument,
} from "./support/harness";

/**
 * How long the page is given to hydrate. Alpine is loaded from a CDN, so first
 * paint of anything interactive waits on a third party rather than on the
 * server under test.
 */
const HYDRATION_TIMEOUT_MS = 15_000;

/**
 * The parts of the screen the assertions below reach for. Each selector is
 * spelled once rather than at every call site, so a change to the markup lands
 * in one place.
 */
const stream = (page: Page) => page.getByTestId("requests");
const resultCount = (page: Page) => page.getByTestId("search-results");
const searchBox = (page: Page) => page.getByTestId("search-input");
const requestCards = (page: Page) => page.getByTestId("request");
const newestCard = (page: Page) => requestCards(page).first();

/**
 * The capture a send produced, found by the UUID the server echoes back. Going
 * through the UUID rather than through position is what lets a test assert
 * about its own request while the stream carries others.
 */
const capturedUuid = (response: APIResponse) =>
  response.headers()["httphq-request-uuid"];

const cardFor = (page: Page, uuid: string) => page.locator(`#request-${uuid}`);

test.describe("Endpoint screen", () => {
  let endpointId: string;
  let endpointPath: string;
  let endpointUrl: string;

  test.beforeEach(async ({ page }) => {
    endpointId = newEndpointId();
    endpointPath = `/to/${endpointId}`;
    endpointUrl = captureUrl(endpointId);
    await page.goto(`/${endpointId}`);
    // Gate on something only Alpine can have rendered. The endpoint URL and
    // every other server-rendered element is on screen before the page has
    // hydrated, so waiting on one lets each test start racing a boot that has
    // a third-party script in front of it. The waiting panel is the first
    // thing the mounted component draws on an endpoint with no traffic.
    //
    // Given longer than an ordinary assertion because it is waiting on that
    // script to arrive from a CDN, which is slower and less predictable than
    // anything the app itself does.
    await expect(page.getByTestId("empty-waiting")).toBeVisible({
      timeout: HYDRATION_TIMEOUT_MS,
    });
  });

  test.describe("Page", () => {
    test("the title is the endpoint id", async ({ page }) => {
      await expect(page).toHaveTitle(`${endpointId} | httphq`);
    });

    test("the endpoint URL is shown with a copy button", async ({ page }) => {
      await expect(page.getByTestId("endpoint-url")).toContainText(endpointUrl);
      await expect(page.getByTestId("copy-url")).toBeVisible();
    });

    test("the copy-url button writes the URL to the clipboard", async ({
      page,
    }) => {
      await page.getByTestId("copy-url").click();
      expect(await readClipboard(page)).toBe(endpointUrl);
      await expect(page.getByTestId("copy-url-label")).toContainText("Copied!");
    });

    test("the retention and visibility terms are stated", async ({ page }) => {
      await expect(
        page.getByText("Requests are deleted after 4 hours"),
      ).toBeVisible();
    });

    test("the connection indicator settles on live", async ({ page }) => {
      await expect(page.getByTestId("connection-status")).toContainText("Live");
    });
  });

  test.describe("Capture stream", () => {
    test("an endpoint with no traffic says it is waiting", async ({ page }) => {
      await expect(stream(page)).toContainText("Waiting for requests");
    });

    test("the waiting state goes once a request arrives", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "Hello, World!" });
      await expect(stream(page)).not.toContainText("Waiting for requests");
    });

    test("a request sent while the page is open appears without a reload", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "Real-time-payload" });
      await expect(stream(page)).toContainText("Real-time-payload");
    });

    test("a capture shows its details, headers and body", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "Hello, World!",
        headers: { "Content-Type": "text/plain" },
      });
      const card = newestCard(page);
      const details = card.getByTestId("request-details");
      await expect(details).toContainText(/now|seconds? ago/);
      await expect(details).toContainText("127.0.0.1");
      await expect(details).toContainText(endpointPath);
      await expect(card).toContainText("POST");
      await expect(card.getByTestId("request-headers")).toContainText(
        "Content-Type",
      );
      await expect(card.getByTestId("request-body")).toContainText(
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
      const withQuery = page.getByTestId("query-string").first();
      await expect(withQuery).toContainText("event=charge.succeeded");
      await expect(page.getByTestId("request-path").first()).toContainText(
        "?event=charge.succeeded",
      );

      await send(request, endpointUrl, { data: "y" });
      await expect(page.getByTestId("query-string").first()).toContainText(
        "None",
      );
    });

    test("a bodyless request reports no body rather than an empty panel", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { method: "GET" });
      await expect(newestBody(page)).toContainText("None");
    });

    test("the header count matches the headers listed", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "x",
        headers: { "X-One": "1", "X-Two": "2" },
      });
      const headers = page.getByTestId("request-headers").first();
      await expect(headers).toContainText("X-One");
      await expect(headers).toContainText("X-Two");
      await expect(headers).toContainText(/Headers\s*\(\d+\)/);
    });

    // The size is the reason to open a card before copying it, and it is stated
    // in units rather than raw bytes so a large payload reads at a glance.
    test("the body panel states the payload size", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "a".repeat(2048) });
      const body = newestBody(page);

      await expect(body).toContainText("2.0 KB");
    });

    // The panel measures what was stored, which is bytes, matching the size the
    // HAR export reports and the one a multipart part carries. An ASCII payload
    // cannot tell the two apart, so this one is deliberately not ASCII: 2048
    // characters here are 4096 bytes.
    test("the payload size counts bytes rather than characters", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "é".repeat(2048) });
      const body = newestBody(page);

      await expect(body).toContainText("4.0 KB");
    });

    // Everything on this page changes without a navigation, so an arrival that
    // is not spoken here is one a screen reader user is never told about.
    test("an arriving capture is announced", async ({ page, request }) => {
      await send(request, endpointUrl, { method: "PUT", data: "spoken" });

      await expect(page.getByTestId("announcer")).toHaveText(
        "PUT request received",
      );
    });

    test("delete-request removes a single card", async ({ page, request }) => {
      await send(request, endpointUrl, { data: "first" });
      const response = await send(request, endpointUrl, {
        data: { msg: "second" },
      });
      const uuid = capturedUuid(response);
      const card = cardFor(page, uuid);
      await expect(card).toBeAttached();
      await card.getByTestId("delete-request").click();
      await expect(card).not.toBeAttached();
    });

    test("delete-all asks before clearing every request", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      await send(request, endpointUrl, { data: "y" });

      await page.getByTestId("delete-requests").click();
      // Destructive and unrecoverable, so it confirms first; the requests are
      // still there until the confirmation is accepted.
      await expect(page.getByTestId("delete-confirm")).toBeVisible();
      await expect(requestCards(page)).toHaveCount(2);

      await page.getByTestId("delete-cancel").click();
      await expect(requestCards(page)).toHaveCount(2);

      await page.getByTestId("delete-requests").click();
      await page.getByTestId("delete-confirm-button").click();
      await expect(stream(page)).toContainText("Waiting for requests");
    });

    // Nothing tells the page that the server swept a capture out from under it,
    // so a list left open long enough would go on rendering requests that no
    // longer exist, beside the promise that they were deleted. The page runs
    // this on an interval; the test drives the same pass directly rather than
    // holding the suite open for it.
    test("a capture the server no longer holds stops being rendered", async ({
      page,
      request,
    }) => {
      const response = await send(request, endpointUrl, { data: "swept" });
      const uuid = capturedUuid(response);
      await expect(cardFor(page, uuid)).toBeAttached();

      // Deleted behind the page's back, which is what the retention sweep is
      // from the page's point of view.
      await request.delete(requestsUrl(endpointId));

      await pruneExpiredCaptures(page);

      await expect(stream(page)).toContainText("Waiting for requests");
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
      await expect(resultCount(page)).toContainText(`${overOnePage} results`);

      await expect(requestCards(page)).toHaveCount(25);
      const showMore = page.getByTestId("show-more");
      await expect(showMore).toContainText("Show 1 more");

      await showMore.click();
      await expect(requestCards(page)).toHaveCount(overOnePage);
      await expect(showMore).toBeHidden();
    });
  });

  test.describe("Body rendering", () => {
    test("a JSON body is pretty-printed and syntax-highlighted", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: { hello: "world", arr: [1, 2, 3] },
        headers: { "Content-Type": "application/json" },
      });
      const body = newestBody(page);
      // Pretty-printed → contains a newline and 2-space indent.
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"hello": "world"');
      expect(text).toContain("\n  ");
      // Highlight.js wraps tokens in <span class="hljs-…"> elements.
      const tokenCount = await body.locator("pre span.hljs-string").count();
      expect(tokenCount).toBeGreaterThan(0);
    });

    test("an XML body is syntax-highlighted", async ({ page, request }) => {
      await send(request, endpointUrl, {
        data: "<root><a>1</a></root>",
        headers: { "Content-Type": "application/xml" },
      });
      const body = newestBody(page);
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
      const body = newestBody(page);
      await expect(body).toContainText("onerror=alert(1)");
      await expect(body.locator("img")).toHaveCount(0);
    });

    // Highlighting emits roughly an element per token, so colouring a large
    // payload costs a visible stall and tens of thousands of nodes to decorate
    // text the reader scrolls past. Above the renderer's ceiling the colour is
    // dropped and nothing else is: the body is still whole and still formatted.
    test("a body past the highlight ceiling keeps its text and loses its colour", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: { note: "x".repeat(60_000) },
        headers: { "Content-Type": "application/json" },
      });
      const body = newestBody(page);

      await expect(body).toContainText('"note"');
      expect(await body.locator("pre span.hljs-string").count()).toBe(0);
      expect(await newestBodyText(page)).toContain("x".repeat(1_000));
    });

    test("a multipart body shows its fields as a parsed list", async ({
      page,
      request,
    }) => {
      await request.post(endpointUrl, {
        multipart: { firstName: "Ada", role: "engineer" },
      });
      const body = newestBody(page);
      const text = await body.locator("pre").innerText();
      expect(text).toContain('"name": "firstName"');
      expect(text).toContain('"value": "Ada"');
      expect(text).toContain('"name": "role"');
      expect(text).toContain('"value": "engineer"');
      const tokenCount = await body.locator("pre span.hljs-string").count();
      expect(tokenCount).toBeGreaterThan(0);
    });

    test("a multipart file part shows metadata, never its content", async ({
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
      const text = await newestBodyText(page);
      expect(text).toContain('"filename": "avatar.png"');
      expect(text).toContain('"contentType": "image/png"');
      expect(text).toMatch(/"size":\s*\d+/);
      expect(text).not.toContain("fake-png-bytes");
    });

    test("a repeated multipart field name keeps each of its entries", async ({
      page,
      request,
    }) => {
      const formData = new FormData();
      formData.append("tag", "red");
      formData.append("tag", "blue");
      await request.post(endpointUrl, { multipart: formData });
      const text = await newestBodyText(page);
      const tagValues = [
        ...text.matchAll(/"name": "tag",\s*\n\s*"value": "(\w+)"/g),
      ].map((m) => m[1]);
      expect(tagValues).toEqual(["red", "blue"]);
    });

    // The header grammar allows a quoted boundary, and it may sit behind other
    // parameters. Taking the quotes as part of the boundary, or reading only
    // the first parameter, leaves nothing in the body matching the delimiter
    // and the whole payload falls through to raw text.
    test("a quoted boundary is parsed like an unquoted one", async ({
      page,
      request,
    }) => {
      const boundary = "QuotedBnd";
      await send(request, endpointUrl, {
        data: [
          `--${boundary}`,
          'Content-Disposition: form-data; name="city"',
          "",
          "Ghent",
          `--${boundary}--`,
          "",
        ].join("\r\n"),
        headers: {
          "Content-Type": `multipart/form-data; charset=utf-8; boundary="${boundary}"`,
        },
      });
      const text = await newestBodyText(page);
      expect(text).toContain('"name": "city"');
      expect(text).toContain('"value": "Ghent"');
    });

    test("a multipart body with no boundary falls back to raw text", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, {
        data: "not-actually-parseable-multipart",
        headers: { "Content-Type": "multipart/form-data" },
      });
      const body = newestBody(page);
      await expect(body).toContainText("not-actually-parseable-multipart");
    });
  });

  test.describe("Filtering", () => {
    test("the search box narrows by body", async ({ page, request }) => {
      await send(request, endpointUrl, { data: "Hello, World!" });
      const requests = stream(page);
      const results = resultCount(page);
      const search = searchBox(page);

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

    test("the search box narrows by header name and value", async ({
      page,
      request,
    }) => {
      const key = "A-Test";
      const value = "Hello-Header";
      await send(request, endpointUrl, {
        data: "x",
        headers: { [key]: value },
      });
      const requests = stream(page);
      const results = resultCount(page);
      const search = searchBox(page);

      await expect(requests).toContainText(key);
      await expect(requests).toContainText(value);

      await search.fill("A-");
      await expect(results).toContainText("1 result");

      await search.fill("Hello-Header");
      await expect(results).toContainText("1 result");

      await search.fill("not-a-thing");
      await expect(results).toContainText("0 results");
    });

    test("the search box narrows by query string", async ({
      page,
      request,
    }) => {
      await send(request, `${endpointUrl}?event=charge.succeeded`, {
        data: "x",
      });
      const results = resultCount(page);

      await searchBox(page).fill("charge.succeeded");
      await expect(results).toContainText("1 result");
    });

    // A counter that says "1 results" reads as a defect in the thing being
    // counted rather than in the sentence.
    test("the result count agrees in number with what is on screen", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "only" });
      await expect(resultCount(page)).toHaveText("1 result");

      await send(request, endpointUrl, { data: "second" });
      await expect(resultCount(page)).toHaveText("2 results");
    });

    test("the method filter narrows the list to matching methods", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "p1" });
      await send(request, endpointUrl, { method: "PUT", data: "p2" });
      await send(request, endpointUrl, { method: "DELETE" });

      await expect(resultCount(page)).toContainText("3 results");

      await page.getByTestId("method-filter").selectOption("POST");
      await expect(resultCount(page)).toContainText("1 result");
      await expect(requestCards(page)).toHaveCount(1);

      await page.getByTestId("method-filter").selectOption("");
      await expect(resultCount(page)).toContainText("3 results");
    });

    test("a filtered-empty stream is not reported as waiting", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      await expect(requestCards(page)).toHaveCount(1);

      await searchBox(page).fill("nothing-matches-this-term");
      await expect(page.getByTestId("empty-filtered")).toBeVisible();
      await expect(page.getByTestId("empty-waiting")).toHaveCount(0);

      await page.getByTestId("clear-filters").click();
      await expect(requestCards(page)).toHaveCount(1);
    });

    // Delete-all clears the endpoint, not the filtered view its neighbour
    // copies, so its count stays the whole stream.
    test("the delete-all count is the whole stream, not the filtered view", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "p" });
      await send(request, endpointUrl, { method: "PUT", data: "u" });
      await expect(requestCards(page)).toHaveCount(2);

      await page.getByTestId("method-filter").selectOption("PUT");

      await expect(page.getByTestId("copy-all-har-label")).toContainText(
        "Copy shown (1)",
      );
      await expect(page.getByTestId("delete-requests-label")).toContainText(
        "Delete all (2)",
      );
    });

    // The search runs on the server, so the list it returns is not the
    // endpoint. A destructive control must not take its count from it.
    test("a search does not shrink the delete-all count", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "alpha" });
      await send(request, endpointUrl, { data: "beta" });
      await expect(requestCards(page)).toHaveCount(2);

      await searchBox(page).fill("alpha");
      await expect(resultCount(page)).toContainText("1 result");
      await expect(page.getByTestId("delete-requests-label")).toContainText(
        "Delete all (2)",
      );

      await page.getByTestId("delete-requests").click();
      await expect(page.getByTestId("delete-confirm")).toContainText(
        "Delete all 2 captured requests?",
      );
    });

    test("a search that hides everything still reports the endpoint total", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "alpha" });
      await expect(requestCards(page)).toHaveCount(1);

      await searchBox(page).fill("nothing-matches-this-term");

      await expect(page.getByTestId("empty-filtered")).toContainText(
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
      const card = newestCard(page);
      await card.getByTestId("copy-body").click();
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
      const card = newestCard(page);
      await card.getByTestId("copy-headers").click();
      const parsed = await readClipboardJson<Record<string, string>>(page);
      expect(parsed["X-Sample"]).toBe("value");
    });

    test("the query copy button writes the raw query string", async ({
      page,
      request,
    }) => {
      await send(request, `${endpointUrl}?a=1&b=2`, { data: "x" });
      const card = newestCard(page);
      await card.getByTestId("copy-query").click();
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
      const uuid = capturedUuid(response);
      const card = cardFor(page, uuid);
      await expect(card).toBeAttached();

      await card.getByTestId("copy-request-har").click();
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
      const card = newestCard(page);
      await card.getByTestId("copy-request-har").click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries[0]).not.toHaveProperty("response");
      expect(har.entries[0]).not.toHaveProperty("timings");
      expect(har.entries[0]).not.toHaveProperty("cache");
    });

    test("a bodyless request omits postData", async ({ page, request }) => {
      await send(request, endpointUrl, { method: "GET" });
      const card = newestCard(page);
      await card.getByTestId("copy-request-har").click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries[0].request).not.toHaveProperty("postData");
      expect(har.entries[0].request.bodySize).toBe(0);
    });

    test("the request copy button label flips to Copied!", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "x" });
      const card = newestCard(page);
      await card.getByTestId("copy-request-har").click();
      await expect(card.getByTestId("copy-request-har-label")).toContainText(
        "Copied!",
      );
    });

    test("copy-all writes every visible request, newest first", async ({
      page,
      request,
    }) => {
      await send(request, endpointUrl, { data: "first" });
      await expect(requestCards(page)).toHaveCount(1);
      await send(request, endpointUrl, { data: "second" });
      await expect(requestCards(page)).toHaveCount(2);

      await page.getByTestId("copy-all-har").click();
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
      await expect(resultCount(page)).toContainText("2 results");

      await page.getByTestId("method-filter").selectOption("PUT");
      await expect(page.getByTestId("copy-all-har-label")).toContainText(
        "Copy shown (1)",
      );

      await page.getByTestId("copy-all-har").click();
      const har = await readClipboardJson<HarDocument>(page);

      expect(har.entries).toHaveLength(1);
      expect(har.entries[0].request.method).toBe("PUT");
    });

    // Copying nothing produces an empty document, which reads as a failed
    // export rather than an empty stream.
    test("copy-all is disabled while nothing has been captured", async ({
      page,
    }) => {
      await expect(page.getByTestId("copy-all-har")).toBeDisabled();
    });
  });

  test.describe("Connecting an agent", () => {
    test("the panel is collapsed until it is opened", async ({ page }) => {
      await expect(page.getByTestId("agent-prompt")).toBeHidden();

      await page.getByTestId("agent-toggle").click();

      await expect(page.getByTestId("agent-prompt")).toBeVisible();
    });

    // Both panels sit above the stream. Opening one to read a prompt must not
    // push the other open on top of it.
    test("opening it leaves the send panel closed", async ({ page }) => {
      await page.getByTestId("agent-toggle").click();

      await expect(page.getByTestId("agent-prompt")).toBeVisible();
      await expect(page.getByTestId("send-submit")).toBeHidden();
    });

    // The prompt is built from the request, so it has to name the host the
    // page was actually served from rather than a hardcoded one.
    test("the prompt carries this endpoint's own URLs", async ({ page }) => {
      await page.getByTestId("agent-toggle").click();

      const prompt = page.getByTestId("agent-prompt");
      await expect(prompt).toContainText(endpointUrl);
      await expect(prompt).toContainText(
        `/api/endpoints/${endpointId}/requests`,
      );
    });

    test("the prompt states the cursor loop and the poll interval", async ({
      page,
    }) => {
      await page.getByTestId("agent-toggle").click();

      const prompt = page.getByTestId("agent-prompt");
      await expect(prompt).toContainText("?since=");
      await expect(prompt).toContainText("hasMore");
      await expect(prompt).toContainText("2 seconds");
    });

    // Copied verbatim into another tool, so stray edge whitespace from the
    // template would travel with it.
    test("the copy button writes the prompt with no stray whitespace", async ({
      page,
    }) => {
      await page.getByTestId("agent-toggle").click();
      const shown = await page.getByTestId("agent-prompt").textContent();

      await page.getByTestId("copy-agent-prompt").click();

      const copied = await readClipboard(page);
      expect(copied).toBe(shown);
      expect(copied).toBe(copied.trim());
      expect(copied).toContain(endpointUrl);
    });

    test("the copy button label flips to Copied!", async ({ page }) => {
      await page.getByTestId("agent-toggle").click();
      await page.getByTestId("copy-agent-prompt").click();

      await expect(page.getByTestId("copy-agent-prompt-label")).toContainText(
        "Copied!",
      );
    });
  });

  test.describe("Sending a test request", () => {
    test("the panel is collapsed until it is opened", async ({ page }) => {
      await expect(page.getByTestId("send-submit")).toBeHidden();

      await page.getByTestId("send-toggle").click();

      await expect(page.getByTestId("send-submit")).toBeVisible();
    });

    test("submitting the panel produces a captured request", async ({
      page,
    }) => {
      await page.getByTestId("send-toggle").click();
      await page.getByTestId("send-method").selectOption("PUT");
      await page
        .getByTestId("send-headers")
        .fill("X-Source: panel\nContent-Type: application/json");
      await page.getByTestId("send-body").fill('{"hello":"panel"}');
      await page.getByTestId("send-submit").click();

      const card = newestCard(page);
      await expect(card).toContainText("PUT");
      await expect(card.getByTestId("request-headers")).toContainText(
        "X-Source",
      );
      await expect(card.getByTestId("request-body")).toContainText("panel");
    });

    test("the path and query field reaches the capture", async ({ page }) => {
      await page.getByTestId("send-toggle").click();
      await page
        .getByTestId("send-path")
        .fill("/orders/8821?event=charge.succeeded");
      await page.getByTestId("send-submit").click();

      const card = newestCard(page);
      await expect(card.getByTestId("request-path")).toContainText(
        `${endpointPath}/orders/8821?event=charge.succeeded`,
      );
    });

    // "orders/8821" and "/orders/8821" address the same capture path, so a user
    // who leaves the slash off does not quietly send somewhere else.
    test("a sub-path with no leading slash reaches the same place", async ({
      page,
    }) => {
      await page.getByTestId("send-toggle").click();
      await page.getByTestId("send-path").fill("orders/8821");
      await page.getByTestId("send-submit").click();

      await expect(newestCard(page).getByTestId("request-path")).toContainText(
        `${endpointPath}/orders/8821`,
      );
    });

    // Silently discarding a line that is one typo away from an Authorization
    // header, and then reporting success, sends the user chasing an auth bug
    // that does not exist.
    test("a malformed header line is reported instead of dropped", async ({
      page,
    }) => {
      await page.getByTestId("send-toggle").click();
      await page
        .getByTestId("send-headers")
        .fill("Authorization Bearer sk_test_123");
      await page.getByTestId("send-submit").click();
      await expect(page.getByTestId("send-status")).toContainText(
        "is not a header",
      );
      await expect(requestCards(page)).toHaveCount(0);
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
