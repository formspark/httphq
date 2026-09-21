import { test, expect, type Page } from "@playwright/test";
import { loadPageScripts, type CapturedRequest } from "./support/harness";

/**
 * The body renderer's own contract, driven directly rather than through a
 * capture, so each case can state the exact output. Several of these shapes the
 * endpoint screen cannot stage at all: the server parses and re-serialises a
 * multipart body before storing it, so a part with no name, a part framed with
 * bare LF, or a part whose headers never close is normalised away or refused
 * before the page sees it. The parser still has to answer for those, because
 * the stored body is not the only thing it is ever handed.
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

/** The content type every multipart fixture here is framed for. */
const multipartType = contentType(`multipart/form-data; boundary=${BOUNDARY}`);

test.describe("Body renderer", () => {
  test.beforeEach(async ({ page }) => {
    await loadPageScripts(page);
  });

  /** Renders a body and returns what it put on the page, unescaped. */
  const render = async (
    page: Page,
    body: string | null,
    headers: CapturedRequest["headers"] | null = null,
  ) =>
    unescapeHtml(
      await page.evaluate(([b, h]) => window.renderBody(b, h), [
        body,
        headers,
      ] as const),
    );

  /**
   * Renders a multipart body and parses what the renderer printed. The part
   * list is printed as JSON, so the parse is what a test asserts against.
   */
  const renderParts = async (
    page: Page,
    body: string,
    headers: CapturedRequest["headers"] = multipartType,
  ): Promise<unknown> => JSON.parse(await render(page, body, headers));

  test.describe("Multipart parts", () => {
    test("parses parts framed with bare LF as well as CRLF", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart(
          ['Content-Disposition: form-data; name="city"\n\nGhent\n'],
          "\n",
        ),
      );

      expect(parts).toEqual([{ name: "city", value: "Ghent" }]);
    });

    test("a field carrying its own content type is still a field", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart([
          'Content-Disposition: form-data; name="note"\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nhello\r\n',
        ]),
      );

      // No filename means no file, whatever else the part declares about
      // itself. A field rendered as a file would report a size and hide its
      // value, which is the one thing the reader wanted.
      expect(parts).toEqual([{ name: "note", value: "hello" }]);
    });

    test("a file part is measured in bytes, not characters", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart([
          'Content-Disposition: form-data; name="f"; filename="n.txt"\r\nContent-Type: text/plain\r\n\r\nnaïve €\r\n',
        ]),
      );

      // "naïve €" is 7 characters and 10 bytes. A character count would
      // under-report every upload that is not plain ASCII.
      expect(parts).toEqual([
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
        multipartType,
      );

      expect(rendered).not.toContain("SHOULD-NOT-APPEAR");
    });

    // A hand-rolled sender may leave out a file's media type, and the panel
    // still has to say what kind of part it is.
    test("a file part that declares no media type is reported as binary", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart([
          'Content-Disposition: form-data; name="f"; filename="blob"\r\n\r\nxyz\r\n',
        ]),
      );

      expect(parts).toEqual([
        {
          name: "f",
          filename: "blob",
          contentType: "application/octet-stream",
          size: 3,
        },
      ]);
    });

    // A form submitted with a text input left blank still sends the field,
    // and a blank answer is one the reader needs to see.
    test("a field left empty keeps its name and an empty value", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart(['Content-Disposition: form-data; name="note"\r\n\r\n\r\n']),
      );

      expect(parts).toEqual([{ name: "note", value: "" }]);
    });

    // Only the first blank line ends a part's headers. A textarea value can
    // carry blank lines of its own, and they belong to the value.
    test("a value spanning blank lines keeps them", async ({ page }) => {
      const parts = await renderParts(
        page,
        multipart([
          'Content-Disposition: form-data; name="message"\r\n\r\nfirst\r\n\r\nsecond\r\n',
        ]),
      );

      expect(parts).toEqual([
        { name: "message", value: "first\r\n\r\nsecond" },
      ]);
    });

    // Parameter order is not fixed, and "filename" ends in "name". A field
    // name read from the tail of another parameter would label the part with
    // its file's name instead.
    test("a field name is read from its own parameter, whatever the order", async ({
      page,
    }) => {
      const parts = await renderParts(
        page,
        multipart([
          'Content-Disposition: form-data; filename="report.csv"; name="upload"\r\nContent-Type: text/csv\r\n\r\na,b\r\n',
        ]),
      );

      expect(parts).toEqual([
        {
          name: "upload",
          filename: "report.csv",
          contentType: "text/csv",
          size: 3,
        },
      ]);
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
      "a part whose disposition names only a file":
        'Content-Disposition: form-data; filename="a.txt"\r\n\r\nx\r\n',
      "a part whose headers are never closed by a blank line":
        'Content-Disposition: form-data; name="a"\r\n',
    };

    for (const [description, part] of Object.entries(rejected)) {
      test(`${description} drops the whole body to raw text`, async ({
        page,
      }) => {
        const body = multipart([part]);

        const rendered = await render(page, body, multipartType);

        expect(rendered).toBe(body);
      });
    }

    test("a body with no delimiter at all falls through", async ({ page }) => {
      const rendered = await render(
        page,
        "nothing here looks like a part",
        multipartType,
      );

      expect(rendered).toBe("nothing here looks like a part");
    });

    // Falling through rather than reporting a failure is what makes the chain
    // work: a body that is not the multipart it claims to be is still shown as
    // whatever it actually is.
    test("a body that is JSON despite its content type is pretty-printed", async ({
      page,
    }) => {
      const rendered = await render(page, '{"actually":"json"}', multipartType);

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
      "behind a parameter that carries no value": `multipart/form-data; charset; boundary=${BOUNDARY}`,
    };

    for (const [description, header] of Object.entries(accepted)) {
      test(`a boundary ${description} is found`, async ({ page }) => {
        const parts = await renderParts(page, bodyFor(), contentType(header));

        expect(parts).toEqual([{ name: "a", value: "1" }]);
      });
    }

    const ignored: Record<string, string> = {
      absent: "multipart/form-data",
      empty: "multipart/form-data; boundary=",
      "belonging to another media type": `multipart/mixed; boundary=${BOUNDARY}`,
      "quoted but empty": 'multipart/form-data; boundary=""',
      "only as the tail of a longer name": `multipart/form-data; xboundary=${BOUNDARY}`,
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
      const parts = await renderParts(page, bodyFor(), {
        "Content-Type": [
          `multipart/form-data; boundary=${BOUNDARY}`,
          "text/plain",
        ],
      });

      expect(parts).toEqual([{ name: "a", value: "1" }]);
    });
  });

  /**
   * A body that is not multipart. Each strategy claims less about the payload
   * than the one before it, and a body none of them claims is still shown.
   */
  test.describe("Fallback", () => {
    test("an empty or missing body renders nothing", async ({ page }) => {
      expect(await render(page, "")).toBe("");
      expect(await render(page, null)).toBe("");
    });

    // JSON is recognised by parsing it, not by what the sender claimed, so a
    // client that labels its JSON as text still has it pretty-printed.
    test("a JSON body is pretty-printed whatever its content type says", async ({
      page,
    }) => {
      const rendered = await render(
        page,
        '{"a":[1,2]}',
        contentType("text/plain"),
      );

      expect(rendered).toBe('{\n  "a": [\n    1,\n    2\n  ]\n}');
    });

    // The last word is text, and text goes onto the page escaped: a body is
    // captured bytes, never markup. Read without unescaping, since the
    // escaping is what is under test.
    test("a body no strategy claims is escaped rather than rendered", async ({
      page,
    }) => {
      const html = await page.evaluate(() =>
        window.renderBody(`1 < 2 && "a" != 'b' > 0`, null),
      );

      expect(html).toBe(
        "1 &lt; 2 &amp;&amp; &quot;a&quot; != &#39;b&#39; &gt; 0",
      );
    });
  });
});
