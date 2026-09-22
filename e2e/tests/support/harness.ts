import type { APIRequestContext, Page } from "@playwright/test";
import { z } from "zod";

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

/**
 * The helpers the page scripts publish are declared once, in
 * types/page-scripts.d.ts. Only the endpoint page's store is declared here:
 * endpoint.js is not type-checked, and a test reaches into the store for the
 * one pass it drives directly.
 */
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

/**
 * A capture's headers. Scalar-or-array because a header may repeat; see
 * flattenHeaders in src/capture.go. `satisfies` holds the schema to the
 * declaration the page scripts are checked against.
 */
export const captureHeadersSchema = z.record(
  z.string(),
  z.union([z.string(), z.array(z.string())]),
) satisfies z.ZodType<CaptureHeaders>;

/** One capture, as the JSON API hands it over. */
const capturedRequestSchema = z.object({
  uuid: z.string(),
  endpointId: z.string(),
  ip: z.string(),
  method: z.string(),
  path: z.string(),
  queryString: z.string(),
  body: z.string(),
  createdAt: z.string(),
  headers: captureHeadersSchema,
});

export type CapturedRequest = z.infer<typeof capturedRequestSchema>;

/**
 * The listing response, as the page and any poller read it. Keys the suite
 * does not read are dropped rather than refused, so a field the API gains
 * does not fail a test about something else.
 */
const requestListingSchema = z.object({
  requests: z.array(capturedRequestSchema),
  total: z.number(),
  cursor: z.string(),
  hasMore: z.boolean(),
});

export type RequestListing = z.infer<typeof requestListingSchema>;

/**
 * Reads back what an endpoint has captured, parsed against the listing's
 * shape, so a response that changed fails here rather than as an undefined
 * deep inside an assertion.
 */
export const listRequests = async (
  request: APIRequestContext,
  endpointId: string,
): Promise<RequestListing> => {
  const response = await request.get(requestsUrl(endpointId));
  return requestListingSchema.parse(await response.json());
};

/**
 * The body panel of the newest capture on screen. Every body assertion reaches
 * for the same panel, so the locator is spelled once rather than at each call
 * site, and a change to the markup lands in one place.
 */
export const newestBody = (page: Page) =>
  page.getByTestId("request-body").first();

/** The newest capture's body as displayed text, highlighting flattened away. */
export const newestBodyText = (page: Page) =>
  newestBody(page).locator("pre").innerText();

export const readClipboard = (page: Page) =>
  page.evaluate(() => navigator.clipboard.readText());

/** Reads the clipboard as JSON, parsed against the shape the test expects. */
export const readClipboardJson = async <Schema extends z.ZodType>(
  page: Page,
  schema: Schema,
): Promise<z.infer<Schema>> =>
  schema.parse(JSON.parse(await readClipboard(page)));

/**
 * The clipboard export. HAR 1.2 field names, but entries carry only a
 * `request`: httphq never observes a response. Strict objects, because this is
 * the document the exporter is tested on: a key it should not emit fails.
 */
const harNameValueSchema = z.strictObject({
  name: z.string(),
  value: z.string(),
});

const harEntrySchema = z.strictObject({
  id: z.string(),
  startedDateTime: z.string(),
  clientIPAddress: z.string(),
  request: z.strictObject({
    method: z.string(),
    url: z.string(),
    httpVersion: z.string(),
    headers: z.array(harNameValueSchema),
    queryString: z.array(harNameValueSchema),
    postData: z
      .strictObject({ mimeType: z.string(), text: z.string() })
      .optional(),
    bodySize: z.number(),
  }),
});

export const harDocumentSchema = z.strictObject({
  creator: z.strictObject({ name: z.string(), version: z.string() }),
  entries: z.array(harEntrySchema),
});

export type HarDocument = z.infer<typeof harDocumentSchema>;

/** The one entry of a single-request export, failing the test otherwise. */
export const soleEntry = (har: HarDocument) => {
  const [entry, ...rest] = har.entries;
  if (entry === undefined || rest.length > 0) {
    throw new Error(`Expected one HAR entry, found ${har.entries.length}`);
  }
  return entry;
};

/**
 * Runs the exporter over captures built in the test rather than over traffic.
 * A repeated header and a body that is not ASCII both reach the exporter from a
 * real client, but neither survives the round trip through Playwright's request
 * API and the server's own normalising, so driving it here is what holds it to
 * its whole contract.
 */
export const buildHar = async (
  page: Page,
  requests: CapturedRequest[] | null,
): Promise<HarDocument> =>
  harDocumentSchema.parse(
    JSON.parse(
      await page.evaluate((list) => window.buildHarExport(list), requests),
    ),
  );
