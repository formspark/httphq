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
   scalar string or a string[] (see flattenHeaders in capture.go). Exposed as
   window.headerValue because that scalar-or-array contract is shared by every
   consumer of a captured request, not just body rendering. */
function headerValue(headers, name) {
  if (!headers) return undefined;
  const key = Object.keys(headers).find(
    (k) => k.toLowerCase() === name.toLowerCase(),
  );
  if (key === undefined) return undefined;
  const value = headers[key];
  return Array.isArray(value) ? value[0] : value;
}

/* Strips one layer of surrounding double quotes from a header parameter value.
   A value carrying only one quote keeps it: that quote is content, not a
   delimiter. */
function unquote(value) {
  const quoted =
    value.length >= 2 && value.startsWith('"') && value.endsWith('"');
  return quoted ? value.slice(1, -1) : value;
}

/* Looks up one parameter in a header's `key=value` parameter list, unquoted
   and matched case-insensitively against a lower-case name. Returns null when
   the list carries no such parameter, or carries it empty. */
function headerParam(params, name) {
  for (const param of params) {
    const eq = param.indexOf("=");
    if (eq > 0 && param.slice(0, eq).trim().toLowerCase() === name) {
      return unquote(param.slice(eq + 1).trim()) || null;
    }
  }
  return null;
}

/* Extracts the `boundary` parameter from a multipart/form-data Content-Type
   header value (quoted or unquoted). Returns null if the header isn't
   multipart/form-data or carries no boundary. */
function multipartBoundary(contentType) {
  if (!contentType) return null;
  const [mime, ...params] = contentType.split(";").map((s) => s.trim());
  if (mime.toLowerCase() !== "multipart/form-data") return null;
  return headerParam(params, "boundary");
}

/* Drops the line break that ended the delimiter above a part and the one that
   opens the delimiter below it. Both belong to the framing rather than to the
   part. A bare LF is accepted alongside CRLF because a hand-rolled sender does
   not reliably send CRLF. */
function trimPartFraming(segment) {
  return segment.replace(/^\r?\n/, "").replace(/\r?\n$/, "");
}

/* Splits a part at the blank line separating its headers from its content.
   Returns null when there is no blank line, which means the segment is not a
   well-formed part. */
function splitPart(segment) {
  const headerEnd = segment.search(/\r?\n\r?\n/);
  if (headerEnd === -1) return null;
  const contentMatch = segment.slice(headerEnd).match(/^\r?\n\r?\n([\s\S]*)$/);
  return {
    headerLines: segment.slice(0, headerEnd).split(/\r?\n/),
    content: contentMatch ? contentMatch[1] : "",
  };
}

/* Turns one part's headers and content into {name, value} for a field or
   {name, filename, contentType, size} for a file. Returns null when the part
   carries no field name, which is a part the panel cannot label. */
function partFromHeaders(headerLines, content) {
  const disposition = headerLines.find((line) =>
    /^content-disposition\s*:/i.test(line),
  );
  const nameMatch = disposition && disposition.match(/name="([^"]*)"/i);
  if (!nameMatch) return null;
  const name = nameMatch[1];

  const filenameMatch = disposition.match(/filename="([^"]*)"/i);
  if (!filenameMatch) return { name, value: content };

  const contentTypeLine = headerLines.find((line) =>
    /^content-type\s*:/i.test(line),
  );
  return {
    name,
    filename: filenameMatch[1],
    contentType: contentTypeLine
      ? contentTypeLine.split(":").slice(1).join(":").trim()
      : "application/octet-stream",
    // The part is described, never shown: a file's own bytes stay off screen.
    size: window.byteLength(content),
  };
}

/* Best-effort parse of a raw multipart/form-data body into an ordered array of
   parts. Returns null on anything that doesn't look like well-formed
   multipart, so the caller can fall through to the next rendering strategy. */
function parseMultipart(body, boundary) {
  const segments = body.split("--" + boundary);
  // segments[0] is the preamble before the first delimiter, and the last
  // segment is the epilogue after the closing "--boundary--"; both are
  // discarded rather than treated as parts.
  if (segments.length < 3) return null;

  const parts = [];
  for (const segment of segments.slice(1, -1)) {
    const split = splitPart(trimPartFraming(segment));
    if (split === null) return null;
    const part = partFromHeaders(split.headerLines, split.content);
    if (part === null) return null;
    parts.push(part);
  }
  return parts;
}

function renderMultipart(body, headers) {
  const boundary = multipartBoundary(headerValue(headers, "Content-Type"));
  if (!boundary) return null;
  const parts = parseMultipart(body, boundary);
  return parts === null ? null : highlightPrettyJSON(parts);
}

function renderJSON(body) {
  try {
    return highlightPrettyJSON(JSON.parse(body));
  } catch {
    return null;
  }
}

function renderXML(body) {
  return body.trimStart().startsWith("<") ? highlight(body, "xml") : null;
}

/* Ordered fallback. Each strategy claims less about the payload than the one
   before it and returns null to hand the body to the next rather than
   reporting a failure: a body that answers to none of them is still a body the
   reader needs in front of them, so the last word is escaped raw text. */
const renderStrategies = [renderMultipart, renderJSON, renderXML];

window.renderBody = function (body, headers) {
  if (body == null || body === "") return "";

  for (const strategy of renderStrategies) {
    const rendered = strategy(body, headers);
    if (rendered !== null) return rendered;
  }
  return htmlEscape(body);
};

window.headerValue = headerValue;
