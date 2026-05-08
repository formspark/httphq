/* Home page: render recent endpoints from localStorage. */

(function () {
  function relativeTime(ts) {
    if (typeof window.formatTimeAgo !== "function") return "";
    return window.formatTimeAgo(new Date(ts));
  }

  function render() {
    const container = document.getElementById("recent-endpoints");
    const list = document.getElementById("recent-endpoints-list");
    if (!container || !list || !window.recentEndpoints) return;

    const items = window.recentEndpoints.get();
    if (items.length === 0) {
      container.classList.add("hidden");
      return;
    }
    container.classList.remove("hidden");
    list.innerHTML = items
      .map(({ id, createdAt }) => {
        const safeId = window.htmlEscape(id);
        return (
          `<li>` +
          `<a href="/${safeId}" data-test="recent-endpoint" ` +
          `class="flex items-center justify-between gap-3 px-4 py-3 hover:bg-slate-50 transition">` +
          `<span class="font-mono text-sm text-indigo-700">${safeId}</span>` +
          `<span class="text-xs text-slate-400">${window.htmlEscape(relativeTime(createdAt))}</span>` +
          `</a>` +
          `</li>`
        );
      })
      .join("");
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", render);
  } else {
    render();
  }
})();
