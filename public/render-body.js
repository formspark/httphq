/* Body rendering: best-effort content-type-aware display for a captured
   request body (JSON pretty-print, multipart/form-data part list, XML
   highlighting, or escaped raw text). Loaded on the endpoint page only. */

/* Above this many characters the body is shown as plain escaped text.
   Highlighting runs synchronously on the main thread and emits roughly one
   element per token, so a large payload costs hundreds of milliseconds and tens
   of thousands of nodes to colour text the reader has to scroll past anyway.
   The body is still shown in full and still copyable; only the colour is
   dropped. */
const HIGHLIGHT_LIMIT = 40_000;

function htmlEscape(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function highlight(text, language) {
  if (text.length > HIGHLIGHT_LIMIT) return htmlEscape(text);
  if (window.hljs && window.hljs.getLanguage(language)) {
    return window.hljs.highlight(text, { language }).value;
  }
  return htmlEscape(text);
}

function highlightPrettyJSON(value) {
  return highlight(JSON.stringify(value, null, 2), "json");
}

/* Case-insensitive lookup into a headers object whose values are either a
   scalar string or a string[] (see the flattening in application.go). Exposed
   as window.headerValue because that scalar-or-array contract is shared by
   every consumer of a captured request, not just body rendering. */
function headerValue(headers, name) {
  if (!headers) return undefined;
  const key = Object.keys(headers).find(
    (k) => k.toLowerCase() === name.toLowerCase(),
  );
  if (key === undefined) return undefined;
  const value = headers[key];
  return Array.isArray(value) ? value[0] : value;
}

/* Extracts the `boundary` parameter from a multipart/form-data Content-Type
   header value (quoted or unquoted). Returns null if the header isn't
   multipart/form-data or carries no boundary. */
function multipartBoundary(contentType) {
  if (!contentType) return null;
  const [mime, ...params] = contentType.split(";").map((s) => s.trim());
  if (mime.toLowerCase() !== "multipart/form-data") return null;
  for (const param of params) {
    const eq = param.indexOf("=");
    if (eq <= 0) continue;
    if (param.slice(0, eq).trim().toLowerCase() !== "boundary") continue;
    let value = param.slice(eq + 1).trim();
    if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) {
      value = value.slice(1, -1);
    }
    return value || null;
  }
  return null;
}

/* Best-effort parse of a raw multipart/form-data body into an ordered array
   of parts: {name, value} for fields, {name, filename, contentType, size}
   for file parts (the file content itself is never shown). Returns null on
   anything that doesn't look like well-formed multipart, so the caller can
   fall through to the next rendering strategy. */
function parseMultipart(body, boundary) {
  const segments = body.split("--" + boundary);
  // segments[0] is the preamble before the first delimiter, and the last
  // segment is the epilogue after the closing "--boundary--"; both are
  // discarded rather than treated as parts.
  if (segments.length < 3) return null;

  const parts = [];
  for (let i = 1; i < segments.length - 1; i++) {
    let segment = segments[i];
    if (segment.startsWith("\r\n")) segment = segment.slice(2);
    else if (segment.startsWith("\n")) segment = segment.slice(1);
    if (segment.endsWith("\r\n")) segment = segment.slice(0, -2);
    else if (segment.endsWith("\n")) segment = segment.slice(0, -1);

    const headerEnd = segment.search(/\r?\n\r?\n/);
    if (headerEnd === -1) return null;
    const headerLines = segment.slice(0, headerEnd).split(/\r?\n/);
    const contentMatch = segment
      .slice(headerEnd)
      .match(/^\r?\n\r?\n([\s\S]*)$/);
    const content = contentMatch ? contentMatch[1] : "";

    const disposition = headerLines.find((line) =>
      /^content-disposition\s*:/i.test(line),
    );
    if (!disposition) return null;

    const nameMatch = disposition.match(/name="([^"]*)"/i);
    if (!nameMatch) return null;
    const name = nameMatch[1];

    const filenameMatch = disposition.match(/filename="([^"]*)"/i);
    if (!filenameMatch) {
      parts.push({ name, value: content });
      continue;
    }

    const contentTypeLine = headerLines.find((line) =>
      /^content-type\s*:/i.test(line),
    );
    const contentType = contentTypeLine
      ? contentTypeLine.split(":").slice(1).join(":").trim()
      : "application/octet-stream";
    parts.push({
      name,
      filename: filenameMatch[1],
      contentType,
      // Byte length of the captured body, which httphq stores as a Go
      // string — encoding/json replaces invalid UTF-8 with U+FFFD on
      // marshal, so for genuinely binary uploads this can differ from the
      // true original file size. See database.Request.Body.
      size: new TextEncoder().encode(content).length,
    });
  }
  return parts;
}

window.renderBody = function (body, headers) {
  if (body == null || body === "") return "";

  const boundary = multipartBoundary(headerValue(headers, "Content-Type"));
  if (boundary) {
    const parts = parseMultipart(body, boundary);
    if (parts !== null) return highlightPrettyJSON(parts);
    // Malformed multipart body — fall through to the generic paths below.
  }

  // Best-effort JSON pretty + highlight.
  try {
    return highlightPrettyJSON(JSON.parse(body));
  } catch {
    // not JSON
  }
  // Heuristic: looks like XML/HTML if it starts with '<'
  if (body.trimStart().startsWith("<")) {
    return highlight(body, "xml");
  }
  return htmlEscape(body);
};

window.headerValue = headerValue;
