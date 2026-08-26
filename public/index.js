/* Shared helpers for the endpoint page. Loaded only where the capture stream
   renders; the marketing and contact pages ship no JavaScript at all. */

const TIME_DIVISIONS = [
  { amount: 60, name: "seconds" },
  { amount: 60, name: "minutes" },
  { amount: 24, name: "hours" },
  { amount: 7, name: "days" },
  { amount: 4.34524, name: "weeks" },
  { amount: 12, name: "months" },
  { amount: Number.POSITIVE_INFINITY, name: "years" },
];

const timeFormatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

window.formatTimeAgo = function (date) {
  let duration = (date - new Date()) / 1000;
  for (let i = 0; i < TIME_DIVISIONS.length; i++) {
    const division = TIME_DIVISIONS[i];
    if (Math.abs(duration) < division.amount) {
      return timeFormatter.format(Math.round(duration), division.name);
    }
    duration /= division.amount;
  }
  return "";
};

/* Copying to the clipboard. The async API is unavailable outside a secure
   context, which a self-hosted deployment reached over plain HTTP is, so the
   older selection-based path stays as the fallback rather than leaving those
   deployments with dead copy buttons. */

window.copyToClipboard = async function (text) {
  if (navigator.clipboard && window.isSecureContext) {
    return navigator.clipboard.writeText(text);
  }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.left = "-1000px";
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand("copy");
  } finally {
    document.body.removeChild(ta);
  }
};

/* Absolute clock time. This is the primary timestamp on a capture: an arrivals
   board shows when something landed, and a relative age alone goes stale the
   moment it is rendered. */

const clockFormatter = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

window.formatClock = function (date) {
  return clockFormatter.format(date);
};

/* A count with its noun, pluralised by adding an s. Several surfaces state a
   count in prose, and one that says "1 requests" reads as a defect in the thing
   being counted rather than in the sentence. */

window.pluralize = function (count, noun) {
  return `${count} ${count === 1 ? noun : `${noun}s`}`;
};

/* Byte sizes for captured bodies.

   Measured in bytes rather than characters, and measured on what was stored:
   httphq holds a body as a Go string, and encoding/json replaces invalid UTF-8
   with U+FFFD on the way out, so for a genuinely binary upload this can differ
   from the original file's size. See database.Request.Body. */

window.byteLength = function (text) {
  return new TextEncoder().encode(text).length;
};

window.formatBytes = function (bytes) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

/* Header parsing for the send-custom-request panel.

   Malformed lines are returned rather than dropped. Silently discarding a line
   that is one typo away from an Authorization header, and then reporting
   success, sends the user chasing an auth bug that does not exist. */

window.parseHeaderLines = function (text) {
  const headers = {};
  const invalid = [];
  if (!text) return { headers, invalid };
  text.split(/\r?\n/).forEach((line, i) => {
    const trimmed = line.trim();
    if (!trimmed) return;
    const idx = trimmed.indexOf(":");
    const key = idx > 0 ? trimmed.slice(0, idx).trim() : "";
    if (!key) {
      invalid.push({ line: i + 1, text: trimmed });
      return;
    }
    headers[key] = trimmed.slice(idx + 1).trim();
  });
  return { headers, invalid };
};
