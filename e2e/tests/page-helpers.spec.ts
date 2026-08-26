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
