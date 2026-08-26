/* Serialises captured requests to a HAR-shaped JSON document for the clipboard.

   The field names and value shapes follow HAR 1.2 so the output is familiar to
   HAR tooling, but entries carry only a `request`: httphq never observes a
   response, so `response`, `timings` and `cache` are omitted rather than
   emitted as stubs that would read as captured data. Loaded on the endpoint
   page only. */

(function () {
  const CREATOR = { name: "httphq", version: "1" };

  // Not captured: the capture handler never records the protocol, so every
  // entry claims HTTP/1.1. Revisit if the request model starts storing it.
  const ASSUMED_HTTP_VERSION = "HTTP/1.1";

  /* Absolute URL for a captured request. `path` already carries the
     /to/<endpoint> prefix; the scheme comes from the page because it is not
     stored, and the Host header is preferred over the page's so a request
     captured through another hostname still round-trips. */
  function entryURL(request) {
    const host = window.headerValue(request.headers, "Host") || location.host;
    const query = request.queryString ? `?${request.queryString}` : "";
    return `${location.protocol}//${host}${request.path}${query}`;
  }

  /* HAR represents a repeated header as one entry per value, so the stored
     scalar-or-array value is expanded rather than joined.

     Order is alphabetical, not wire order: the capture handler flattens
     headers through a Go map and encoding/json sorts keys on marshal. Nothing
     downstream may treat this ordering as meaningful. */
  function entryHeaders(headers) {
    const out = [];
    for (const [name, value] of Object.entries(headers || {})) {
      for (const one of Array.isArray(value) ? value : [value]) {
        out.push({ name, value: String(one) });
      }
    }
    return out;
  }

  function entryQueryString(queryString) {
    if (!queryString) return [];
    return [...new URLSearchParams(queryString)].map(([name, value]) => ({
      name,
      value,
    }));
  }

  function harEntry(request) {
    const body = request.body || "";
    // Omitted entirely for bodyless requests, matching HAR, where postData is
    // absent rather than empty when there was no payload.
    const postData = body
      ? {
          mimeType: window.headerValue(request.headers, "Content-Type") || "",
          // Verbatim, never base64: the body carries the same U+FFFD
          // substitutions the request panel displays.
          text: body,
        }
      : null;

    return {
      id: request.uuid,
      startedDateTime: new Date(request.createdAt).toISOString(),
      clientIPAddress: request.ip,
      request: {
        method: request.method,
        url: entryURL(request),
        httpVersion: ASSUMED_HTTP_VERSION,
        headers: entryHeaders(request.headers),
        queryString: entryQueryString(request.queryString),
        ...(postData ? { postData } : {}),
        bodySize: window.byteLength(body),
      },
    };
  }

  /* Takes captured requests in display order (newest first) and returns the
     document as pretty-printed JSON. A single request and the whole list share
     one envelope, so consumers parse the same shape either way. */
  window.buildHarExport = function (requests) {
    return JSON.stringify(
      { creator: CREATOR, entries: (requests || []).map(harEntry) },
      null,
      2,
    );
  };
})();
