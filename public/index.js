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
