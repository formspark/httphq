// The globals the page scripts in public/ share on window. They are classic
// scripts with no module system, so this is the one place their shared surface
// is written down. index.js, render-body.js and har.js are type-checked against
// it, and so is the Playwright suite that calls them, so a helper and its tests
// cannot describe it differently.

/**
 * A capture's headers as stored. A header that repeated on the wire is a list,
 * any other a scalar; see flattenHeaders in src/capture.go.
 */
type CaptureHeaders = Record<string, string | string[]>;

/**
 * One capture as the page scripts are handed it. `createdAt` is the API's RFC
 * 3339 string, or the Date the endpoint page parses it into.
 */
interface Capture {
  uuid: string;
  ip: string;
  method: string;
  path: string;
  queryString: string;
  body: string;
  createdAt: string | Date;
  headers: CaptureHeaders;
}

/** One malformed line from the send panel's header field. */
interface InvalidHeaderLine {
  line: number;
  text: string;
}

interface Window {
  /** The syntax highlighter, loaded from a CDN on the endpoint page only. */
  hljs?: {
    getLanguage(name: string): unknown;
    highlight(code: string, options: { language: string }): { value: string };
  };

  // index.js
  formatTimeAgo(date: Date): string;
  copyToClipboard(text: string): Promise<void>;
  formatClock(date: Date): string;
  pluralize(count: number, noun: string): string;
  byteLength(text: string): number;
  formatBytes(bytes: number): string;
  headerValue(
    headers: CaptureHeaders | null | undefined,
    name: string,
  ): string | undefined;
  parseHeaderLines(text: string): {
    headers: Record<string, string>;
    invalid: InvalidHeaderLine[];
  };

  // render-body.js
  renderBody(
    body: string | null | undefined,
    headers: CaptureHeaders | null | undefined,
  ): string;

  // har.js
  buildHarExport(requests: Capture[] | null): string;
}
