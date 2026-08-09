import type { APIRequestContext, Page } from "@playwright/test";

/** Where the server under test is reachable, matching the Playwright baseURL. */
export const BASE_URL = "http://localhost:8080";

/**
 * A fresh endpoint ID for one test. Endpoints are implicit — a page exists for
 * any well-formed ID — so a unique ID per test is all the isolation needed to
 * keep one test's captures out of another's stream.
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

declare global {
  interface Window {
    Alpine: {
      store(name: "main"): { pruneExpired(retentionMs: number): unknown };
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

export const readClipboard = (page: Page) =>
  page.evaluate(() => navigator.clipboard.readText());

/**
 * Reads the clipboard as JSON of an expected shape. The assertion is the one
 * place a shape is declared rather than proven — `JSON.parse` cannot know it —
 * and it is deliberately confined here so no test has to make its own.
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
