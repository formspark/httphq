import type { APIRequestContext, Page } from "@playwright/test";

/**
 * Where the server under test is reachable. Spelled once: the Playwright config
 * reads it as both its baseURL and its readiness probe, so the suite and the
 * server it starts cannot disagree about where to look.
 *
 * `port` in src/application.go is a constant, so the override is for the one
 * case that constant anticipates: running beside something that already holds
 * 8080, with the constant changed to match.
 */
export const BASE_URL = process.env.HTTPHQ_BASE_URL ?? "http://localhost:8080";

/**
 * A fresh endpoint ID for one test. Endpoints are implicit, in that a page
 * exists for any well-formed ID, so a unique ID per test is all the isolation
 * needed to keep one test's captures out of another's stream.
 */
export const newEndpointId = () =>
  `e2e-${Math.random().toString(36).slice(2, 8)}`;

/** The public capture URL for an endpoint, as printed on its page. */
export const captureUrl = (endpointId: string) =>
  `${BASE_URL}/to/${endpointId}`;

/** The JSON listing for an endpoint, which is also its delete-all target. */
export const requestsUrl = (endpointId: string) =>
  `${BASE_URL}/api/endpoints/${endpointId}/requests`;

/**
 * The page drops captures that have aged past the retention window and resyncs
 * with the server, on an interval measured in tens of seconds. This runs that
 * pass on demand with a window short enough to expire everything, so a test can
 * assert what the page does about a swept capture without waiting for a tick.
 */
export const pruneExpiredCaptures = (page: Page) =>
  page.evaluate(() => window.Alpine.store("main").pruneExpired(1));

/**
 * Loads the page scripts into a page that carries none of its own, so a test
 * can call them directly.
 *
 * The endpoint page only ever reaches these helpers through captured traffic,
 * and traffic cannot carry every shape they handle: the server normalises a
 * multipart body before storing it, so several part shapes a client can put on
 * the wire never arrive intact. Driving the helpers here is what holds them to
 * their whole contract rather than the part the wire can express.
 */
export const loadPageScripts = async (page: Page) => {
  // The contact screen ships no JavaScript, so whatever answers afterwards came
  // from the scripts under test and not from the page around them.
  await page.goto("/contact");
  for (const url of ["/index.js", "/render-body.js", "/har.js"]) {
    await page.addScriptTag({ url });
  }
};

/** One malformed line from the send panel's header field. */
export type InvalidHeaderLine = { line: number; text: string };

declare global {
  interface Window {
    Alpine: {
      store(name: "main"): { pruneExpired(retentionMs: number): unknown };
    };
    renderBody(
      body: string | null,
      headers: CapturedRequest["headers"] | null,
    ): string;
    byteLength(text: string): number;
    formatBytes(bytes: number): string;
    pluralize(count: number, noun: string): string;
    parseHeaderLines(text: string): {
      headers: Record<string, string>;
      invalid: InvalidHeaderLine[];
    };
  }
}

export type SendOptions = {
  method?: string;
  data?: string | object;
  headers?: Record<string, string>;
};

/**
 * Sends a request straight to the capture URL, bypassing the page. Traffic
 * under test comes from outside the browser, which is how a real user's
 * webhook arrives.
 */
export const send = (
  request: APIRequestContext,
  url: string,
  { method = "POST", ...init }: SendOptions = {},
) => request.fetch(url, { method, ...init });

/**
 * One capture, as the JSON API hands it over. Headers are scalar-or-array
 * because a header may repeat; see flattenHeaders in src/capture.go.
 */
export type CapturedRequest = {
  uuid: string;
  endpointId: string;
  ip: string;
  method: string;
  path: string;
  queryString: string;
  body: string;
  createdAt: string;
  headers: Record<string, string | string[]>;
};

/** The listing response, as the page and any poller read it. */
export type RequestListing = {
  requests: CapturedRequest[];
  total: number;
  cursor: string;
  hasMore: boolean;
};

/**
 * Reads back what an endpoint has captured. `APIResponse.json` cannot know the
 * shape it returns, so this is the one place the listing's is declared rather
 * than proven, confined here so no test has to declare its own.
 */
export const listRequests = async (
  request: APIRequestContext,
  endpointId: string,
): Promise<RequestListing> => {
  const response = await request.get(requestsUrl(endpointId));
  return (await response.json()) as RequestListing;
};

/**
 * The body panel of the newest capture on screen. Every body assertion reaches
 * for the same panel, so the locator is spelled once rather than at each call
 * site, and a change to the markup lands in one place.
 */
export const newestBody = (page: Page) =>
  page.locator('[data-test="request-body"]').first();

/** The newest capture's body as displayed text, highlighting flattened away. */
export const newestBodyText = (page: Page) =>
  newestBody(page).locator("pre").innerText();

export const readClipboard = (page: Page) =>
  page.evaluate(() => navigator.clipboard.readText());

/**
 * Reads the clipboard as JSON of an expected shape. Confined here for the same
 * reason as listRequests: `JSON.parse` cannot know the shape it returns, so the
 * one declaration of it lives in the harness rather than in each test.
 */
export const readClipboardJson = async <T>(page: Page): Promise<T> =>
  JSON.parse(await readClipboard(page)) as T;

/**
 * The clipboard export shape. HAR 1.2 field names, but entries carry only a
 * `request`: httphq never observes a response.
 */
export type HarNameValue = { name: string; value: string };

export type HarEntry = {
  id: string;
  startedDateTime: string;
  clientIPAddress: string;
  request: {
    method: string;
    url: string;
    httpVersion: string;
    headers: HarNameValue[];
    queryString: HarNameValue[];
    postData?: { mimeType: string; text: string };
    bodySize: number;
  };
};

export type HarDocument = {
  creator: { name: string; version: string };
  entries: HarEntry[];
};
