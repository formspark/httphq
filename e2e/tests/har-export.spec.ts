import { test, expect } from "@playwright/test";
import {
  buildHar,
  loadPageScripts,
  type CapturedRequest,
} from "./support/harness";

/**
 * The clipboard export, driven directly. The endpoint screen already covers
 * what a copied request looks like end to end; this covers the shapes traffic
 * cannot put in front of the exporter. A header that repeated on the wire is
 * stored as an array, and a body that is not ASCII is measured in bytes rather
 * than characters, and neither survives the round trip through Playwright's
 * request API and the server's own normalising.
 */

/** A capture with every field populated, overridden per test. */
const capture = (fields: Partial<CapturedRequest> = {}): CapturedRequest => ({
  uuid: "00000000-0000-4000-8000-000000000000",
  endpointId: "e2e-fixture",
  ip: "203.0.113.7",
  method: "POST",
  path: "/to/e2e-fixture",
  queryString: "",
  body: "",
  createdAt: "2026-08-09T07:20:05.123Z",
  headers: {},
  ...fields,
});

test.describe("HAR export", () => {
  test.beforeEach(async ({ page }) => {
    await loadPageScripts(page);
  });

  test.describe("Envelope", () => {
    test("names httphq as the creator", async ({ page }) => {
      const har = await buildHar(page, [capture()]);

      expect(har.creator).toEqual({ name: "httphq", version: "1" });
    });

    // A single request and a whole list share one envelope, so a consumer
    // parses the same shape whichever button produced the document.
    test("captures keep the order they were given", async ({ page }) => {
      const har = await buildHar(page, [
        capture({ uuid: "newest", body: "second" }),
        capture({ uuid: "oldest", body: "first" }),
      ]);

      expect(har.entries.map((entry) => entry.id)).toEqual([
        "newest",
        "oldest",
      ]);
    });

    // Reached when the page holds nothing. The envelope still has to parse, or
    // a consumer reading the clipboard sees a syntax error rather than an
    // empty export.
    test("nothing to export is still a valid document", async ({ page }) => {
      const har = await buildHar(page, []);

      expect(har.entries).toEqual([]);
      expect(har.creator.name).toBe("httphq");
    });
  });

  test.describe("Headers", () => {
    // HAR represents a repeated header as one entry per value, so the stored
    // array is expanded rather than joined into a single entry.
    test("a repeated header becomes one entry per value", async ({ page }) => {
      const har = await buildHar(page, [
        capture({ headers: { "X-Trace": ["first", "second"] } }),
      ]);

      expect(har.entries[0].request.headers).toEqual([
        { name: "X-Trace", value: "first" },
        { name: "X-Trace", value: "second" },
      ]);
    });

    test("a single-valued header becomes one entry", async ({ page }) => {
      const har = await buildHar(page, [
        capture({ headers: { "X-Trace": "only" } }),
      ]);

      expect(har.entries[0].request.headers).toEqual([
        { name: "X-Trace", value: "only" },
      ]);
    });

    test("a capture with no headers exports an empty list", async ({
      page,
    }) => {
      const har = await buildHar(page, [capture({ headers: {} })]);

      expect(har.entries[0].request.headers).toEqual([]);
    });
  });

  test.describe("Request URL", () => {
    // A capture taken through one hostname has to round-trip to that hostname,
    // not to whichever host the reader happens to have the page open on.
    test("the captured Host header names the host", async ({ page }) => {
      const har = await buildHar(page, [
        capture({ headers: { Host: "hooks.example.com" } }),
      ]);

      expect(har.entries[0].request.url).toBe(
        "http://hooks.example.com/to/e2e-fixture",
      );
    });

    test("a capture that stored no Host falls back to the page's", async ({
      page,
    }) => {
      const har = await buildHar(page, [capture()]);

      expect(har.entries[0].request.url).toBe(
        `${new URL(page.url()).origin}/to/e2e-fixture`,
      );
    });

    test("the query string is carried on the URL as well", async ({ page }) => {
      const har = await buildHar(page, [
        capture({
          headers: { Host: "hooks.example.com" },
          queryString: "a=1&b=2",
        }),
      ]);

      expect(har.entries[0].request.url).toBe(
        "http://hooks.example.com/to/e2e-fixture?a=1&b=2",
      );
    });
  });

  test.describe("Query string", () => {
    test("each parameter becomes a name and a value", async ({ page }) => {
      const har = await buildHar(page, [capture({ queryString: "a=1&b=2" })]);

      expect(har.entries[0].request.queryString).toEqual([
        { name: "a", value: "1" },
        { name: "b", value: "2" },
      ]);
    });

    // A flag-style parameter carries no value, and dropping it would lose the
    // fact that the client sent it at all.
    test("a parameter with no value keeps its name", async ({ page }) => {
      const har = await buildHar(page, [capture({ queryString: "debug" })]);

      expect(har.entries[0].request.queryString).toEqual([
        { name: "debug", value: "" },
      ]);
    });

    test("a repeated parameter keeps every occurrence", async ({ page }) => {
      const har = await buildHar(page, [
        capture({ queryString: "tag=a&tag=b" }),
      ]);

      expect(har.entries[0].request.queryString).toEqual([
        { name: "tag", value: "a" },
        { name: "tag", value: "b" },
      ]);
    });

    test("no query string exports an empty list", async ({ page }) => {
      const har = await buildHar(page, [capture()]);

      expect(har.entries[0].request.queryString).toEqual([]);
    });
  });

  test.describe("Body", () => {
    test("the body is carried verbatim under its own media type", async ({
      page,
    }) => {
      const har = await buildHar(page, [
        capture({
          body: '{"hello":"world"}',
          headers: { "Content-Type": "application/json" },
        }),
      ]);

      expect(har.entries[0].request.postData).toEqual({
        mimeType: "application/json",
        text: '{"hello":"world"}',
      });
    });

    // Everything that reports a size measures what was stored, which is bytes.
    // Counting characters would under-report every payload that is not ASCII:
    // this body is 11 bytes and 8 UTF-16 units.
    test("the size is measured in bytes, not characters", async ({ page }) => {
      const body = "naïve 🙂";
      const har = await buildHar(page, [capture({ body })]);

      expect(har.entries[0].request.bodySize).toBe(11);
      expect(har.entries[0].request.bodySize).not.toBe(body.length);
    });

    // HAR leaves postData absent rather than empty when there was no payload,
    // so a consumer can tell "sent nothing" from "sent an empty body".
    test("a bodyless capture omits postData entirely", async ({ page }) => {
      const har = await buildHar(page, [capture({ method: "GET" })]);

      expect(har.entries[0].request).not.toHaveProperty("postData");
      expect(har.entries[0].request.bodySize).toBe(0);
    });

    // A body that arrived without one still has to export, or a client that
    // omitted Content-Type produces a document its own consumer cannot parse.
    test("a body with no media type exports an empty one", async ({ page }) => {
      const har = await buildHar(page, [capture({ body: "loose text" })]);

      expect(har.entries[0].request.postData).toEqual({
        mimeType: "",
        text: "loose text",
      });
    });
  });

  test.describe("Capture metadata", () => {
    test("the stored timestamp and client IP are carried", async ({ page }) => {
      const har = await buildHar(page, [capture()]);

      expect(har.entries[0].startedDateTime).toBe("2026-08-09T07:20:05.123Z");
      expect(har.entries[0].clientIPAddress).toBe("203.0.113.7");
    });

    // httphq never observes the protocol, so every entry claims the same one
    // rather than guessing per capture.
    test("every entry claims HTTP/1.1", async ({ page }) => {
      const har = await buildHar(page, [capture({ method: "PUT" })]);

      expect(har.entries[0].request.httpVersion).toBe("HTTP/1.1");
      expect(har.entries[0].request.method).toBe("PUT");
    });
  });
});
