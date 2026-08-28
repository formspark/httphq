import { test, expect } from "@playwright/test";
import { loadPageScripts } from "./support/harness";

/**
 * The body renderer's own contract, driven directly rather than through a
 * capture. Everything asserted here is a shape the endpoint screen cannot
 * stage: the server parses and re-serialises a multipart body before storing
 * it, so a part with no name, a part framed with bare LF, or a part whose
 * headers never close is normalised away or refused before the page sees it.
 * The parser still has to answer for those, because the stored body is not the
 * only thing it is ever handed.
 */

/**
 * Undoes the escaping the renderer applies on its way onto the page. No
 * highlighter is loaded here, so a rendered body is its text with the five HTML
 * characters escaped, and reversing that recovers what was rendered.
 */
const unescapeHtml = (html: string) =>
  html
    .replaceAll("&quot;", '"')
    .replaceAll("&#39;", "'")
    .replaceAll("&lt;", "<")
    .replaceAll("&gt;", ">")
    .replaceAll("&amp;", "&");

const BOUNDARY = "SpecBnd";

/** A multipart body from its parts, framed with the given line ending. */
const multipart = (parts: string[], eol = "\r\n") =>
  parts
    .map((part) => `--${BOUNDARY}${eol}${part}`)
    .concat(`--${BOUNDARY}--${eol}`)
    .join("");

const contentType = (value: string) => ({ "Content-Type": value });

test.describe("Body renderer", () => {
  test.beforeEach(async ({ page }) => {
    await loadPageScripts(page);
  });

  /** Renders a body and returns what it put on the page, unescaped. */
  const render = async (
    page: Parameters<typeof loadPageScripts>[0],
    body: string | null,
    headers: Record<string, string> | null = null,
  ) =>
    unescapeHtml(
      await page.evaluate(([b, h]) => window.renderBody(b, h), [
        body,
        headers,
      ] as const),
    );

  test.describe("Multipart parts", () => {
    test("parses parts framed with bare LF as well as CRLF", async ({
      page,
    }) => {
      const rendered = await render(
        page,
        multipart(
          ['Content-Disposition: form-data; name="city"\n\nGhent\n'],
          "\n",
        ),
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      expect(JSON.parse(rendered)).toEqual([{ name: "city", value: "Ghent" }]);
    });

    test("a field carrying its own content type is still a field", async ({
      page,
    }) => {
      const rendered = await render(
        page,
        multipart([
          'Content-Disposition: form-data; name="note"\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nhello\r\n',
        ]),
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      // No filename means no file, whatever else the part declares about
      // itself. A field rendered as a file would report a size and hide its
      // value, which is the one thing the reader wanted.
      expect(JSON.parse(rendered)).toEqual([{ name: "note", value: "hello" }]);
    });

    test("a file part is measured in bytes, not characters", async ({
      page,
    }) => {
      const rendered = await render(
        page,
        multipart([
          'Content-Disposition: form-data; name="f"; filename="n.txt"\r\nContent-Type: text/plain\r\n\r\nnaïve €\r\n',
        ]),
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      // "naïve €" is 7 characters and 10 bytes. A character count would
      // under-report every upload that is not plain ASCII.
      expect(JSON.parse(rendered)).toEqual([
        {
          name: "f",
          filename: "n.txt",
          contentType: "text/plain",
          size: 10,
        },
      ]);
    });

    test("a file part's own bytes are never rendered", async ({ page }) => {
      const rendered = await render(
        page,
        multipart([
          'Content-Disposition: form-data; name="f"; filename="secret.bin"\r\nContent-Type: application/octet-stream\r\n\r\nSHOULD-NOT-APPEAR\r\n',
        ]),
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      expect(rendered).not.toContain("SHOULD-NOT-APPEAR");
    });
  });

  /**
   * A body that does not parse cleanly is handed to the next strategy rather
   * than half-rendered. A partial part list would read as the whole payload,
   * and the reader would never learn that anything was dropped.
   */
  test.describe("Multipart rejection", () => {
    const rejected: Record<string, string> = {
      "a part with no content-disposition": "X-Other: 1\r\n\r\norphan\r\n",
      "a part whose disposition names no field":
        "Content-Disposition: form-data\r\n\r\nnameless\r\n",
      "a part whose headers are never closed by a blank line":
        'Content-Disposition: form-data; name="a"\r\n',
    };

    for (const [description, part] of Object.entries(rejected)) {
      test(`${description} drops the whole body to raw text`, async ({
        page,
      }) => {
        const body = multipart([part]);

        const rendered = await render(
          page,
          body,
          contentType(`multipart/form-data; boundary=${BOUNDARY}`),
        );

        expect(rendered).toBe(body);
      });
    }

    test("a body with no delimiter at all falls through", async ({ page }) => {
      const rendered = await render(
        page,
        "nothing here looks like a part",
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      expect(rendered).toBe("nothing here looks like a part");
    });

    // Falling through rather than reporting a failure is what makes the chain
    // work: a body that is not the multipart it claims to be is still shown as
    // whatever it actually is.
    test("a body that is JSON despite its content type is pretty-printed", async ({
      page,
    }) => {
      const rendered = await render(
        page,
        '{"actually":"json"}',
        contentType(`multipart/form-data; boundary=${BOUNDARY}`),
      );

      expect(rendered).toBe('{\n  "actually": "json"\n}');
    });
  });

  test.describe("Boundary parameter", () => {
    const bodyFor = () =>
      multipart(['Content-Disposition: form-data; name="a"\r\n\r\n1\r\n']);

    const accepted: Record<string, string> = {
      unquoted: `multipart/form-data; boundary=${BOUNDARY}`,
      quoted: `multipart/form-data; boundary="${BOUNDARY}"`,
      "behind another parameter": `multipart/form-data; charset=utf-8; boundary=${BOUNDARY}`,
      "spelled in any case": `MULTIPART/FORM-DATA; BOUNDARY=${BOUNDARY}`,
      "padded with spaces": `multipart/form-data ;  boundary = ${BOUNDARY} `,
    };

    for (const [description, header] of Object.entries(accepted)) {
      test(`a boundary ${description} is found`, async ({ page }) => {
        const rendered = await render(page, bodyFor(), contentType(header));

        expect(JSON.parse(rendered)).toEqual([{ name: "a", value: "1" }]);
      });
    }

    const ignored: Record<string, string> = {
      absent: "multipart/form-data",
      empty: "multipart/form-data; boundary=",
      "belonging to another media type": `multipart/mixed; boundary=${BOUNDARY}`,
    };

    for (const [description, header] of Object.entries(ignored)) {
      test(`a boundary ${description} leaves the body raw`, async ({
        page,
      }) => {
        const body = bodyFor();

        const rendered = await render(page, body, contentType(header));

        expect(rendered).toBe(body);
      });
    }

    // The header may repeat, and the stored value is then a list. Reading the
    // list itself rather than its first entry finds no boundary at all.
    test("a repeated content-type header is read from its first value", async ({
      page,
    }) => {
      const rendered = unescapeHtml(
        await page.evaluate(
          ([body, boundary]) =>
            window.renderBody(body, {
              "Content-Type": [
                `multipart/form-data; boundary=${boundary}`,
                "text/plain",
              ],
            }),
          [bodyFor(), BOUNDARY] as const,
        ),
      );

      expect(JSON.parse(rendered)).toEqual([{ name: "a", value: "1" }]);
    });
  });
});
