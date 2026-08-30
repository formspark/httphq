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

  // How long the status line holds a successful send before it clears.
  const SENT_MS = 2_000;

  function transformRequest(r) {
    return { ...r, createdAt: new Date(r.createdAt) };
  }

  document.addEventListener("alpine:init", () => {
    // The page scripts are loaded as a set, and only the endpoint page asks
    // for them. Registering the store and the component regardless would put
    // both on any page that later loads the set for one of the other helpers.
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

      // Every call is scoped to the endpoint the page is open on, so the route
      // is spelled once rather than at each call site.
      requestsUrl(suffix = "") {
        return `/api/endpoints/${this.endpointId}/requests${suffix}`;
      },

      fetchRequests() {
        if (!this.endpointId) return;
        const url = this.requestsUrl(
          `?search=${encodeURIComponent(this.search)}`,
        );
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
        return fetch(this.requestsUrl(), { method: "DELETE" })
          .then(() => {
            this.requests = [];
            this.total = 0;
          })
          .catch((err) => console.error(err));
      },

      deleteRequest(uuid) {
        return fetch(this.requestsUrl(`/${uuid}`), { method: "DELETE" })
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
      // Captured rather than hardcoded so the restore returns to whatever URL
      // the markup asked for, including the version the asset index stamped on.
      _baseFavicon:
        document.querySelector("link[rel='icon']")?.href || "/favicon.ico",
      _unread: 0,
      _reconnectDelay: RECONNECT_MIN_MS,
      _socket: null,
      _closed: false,
      _wsUrl: "",
      _retentionMs: 0,

      // The page has exactly one store, and every reader below reaches for it.
      // Naming it once keeps the lookup out of the expressions that use it.
      get store() {
        return Alpine.store("main");
      },

      // Alpine calls this once when the component mounts. Everything the
      // component needs about its endpoint arrives on data-* attributes of the
      // root element rather than as arguments, so the method can match Alpine's
      // expected `init()` signature.
      //
      // The socket URL is taken as rendered rather than rebuilt from location:
      // its ws/wss scheme has to follow the scheme the page was served over,
      // and the server is the side that resolves that through its trusted-proxy
      // config. See endpointURLs in src/endpoint.go.
      init() {
        const { endpointId, websocketUrl, retentionSeconds } = this.$el.dataset;
        // Neither is recoverable from the page, so a render missing either one
        // leaves the component inert rather than pointing a socket at a guess.
        if (!endpointId || !websocketUrl) return;
        this._wsUrl = websocketUrl;
        // A page served without the window keeps every capture it was given
        // rather than expiring them against a guess.
        const seconds = Number(retentionSeconds);
        this._retentionMs =
          Number.isFinite(seconds) && seconds > 0 ? seconds * 1000 : 0;
        this.store.setEndpoint(endpointId);
        this._connectWebSocket();
        setInterval(() => {
          this.tick++;
          this.store.pruneExpired(this._retentionMs);
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
        if (this.store.visibleRequests.length > 0) return "list";
        if (this.store.filtered) return "filtered";
        return "waiting";
      },

      get renderedRequests() {
        return this.store.visibleRequests.slice(0, this.renderLimit);
      },

      get hiddenCount() {
        return Math.max(
          0,
          this.store.visibleRequests.length - this.renderLimit,
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
          this.store.fetchRequests();
        });

        socket.addEventListener("message", (payload) => {
          let request;
          try {
            request = JSON.parse(payload.data);
          } catch {
            return;
          }
          this.store.addRequest(request);
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
        this._setFavicon(this._baseFavicon);
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

      // An unrecognised method still needs a badge. It borrows HEAD's palette
      // rather than carrying an eighth of its own, so the set stays one family.
      methodBadge(method) {
        return METHOD_CLASSES[method] || METHOD_CLASSES.HEAD;
      },

      formatTimeAgo: window.formatTimeAgo,
      formatClock: window.formatClock,
      formatBytes: window.formatBytes,
      byteLength: window.byteLength,
      pluralize: window.pluralize,
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

      // The label a copy button carries: its own while idle, the confirmation
      // while it is the one that last wrote to the clipboard. Several buttons
      // share this component instance, so the key is what tells them apart, and
      // it has to be the same key the click handler passes to copy().
      copyLabel(key, idle) {
        return this.copiedKey === key ? "Copied!" : idle;
      },

      copyHar(requests, key) {
        return this._copyAndFlash(
          window.buildHarExport(requests),
          key,
          `Copied ${window.pluralize(requests.length, "request")} to clipboard`,
        );
      },

      confirmDeleteAll() {
        this.pendingDeleteAll = true;
      },

      async deleteAllConfirmed() {
        const count = this.store.requests.length;
        this.pendingDeleteAll = false;
        await this.store.deleteRequests();
        this.announce(`Deleted ${window.pluralize(count, "request")}`);
      },

      // Every send outcome reaches the user the same way: the status line, the
      // flag that colours it, and the live region that speaks it.
      _reportSend(failed, status) {
        this.sendFailed = failed;
        this.sendStatus = status;
        this.announce(status);
      },

      // Where the compose form points. Leading slashes are dropped from the
      // sub-path so "foo" and "/foo" address the same capture path.
      _sendTarget() {
        const path = this.sendForm.path.replace(/^\/+/, "");
        return `/to/${this.store.endpointId}${path ? "/" + path : ""}`;
      },

      // The header field, parsed. A malformed line is reported and nothing is
      // returned, because sending a request that is missing a header the user
      // wrote would send them chasing a failure that was never in it.
      _sendHeaders() {
        const parsed = window.parseHeaderLines(this.sendForm.headers);
        const [first] = parsed.invalid;
        if (!first) return parsed.headers;
        this._reportSend(
          true,
          `Line ${first.line} is not a header: "${first.text}". Use Key: Value.`,
        );
        return null;
      },

      // The compose form as a request. An empty body field sends no body at
      // all: fetch refuses a body on GET or HEAD, and an empty string is still
      // a body.
      _sendRequest(headers) {
        return fetch(this._sendTarget(), {
          method: this.sendForm.method,
          headers,
          body: this.sendForm.body || undefined,
        });
      },

      async sendCustom() {
        if (!this.store.endpointId) return;
        const headers = this._sendHeaders();
        if (!headers) return;
        try {
          this.sendFailed = false;
          this.sendStatus = "Sending…";
          const res = await this._sendRequest(headers);
          if (!res.ok) {
            this._reportSend(
              true,
              `The server rejected it: HTTP ${res.status}.`,
            );
            return;
          }
          this._reportSend(false, `Sent ${this.sendForm.method}`);
          setTimeout(() => (this.sendStatus = ""), SENT_MS);
        } catch (err) {
          // The browser refuses some combinations outright, e.g. a body on GET.
          // Report its reason rather than swallowing it, and keep it on screen.
          this._reportSend(true, `Could not send: ${err.message}`);
        }
      },
    };
  }

  // Generate a 32x32 data URL of the mark carrying an alert badge, for the tab
  // icon while captures are waiting unread. The fills are literal hexes because
  // a canvas cannot read a CSS custom property; they are brand-600 and
  // danger-600 and must be kept in step with them.
  function faviconWithDot() {
    const c = document.createElement("canvas");
    c.width = 32;
    c.height = 32;
    const ctx = c.getContext("2d");
    // The logo's own geometry, a diamond inset from a 600-unit square, scaled
    // to the canvas: the unread tab shows the same mark as the real favicon
    // rather than a second, unrelated one.
    const scale = 32 / 600;
    ctx.fillStyle = "#525cc1"; // brand-600
    ctx.beginPath();
    ctx.moveTo(300 * scale, 25 * scale);
    ctx.lineTo(575 * scale, 300 * scale);
    ctx.lineTo(300 * scale, 575 * scale);
    ctx.lineTo(25 * scale, 300 * scale);
    ctx.closePath();
    ctx.fill();
    // Badge in the upper right, ringed so its edge survives both the mark
    // behind it and whatever the browser paints behind the tab.
    ctx.fillStyle = "#ffffff";
    ctx.beginPath();
    ctx.arc(24, 8, 7.5, 0, 2 * Math.PI);
    ctx.fill();
    ctx.fillStyle = "#c63144"; // danger-600
    ctx.beginPath();
    ctx.arc(24, 8, 6, 0, 2 * Math.PI);
    ctx.fill();
    return c.toDataURL("image/png");
  }
})();
