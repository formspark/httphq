/* Endpoint page: Alpine store + WebSocket + favicon/title indicator. */

(function () {
  const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

  // Tailwind classes for each HTTP method badge.
  const METHOD_CLASSES = {
    GET: "bg-blue-50 text-blue-700",
    POST: "bg-emerald-50 text-emerald-700",
    PUT: "bg-amber-50 text-amber-700",
    PATCH: "bg-violet-50 text-violet-700",
    DELETE: "bg-rose-50 text-rose-700",
    HEAD: "bg-slate-100 text-slate-700",
    OPTIONS: "bg-sky-50 text-sky-700",
  };

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
      requests: [],
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

      fetchRequests() {
        if (!this.endpointId) return;
        const url =
          `/api/endpoints/${this.endpointId}/requests?search=` +
          encodeURIComponent(this.search);
        return fetch(url)
          .then((r) => r.json())
          .then((d) => {
            this.requests = (d.requests || []).map(transformRequest);
          })
          .catch((err) => console.error(err));
      },

      addRequest(request) {
        this.requests = [transformRequest(request), ...this.requests];
      },

      deleteRequests() {
        return fetch(`/api/endpoints/${this.endpointId}/requests`, {
          method: "DELETE",
        })
          .then(() => (this.requests = []))
          .catch((err) => console.error(err));
      },

      deleteRequest(uuid) {
        return fetch(
          `/api/endpoints/${this.endpointId}/requests/${uuid}`,
          { method: "DELETE" },
        )
          .then(() => {
            this.requests = this.requests.filter((r) => r.uuid !== uuid);
          })
          .catch((err) => console.error(err));
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
      sendForm: { method: "POST", body: "", headers: "" },
      sendStatus: "",
      copyLabel: "Copy",
      _baseTitle: document.title,
      _unread: 0,

      // Alpine calls this once when the component mounts. Endpoint id and
      // WebSocket URL come from data-* attributes on the root element so the
      // method can match Alpine's expected `init()` signature.
      init() {
        const endpointId = this.$el.dataset.endpointId;
        if (!endpointId) return;
        const scheme = location.protocol === "https:" ? "wss:" : "ws:";
        const wsUrl = `${scheme}//${location.host}/ws/${endpointId}`;
        Alpine.store("main").setEndpoint(endpointId);
        this._connectWebSocket(wsUrl);
        document.addEventListener("visibilitychange", () => {
          if (!document.hidden) this._clearUnread();
        });
      },

      _connectWebSocket(url) {
        const socket = new WebSocket(url);
        socket.addEventListener("message", (payload) => {
          let request;
          try {
            request = JSON.parse(payload.data);
          } catch (_) {
            return;
          }
          Alpine.store("main").addRequest(request);
          if (document.hidden) {
            this._unread += 1;
            this._renderUnread();
          }
        });
        socket.addEventListener("open", () => console.debug("ws:open"));
        socket.addEventListener("close", () => console.debug("ws:close"));
        socket.addEventListener("error", () => console.debug("ws:error"));
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
        return METHOD_CLASSES[method] || "bg-slate-100 text-slate-700";
      },

      formatTimeAgo: window.formatTimeAgo,
      renderBody: window.renderBody,

      async copy(text) {
        try {
          await window.copyToClipboard(text);
          this.copyLabel = "Copied!";
          setTimeout(() => (this.copyLabel = "Copy"), 1500);
        } catch (err) {
          console.error(err);
        }
      },

      async sendCustom() {
        const store = Alpine.store("main");
        if (!store.endpointId) return;
        const headers = window.parseHeaderLines(this.sendForm.headers);
        try {
          this.sendStatus = "Sending…";
          const res = await fetch(`/to/${store.endpointId}`, {
            method: this.sendForm.method,
            headers,
            body: this.sendForm.body || undefined,
          });
          this.sendStatus = res.ok ? "Sent" : `Error: HTTP ${res.status}`;
          setTimeout(() => (this.sendStatus = ""), 2000);
        } catch (err) {
          this.sendStatus = `Error: ${err.message}`;
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
    ctx.fillStyle = "#1e293b"; // slate-800
    ctx.beginPath();
    ctx.arc(16, 16, 14, 0, 2 * Math.PI);
    ctx.fill();
    ctx.fillStyle = "#ffffff";
    ctx.font = "bold 18px sans-serif";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText("h", 16, 17);
    // Red badge in upper right
    ctx.fillStyle = "#e11d48"; // rose-600
    ctx.beginPath();
    ctx.arc(24, 8, 7, 0, 2 * Math.PI);
    ctx.fill();
    return c.toDataURL("image/png");
  }
})();
