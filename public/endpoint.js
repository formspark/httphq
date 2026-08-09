/* Endpoint page: Alpine store + WebSocket + favicon/title indicator. */

(function () {
  const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

  // Palette for each HTTP method badge. Every ink is built at one lightness and
  // every wash at another, so the seven read as one family rather than as seven
  // unrelated defaults. Defined in src/styles/theme.css.
  const METHOD_CLASSES = {
    GET: "bg-get-wash text-get-ink",
    POST: "bg-post-wash text-post-ink",
    PUT: "bg-put-wash text-put-ink",
    PATCH: "bg-patch-wash text-patch-ink",
    DELETE: "bg-delete-wash text-delete-ink",
    HEAD: "bg-head-wash text-head-ink",
    OPTIONS: "bg-options-wash text-options-ink",
  };

  // How many cards are in the DOM at once, and how many more each reveal adds.
  const PAGE_SIZE = 25;

  // Relative timestamps are re-evaluated on this interval. Anything longer and
  // "a few seconds ago" is still on screen a minute later.
  const TICK_MS = 30_000;

  // Reconnect backoff bounds. The page is designed to sit open for hours, so a
  // dropped socket is expected rather than exceptional.
  const RECONNECT_MIN_MS = 1_000;
  const RECONNECT_MAX_MS = 30_000;

  // How long a copy button reads "Copied!" before returning to its label.
  const COPIED_MS = 1_500;

  function transformRequest(r) {
    return { ...r, createdAt: new Date(r.createdAt) };
  }

  document.addEventListener("alpine:init", () => {
    // Only initialise when the endpoint page is rendered.
    const isEndpointPage = !!document.querySelector(
      'main[x-data^="endpointPage"]',
    );
    if (!isEndpointPage) return;

    Alpine.data("endpointPage", endpointPageFactory);

    Alpine.store("main", {
      endpointId: null,
      // `requests` is the current view: the search runs on the server and the
      // response is windowed, so it is not the endpoint. `total` is, and it is
      // what any control acting on the whole endpoint has to report.
      requests: [],
      total: 0,
      search: "",
      methodFilter: "",

      init() {
        // Re-fetches whenever endpointId or search changes.
        Alpine.effect(() => this.fetchRequests());
      },

      setEndpoint(id) {
        this.endpointId = id;
      },

      get visibleRequests() {
        if (!this.methodFilter) return this.requests;
        return this.requests.filter((r) => r.method === this.methodFilter);
      },

      get filtered() {
        return !!this.methodFilter || !!this.search;
      },

      clearFilters() {
        this.methodFilter = "";
        this.search = "";
      },

      fetchRequests() {
        if (!this.endpointId) return;
        const url =
          `/api/endpoints/${this.endpointId}/requests?search=` +
          encodeURIComponent(this.search);
        return fetch(url)
          .then((r) => r.json())
          .then((d) => {
            this.requests = (d.requests || []).map(transformRequest);
            this.total = d.total ?? this.requests.length;
          })
          .catch((err) => console.error(err));
      },

      addRequest(request) {
        this.requests = [transformRequest(request), ...this.requests];
        this.total += 1;
      },

      deleteRequests() {
        return fetch(`/api/endpoints/${this.endpointId}/requests`, {
          method: "DELETE",
        })
          .then(() => {
            this.requests = [];
            this.total = 0;
          })
          .catch((err) => console.error(err));
      },

      deleteRequest(uuid) {
        return fetch(`/api/endpoints/${this.endpointId}/requests/${uuid}`, {
          method: "DELETE",
        })
          .then(() => {
            this.requests = this.requests.filter((r) => r.uuid !== uuid);
            this.total = Math.max(0, this.total - 1);
          })
          .catch((err) => console.error(err));
      },

      // Nothing tells the page that the server swept a capture out from under
      // it, so a list left open long enough renders requests that no longer
      // exist, next to the promise that they were deleted. Dropping them
      // locally is only the immediate half: the list is windowed, so the
      // refetch is what resyncs `total` and pulls any older-but-live capture
      // into the window.
      pruneExpired(retentionMs) {
        if (!retentionMs) return;
        const cutoff = Date.now() - retentionMs;
        const live = this.requests.filter(
          (r) => r.createdAt.getTime() > cutoff,
        );
        if (live.length === this.requests.length) return;
        this.requests = live;
        return this.fetchRequests();
      },
    });
  });

  // x-data factory registered with Alpine via Alpine.data so it's available
  // to the template at the moment Alpine processes x-data (i.e. inside the
  // alpine:init phase). Setting `window.endpointPage` would race with
  // Alpine's DOM walk on first boot.
  function endpointPageFactory() {
    return {
      METHODS,
      pageSize: PAGE_SIZE,
      sendForm: { method: "POST", path: "", body: "", headers: "" },
      sendStatus: "",
      sendFailed: false,
      copiedKey: null,
      pendingDeleteAll: false,
      announcement: "",
      connection: "connecting",
      renderLimit: PAGE_SIZE,
      // Bumped on an interval purely so relative-time expressions that read it
      // re-evaluate. Alpine has no other reason to know that wall-clock time
      // passed, so without this every timestamp freezes at first render.
      tick: 0,
      _baseTitle: document.title,
      _unread: 0,
      _reconnectDelay: RECONNECT_MIN_MS,
      _socket: null,
      _closed: false,

      // Alpine calls this once when the component mounts. Endpoint id and
      // WebSocket URL come from data-* attributes on the root element so the
      // method can match Alpine's expected `init()` signature.
      init() {
        const endpointId = this.$el.dataset.endpointId;
        if (!endpointId) return;
        const scheme = location.protocol === "https:" ? "wss:" : "ws:";
        this._wsUrl = `${scheme}//${location.host}/ws/${endpointId}`;
        // A page served without the window keeps every capture it was given
        // rather than expiring them against a guess.
        const retentionSeconds = Number(this.$el.dataset.retentionSeconds);
        this._retentionMs =
          Number.isFinite(retentionSeconds) && retentionSeconds > 0
            ? retentionSeconds * 1000
            : 0;
        Alpine.store("main").setEndpoint(endpointId);
        this._connectWebSocket();
        setInterval(() => {
          this.tick++;
          Alpine.store("main").pruneExpired(this._retentionMs);
        }, TICK_MS);
        document.addEventListener("visibilitychange", () => {
          if (!document.hidden) this._clearUnread();
        });
        window.addEventListener("beforeunload", () => {
          this._closed = true;
          if (this._socket) this._socket.close();
        });
      },

      // Which empty state applies. They are distinct on purpose: "nothing has
      // arrived" and "your filter hides everything" are different facts, and a
      // panel that asserts the first while the second is true tells the user
      // their traffic never landed.
      get streamState() {
        if (Alpine.store("main").visibleRequests.length > 0) return "list";
        if (Alpine.store("main").filtered) return "filtered";
        return "waiting";
      },

      get renderedRequests() {
        return Alpine.store("main").visibleRequests.slice(0, this.renderLimit);
      },

      get hiddenCount() {
        return Math.max(
          0,
          Alpine.store("main").visibleRequests.length - this.renderLimit,
        );
      },

      showMore() {
        this.renderLimit += PAGE_SIZE;
      },

      _connectWebSocket() {
        if (this._closed) return;
        this.connection =
          this.connection === "live" ? "connecting" : this.connection;
        const socket = new WebSocket(this._wsUrl);
        this._socket = socket;

        socket.addEventListener("open", () => {
          this.connection = "live";
          this._reconnectDelay = RECONNECT_MIN_MS;
          // Traffic that landed while the socket was down is not replayed, so
          // refetch to close the gap rather than silently missing it.
          Alpine.store("main").fetchRequests();
        });

        socket.addEventListener("message", (payload) => {
          let request;
          try {
            request = JSON.parse(payload.data);
          } catch {
            return;
          }
          Alpine.store("main").addRequest(request);
          this.announce(`${request.method} request received`);
          if (document.hidden) {
            this._unread += 1;
            this._renderUnread();
          }
        });

        socket.addEventListener("close", () => this._scheduleReconnect());
        socket.addEventListener("error", () => socket.close());
      },

      // Exponential backoff to a 30s ceiling. A page left open overnight should
      // keep trying without hammering the server once it is genuinely gone.
      _scheduleReconnect() {
        if (this._closed) return;
        this.connection = "disconnected";
        const delay = this._reconnectDelay;
        this._reconnectDelay = Math.min(delay * 2, RECONNECT_MAX_MS);
        setTimeout(() => {
          this.connection = "connecting";
          this._connectWebSocket();
        }, delay);
      },

      _renderUnread() {
        document.title = `(${this._unread}) ${this._baseTitle}`;
        this._setFavicon(faviconWithDot());
      },

      _clearUnread() {
        if (this._unread === 0) return;
        this._unread = 0;
        document.title = this._baseTitle;
        this._setFavicon("/favicon.ico");
      },

      _setFavicon(href) {
        let link = document.querySelector("link[rel='icon']");
        if (!link) {
          link = document.createElement("link");
          link.rel = "icon";
          document.head.appendChild(link);
        }
        link.href = href;
      },

      methodBadge(method) {
        return METHOD_CLASSES[method] || "bg-head-wash text-head-ink";
      },

      formatTimeAgo: window.formatTimeAgo,
      formatClock: window.formatClock,
      formatBytes: window.formatBytes,
      renderBody: window.renderBody,

      // Screen readers get no navigation on this page, so every change that a
      // sighted user sees has to be spoken here or it does not exist.
      announce(message) {
        this.announcement = message;
      },

      // Writes text to the clipboard and flashes the button that asked for it.
      // `key` names that button: several share this one component instance, so
      // a single label field can't tell them apart.
      async _copyAndFlash(text, key, announcement) {
        try {
          await window.copyToClipboard(text);
          this.copiedKey = key;
          this.announce(announcement);
          setTimeout(() => {
            if (this.copiedKey === key) this.copiedKey = null;
          }, COPIED_MS);
        } catch (err) {
          console.error(err);
          this.announce("Copy failed");
        }
      },

      copy(text, key) {
        return this._copyAndFlash(text, key, "Copied to clipboard");
      },

      // Copies requests as a HAR-shaped document.
      copyHar(requests, key) {
        const count = requests.length;
        return this._copyAndFlash(
          window.buildHarExport(requests),
          key,
          `Copied ${count} ${count === 1 ? "request" : "requests"} to clipboard`,
        );
      },

      confirmDeleteAll() {
        this.pendingDeleteAll = true;
      },

      async deleteAllConfirmed() {
        const count = Alpine.store("main").requests.length;
        this.pendingDeleteAll = false;
        await Alpine.store("main").deleteRequests();
        this.announce(
          `Deleted ${count} ${count === 1 ? "request" : "requests"}`,
        );
      },

      async sendCustom() {
        const store = Alpine.store("main");
        if (!store.endpointId) return;
        const parsed = window.parseHeaderLines(this.sendForm.headers);
        if (parsed.invalid.length) {
          this.sendFailed = true;
          this.sendStatus = `Line ${parsed.invalid[0].line} is not a header: "${parsed.invalid[0].text}". Use Key: Value.`;
          this.announce(this.sendStatus);
          return;
        }
        const path = this.sendForm.path.replace(/^\/+/, "");
        const target = `/to/${store.endpointId}${path ? "/" + path : ""}`;
        try {
          this.sendFailed = false;
          this.sendStatus = "Sending…";
          const res = await fetch(target, {
            method: this.sendForm.method,
            headers: parsed.headers,
            body: this.sendForm.body || undefined,
          });
          this.sendFailed = !res.ok;
          this.sendStatus = res.ok
            ? `Sent ${this.sendForm.method}`
            : `The server rejected it: HTTP ${res.status}.`;
          this.announce(this.sendStatus);
          if (res.ok) setTimeout(() => (this.sendStatus = ""), 2000);
        } catch (err) {
          // The browser refuses some combinations outright, e.g. a body on GET.
          // Report its reason rather than swallowing it, and keep it on screen.
          this.sendFailed = true;
          this.sendStatus = `Could not send: ${err.message}`;
          this.announce(this.sendStatus);
        }
      },
    };
  }

  // Generate a 32x32 favicon-with-red-dot data URL.
  function faviconWithDot() {
    const c = document.createElement("canvas");
    c.width = 32;
    c.height = 32;
    const ctx = c.getContext("2d");
    // Soft background so the dot is visible against any browser tab style.
    ctx.fillStyle = "#1e293b"; // neutral-800
    ctx.beginPath();
    ctx.arc(16, 16, 14, 0, 2 * Math.PI);
    ctx.fill();
    ctx.fillStyle = "#ffffff";
    ctx.font = "bold 18px sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText("h", 16, 17);
    // Red badge in upper right
    ctx.fillStyle = "#e11d48"; // danger-600
    ctx.beginPath();
    ctx.arc(24, 8, 7, 0, 2 * Math.PI);
    ctx.fill();
    return c.toDataURL("image/png");
  }
})();
