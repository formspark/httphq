/* Shared helpers loaded on every page. */

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

window.htmlEscape = function (s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
};

window.renderBody = function (body, headers) {
  if (body == null || body === "") return "";
  // Best-effort JSON pretty + highlight.
  try {
    const pretty = JSON.stringify(JSON.parse(body), null, 2);
    if (window.hljs && window.hljs.getLanguage("json")) {
      return window.hljs.highlight(pretty, { language: "json" }).value;
    }
    return window.htmlEscape(pretty);
  } catch (_) {
    // not JSON
  }
  // Heuristic: looks like XML/HTML if it starts with '<'
  const trimmed = body.trimStart();
  if (
    trimmed.startsWith("<") &&
    window.hljs &&
    window.hljs.getLanguage("xml")
  ) {
    return window.hljs.highlight(body, { language: "xml" }).value;
  }
  return window.htmlEscape(body);
};

/* Clipboard helper */

window.copyToClipboard = async function (text) {
  if (navigator.clipboard && window.isSecureContext) {
    return navigator.clipboard.writeText(text);
  }
  // Fallback: temporary textarea
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

/* Header parsing for the send-custom-request panel */

window.parseHeaderLines = function (text) {
  const out = {};
  if (!text) return out;
  for (const line of text.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const idx = trimmed.indexOf(":");
    if (idx <= 0) continue;
    const k = trimmed.slice(0, idx).trim();
    const v = trimmed.slice(idx + 1).trim();
    if (k) out[k] = v;
  }
  return out;
};

/* App-wide style overrides (loading-dots animation, native-select chevron,
   pointer cursor on actionable elements). Injected once at startup. */

(function injectAppStyles() {
  if (document.getElementById("httphq-app-style")) return;
  const style = document.createElement("style");
  style.id = "httphq-app-style";
  style.textContent = [
    // Loading dots in the empty state.
    ".loading-dots::after{content:'';display:inline-block;width:1.5ch;text-align:left;animation:httphq-dots 1.2s steps(4,end) infinite}",
    "@keyframes httphq-dots{0%{content:''}25%{content:'.'}50%{content:'..'}75%,100%{content:'...'}}",
    // Tailwind v4 doesn't set cursor:pointer on buttons by default; restore it.
    "button:not(:disabled),summary,[role=button]:not(:disabled){cursor:pointer}",
    // Custom select chevron, anchored right after the value with sane padding.
    "select.app-select{appearance:none;-webkit-appearance:none;background-image:url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 20 20' fill='%2364748b'%3E%3Cpath fill-rule='evenodd' d='M5.23 7.21a.75.75 0 011.06.02L10 11.083l3.71-3.853a.75.75 0 111.08 1.04l-4.25 4.41a.75.75 0 01-1.08 0L5.21 8.27a.75.75 0 01.02-1.06z' clip-rule='evenodd'/%3E%3C/svg%3E\");background-repeat:no-repeat;background-position:right 0.5rem center;background-size:1.1em;padding-right:2rem}",
  ].join("");
  document.head.appendChild(style);
})();
