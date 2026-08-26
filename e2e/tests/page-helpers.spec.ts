import { test, expect } from "@playwright/test";
import { loadPageScripts } from "./support/harness";

/**
 * The helpers the page scripts share, driven directly. Each is a pure function
 * with edges the screens only reach by accident: a count of exactly one, a body
 * sitting on a unit boundary, a header line one character away from valid.
 */

test.describe("Page helpers", () => {
  test.beforeEach(async ({ page }) => {
    await loadPageScripts(page);
  });

  test.describe("Byte length", () => {
    // Everything that reports a size measures what was stored, which is bytes.
    // Counting characters would under-report every payload that is not ASCII.
    test("counts bytes rather than characters", async ({ page }) => {
      const lengths = await page.evaluate(() =>
        ["", "abc", "naïve €", "🙂"].map((text) => window.byteLength(text)),
      );

      expect(lengths).toEqual([0, 3, 10, 4]);
    });
  });

  test.describe("Byte formatting", () => {
    test("changes unit at each threshold", async ({ page }) => {
      const formatted = await page.evaluate(() =>
        [0, 1023, 1024, 1536, 1_048_576, 5_242_880].map((bytes) =>
          window.formatBytes(bytes),
        ),
      );

      expect(formatted).toEqual([
        "0 B",
        "1023 B",
        "1.0 KB",
        "1.5 KB",
        "1.0 MB",
        "5.0 MB",
      ]);
    });
  });

  /**
   * The two timestamps a capture carries. They are rendered together and say
   * different things: the clock is when it landed, the relative age is how long
   * ago that was. A card carrying only the second goes stale on screen, and one
   * carrying only the first leaves the reader to do the subtraction.
   */
  test.describe("Relative time", () => {
    // Driven from an offset rather than a fixed date, because the helper
    // measures against the moment it is called. Every offset is a whole number
    // of its unit: the few milliseconds that pass inside the call would round
    // a value sitting exactly on .5 to the neighbouring unit instead.
    const agoFor = (page: Parameters<typeof loadPageScripts>[0], ms: number) =>
      page.evaluate(
        (offset) => window.formatTimeAgo(new Date(Date.now() + offset)),
        ms,
      );

    test("crosses into the next unit at each division", async ({ page }) => {
      const phrases = await Promise.all(
        [-5_000, -120_000, -7_200_000, -172_800_000].map((ms) =>
          agoFor(page, ms),
        ),
      );

      expect(phrases).toEqual([
        "5 seconds ago",
        "2 minutes ago",
        "2 hours ago",
        "2 days ago",
      ]);
    });

    // The capture that just landed is the one the reader is waiting on, and
    // "0 seconds ago" reads as a stopped clock rather than a fresh arrival.
    test("the capture that just landed reads as now", async ({ page }) => {
      expect(await agoFor(page, 0)).toBe("now");
    });
  });

  test.describe("Clock time", () => {
    test("states hours, minutes and seconds", async ({ page }) => {
      const shown = await page.evaluate(() =>
        window.formatClock(new Date(2026, 0, 2, 13, 4, 5)),
      );

      // The viewer's locale decides between 13:04:05 and 01:04:05 PM, so this
      // asserts the parts every locale renders rather than one arrangement of
      // them. Seconds are the part that matters: captures arrive seconds apart,
      // and a clock without them shows a column of identical timestamps.
      expect(shown.split(":")).toHaveLength(3);
      expect(shown).toContain("04");
      expect(shown).toContain("05");
    });
  });

  /**
   * Header lookup, shared by body rendering and the HAR export. A stored header
   * is a scalar or an array depending on whether it repeated on the wire (see
   * flattenHeaders in src/capture.go), and every caller wants one value.
   */
  test.describe("Header lookup", () => {
    const lookup = (
      page: Parameters<typeof loadPageScripts>[0],
      headers: Record<string, string | string[]> | null,
      name: string,
    ) =>
      page.evaluate((args) => window.headerValue(args.headers, args.name), {
        headers,
        name,
      });

    // Header names are case-insensitive on the wire, and a capture stores
    // whichever case its sender happened to use.
    test("matches a name in any case", async ({ page }) => {
      const found = await lookup(
        page,
        { "content-TYPE": "application/json" },
        "Content-Type",
      );

      expect(found).toBe("application/json");
    });

    test("a repeated header yields its first value", async ({ page }) => {
      const found = await lookup(
        page,
        { Accept: ["text/html", "*/*"] },
        "accept",
      );

      expect(found).toBe("text/html");
    });

    test("a name that was never sent yields nothing", async ({ page }) => {
      const found = await lookup(
        page,
        { Host: "example.com" },
        "Authorization",
      );

      expect(found).toBeUndefined();
    });

    // A capture whose headers did not store is still a capture the page has to
    // render, so the lookup answers rather than throwing.
    test("headers absent altogether yield nothing", async ({ page }) => {
      const found = await lookup(page, null, "Content-Type");

      expect(found).toBeUndefined();
    });
  });

  test.describe("Pluralising", () => {
    // A panel that says "1 requests" reads as a defect in the thing being
    // counted rather than in the sentence counting it.
    test("a count of one keeps the noun singular", async ({ page }) => {
      const phrases = await page.evaluate(() =>
        [0, 1, 2].map((count) => window.pluralize(count, "request")),
      );

      expect(phrases).toEqual(["0 requests", "1 request", "2 requests"]);
    });
  });

  /**
   * The send panel's header field. A malformed line is reported rather than
   * dropped: silently discarding a line that is one typo away from an
   * Authorization header, then reporting success, sends the user chasing an
   * auth failure that was never in their request.
   */
  test.describe("Header lines", () => {
    const parse = (page: Parameters<typeof loadPageScripts>[0], text: string) =>
      page.evaluate((source) => window.parseHeaderLines(source), text);

    test("splits each line on its first colon", async ({ page }) => {
      const parsed = await parse(
        page,
        "Content-Type: application/json\nX-Token: abc",
      );

      expect(parsed.headers).toEqual({
        "Content-Type": "application/json",
        "X-Token": "abc",
      });
      expect(parsed.invalid).toEqual([]);
    });

    test("whitespace around a key and its value is trimmed", async ({
      page,
    }) => {
      const parsed = await parse(page, "   X-Token   :   abc   ");

      expect(parsed.headers).toEqual({ "X-Token": "abc" });
    });

    // A colon is legal inside a value, and a URL or a timestamp carries one.
    test("a colon inside the value is kept", async ({ page }) => {
      const parsed = await parse(page, "X-Origin: https://example.com:8443/a");

      expect(parsed.headers).toEqual({
        "X-Origin": "https://example.com:8443/a",
      });
    });

    test("a blank line is skipped rather than reported", async ({ page }) => {
      const parsed = await parse(page, "\n\nX-Token: abc\n   \n");

      expect(parsed.headers).toEqual({ "X-Token": "abc" });
      expect(parsed.invalid).toEqual([]);
    });

    test("a header repeated in the field keeps its last value", async ({
      page,
    }) => {
      const parsed = await parse(page, "X-Token: first\nX-Token: second");

      expect(parsed.headers).toEqual({ "X-Token": "second" });
    });

    // The number reported is the line the user is looking at, so a report has
    // to count the blank lines it otherwise ignores.
    test("a line with no colon is reported at its own line number", async ({
      page,
    }) => {
      const parsed = await parse(page, "X-Token: abc\n\nnot a header at all");

      expect(parsed.headers).toEqual({ "X-Token": "abc" });
      expect(parsed.invalid).toEqual([{ line: 3, text: "not a header at all" }]);
    });

    test("a line with no key before its colon is reported", async ({
      page,
    }) => {
      const parsed = await parse(page, ": orphaned value");

      expect(parsed.headers).toEqual({});
      expect(parsed.invalid).toEqual([{ line: 1, text: ": orphaned value" }]);
    });

    test("every malformed line is reported, not just the first", async ({
      page,
    }) => {
      const parsed = await parse(page, "one\ntwo\nX-Token: abc");

      expect(parsed.invalid).toEqual([
        { line: 1, text: "one" },
        { line: 2, text: "two" },
      ]);
    });

    test("an empty field yields no headers and no complaints", async ({
      page,
    }) => {
      const parsed = await parse(page, "");

      expect(parsed.headers).toEqual({});
      expect(parsed.invalid).toEqual([]);
    });
  });
});
