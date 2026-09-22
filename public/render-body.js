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

/** @param {string} s */
function htmlEscape(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

/**
 * @param {string} text
 * @param {string} language
 */
function highlight(text, language) {
  if (text.length > HIGHLIGHT_LIMIT) return htmlEscape(text);
  if (window.hljs && window.hljs.getLanguage(language)) {
    return window.hljs.highlight(text, { language }).value;
  }
  return htmlEscape(text);
}

/** @param {unknown} value */
function highlightPrettyJSON(value) {
  return highlight(JSON.stringify(value, null, 2), "json");
}

/**
 * Strips one layer of surrounding double quotes from a header parameter value.
 * A value carrying only one quote keeps it: that quote is content, not a
 * delimiter.
 *
 * @param {string} value
 */
function unquote(value) {
  const quoted =
    value.length >= 2 && value.startsWith('"') && value.endsWith('"');
  return quoted ? value.slice(1, -1) : value;
}

/**
 * Splits one `key=value` header parameter into its lower-cased key and its
 * unquoted value. Returns null when there is no `=` or nothing precedes it,
 * since neither names a parameter.
 *
 * @param {string} param
 */
function parseParam(param) {
  const eq = param.indexOf("=");
  if (eq <= 0) return null;
  return {
    key: param.slice(0, eq).trim().toLowerCase(),
    value: unquote(param.slice(eq + 1).trim()),
  };
}

/**
 * Looks up one parameter in a header's `key=value` parameter list, unquoted and
 * matched case-insensitively against a lower-case name. Returns null when the
 * list carries no such parameter, or carries it empty.
 *
 * @param {string[]} params
 * @param {string} name
 */
function headerParam(params, name) {
  const param = params.map(parseParam).find((p) => p && p.key === name);
  return (param && param.value) || null;
}

/**
 * Extracts the `boundary` parameter from a multipart/form-data Content-Type
 * header value (quoted or unquoted). Returns null if the header isn't
 * multipart/form-data or carries no boundary.
 *
 * @param {string | undefined} contentType
 */
function multipartBoundary(contentType) {
  if (!contentType) return null;
  const [mime = "", ...params] = contentType.split(";").map((s) => s.trim());
  if (mime.toLowerCase() !== "multipart/form-data") return null;
  return headerParam(params, "boundary");
}

/**
 * Drops the line break that ended the delimiter above a part and the one that
 * opens the delimiter below it. Both belong to the framing rather than to the
 * part. A bare LF is accepted alongside CRLF because a hand-rolled sender does
 * not reliably send CRLF.
 *
 * @param {string} segment
 */
function trimPartFraming(segment) {
  return segment.replace(/^\r?\n/, "").replace(/\r?\n$/, "");
}

/**
 * Splits a part at the blank line separating its headers from its content.
 * Returns null when there is no blank line, which means the segment is not a
 * well-formed part.
 *
 * @param {string} segment
 */
function splitPart(segment) {
  const headerEnd = segment.search(/\r?\n\r?\n/);
  if (headerEnd === -1) return null;
  const contentMatch = segment.slice(headerEnd).match(/^\r?\n\r?\n([\s\S]*)$/);
  return {
    headerLines: segment.slice(0, headerEnd).split(/\r?\n/),
    content: contentMatch?.[1] ?? "",
  };
}

/**
 * The value of one of a part's own headers, matched case-insensitively against
 * a lower-case name, or undefined when the part does not carry it.
 *
 * @param {string[]} headerLines
 * @param {string} name
 */
function partHeader(headerLines, name) {
  const pattern = new RegExp(`^${name}\\s*:`, "i");
  const line = headerLines.find((candidate) => pattern.test(candidate));
  return line === undefined
    ? undefined
    : line.slice(line.indexOf(":") + 1).trim();
}

/**
 * The media type a file part declares, or the generic binary type when it
 * declares none.
 *
 * @param {string[]} headerLines
 */
function partContentType(headerLines) {
  return partHeader(headerLines, "content-type") ?? "application/octet-stream";
}

/**
 * Turns one part's headers and content into {name, value} for a field or
 * {name, filename, contentType, size} for a file. Returns null when the part
 * carries no field name, which is a part the panel cannot label.
 *
 * @param {string[]} headerLines
 * @param {string} content
 */
function partFromHeaders(headerLines, content) {
  const disposition = partHeader(headerLines, "content-disposition");
  // Matched as a whole parameter, because "filename" ends in "name" and
  // parameter order is not fixed.
  const nameMatch =
    disposition && disposition.match(/(?:^|;)\s*name="([^"]*)"/i);
  if (!nameMatch) return null;
  const name = nameMatch[1];

  const filenameMatch = disposition.match(/(?:^|;)\s*filename="([^"]*)"/i);
  if (!filenameMatch) return { name, value: content };

  return {
    name,
    filename: filenameMatch[1],
    contentType: partContentType(headerLines),
    // The part is described, never shown: a file's own bytes stay off screen.
    size: window.byteLength(content),
  };
}

/**
 * Parses the segment between two delimiters into one part. Returns null when
 * the segment is not a well-formed part or is one the panel cannot label.
 *
 * @param {string} segment
 */
function parsePart(segment) {
  const split = splitPart(trimPartFraming(segment));
  if (split === null) return null;
  return partFromHeaders(split.headerLines, split.content);
}

/**
 * Best-effort parse of a raw multipart/form-data body into an ordered array of
 * parts. Returns null on anything that doesn't look like well-formed multipart,
 * so the caller can fall through to the next rendering strategy.
 *
 * @param {string} body
 * @param {string} boundary
 */
function parseMultipart(body, boundary) {
  const segments = body.split("--" + boundary);
  // segments[0] is the preamble before the first delimiter, and the last
  // segment is the epilogue after the closing "--boundary--"; both are
  // discarded rather than treated as parts.
  if (segments.length < 3) return null;

  const parts = [];
  for (const segment of segments.slice(1, -1)) {
    const part = parsePart(segment);
    if (part === null) return null;
    parts.push(part);
  }
  return parts;
}

/**
 * @param {string} body
 * @param {CaptureHeaders | null | undefined} headers
 */
function renderMultipart(body, headers) {
  const boundary = multipartBoundary(
    window.headerValue(headers, "Content-Type"),
  );
  if (!boundary) return null;
  const parts = parseMultipart(body, boundary);
  return parts === null ? null : highlightPrettyJSON(parts);
}

/** @param {string} body */
function renderJSON(body) {
  try {
    return highlightPrettyJSON(JSON.parse(body));
  } catch {
    return null;
  }
}

/** @param {string} body */
function renderXML(body) {
  return body.trimStart().startsWith("<") ? highlight(body, "xml") : null;
}

/* Ordered fallback. Each strategy claims less about the payload than the one
   before it and returns null to hand the body to the next rather than
   reporting a failure: a body that answers to none of them is still a body the
   reader needs in front of them, so the last word is escaped raw text. */
const renderStrategies = [renderMultipart, renderJSON, renderXML];

/**
 * @param {string} body
 * @param {CaptureHeaders | null | undefined} headers
 */
function renderByStrategy(body, headers) {
  for (const strategy of renderStrategies) {
    const rendered = strategy(body, headers);
    if (rendered !== null) return rendered;
  }
  return htmlEscape(body);
}

window.renderBody = function (body, headers) {
  if (body == null || body === "") return "";
  return renderByStrategy(body, headers);
};
