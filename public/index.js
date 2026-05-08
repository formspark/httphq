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
  if (trimmed.startsWith("<") && window.hljs && window.hljs.getLanguage("xml")) {
    return window.hljs.highlight(body, { language: "xml" }).value;
  }
  return window.htmlEscape(body);
};

/* Recent endpoints history (localStorage) */

const RECENT_KEY = "httphq:recent-endpoints";
const RECENT_LIMIT = 5;

window.recentEndpoints = {
  get() {
    try {
      const raw = localStorage.getItem(RECENT_KEY);
      if (!raw) return [];
      const list = JSON.parse(raw);
      return Array.isArray(list) ? list.slice(0, RECENT_LIMIT) : [];
    } catch (_) {
      return [];
    }
  },
  add(id) {
    if (!id) return;
    try {
      const existing = window.recentEndpoints.get().filter((e) => e.id !== id);
      const next = [{ id, createdAt: Date.now() }, ...existing].slice(
        0,
        RECENT_LIMIT,
      );
      localStorage.setItem(RECENT_KEY, JSON.stringify(next));
    } catch (_) {
      /* localStorage may be unavailable; fail silently */
    }
  },
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

/* Loading-dots animation: pure CSS, registered at startup */

(function injectLoadingDotsStyle() {
  if (document.getElementById("httphq-loading-dots-style")) return;
  const style = document.createElement("style");
  style.id = "httphq-loading-dots-style";
  style.textContent =
    ".loading-dots::after{content:'';display:inline-block;width:1.5ch;text-align:left;animation:httphq-dots 1.2s steps(4,end) infinite}" +
    "@keyframes httphq-dots{0%{content:''}25%{content:'.'}50%{content:'..'}75%,100%{content:'...'}}";
  document.head.appendChild(style);
})();
