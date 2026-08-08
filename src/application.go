package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/contrib/v3/socketio"
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/datatypes"

	"httphq/src/database"
	"httphq/src/logging"
)

const (
	port      = 8080
	bodyLimit = 1 << 20 // 1 MiB
)

var isProduction = os.Getenv("APPLICATION_ENV") == "production"

// Generic forwarding headers stripped from every captured request so users
// see their original payload, not infrastructure-added headers. Vendor headers
// specific to a hosting platform are stripped separately, see platformConfig.
var omittedHeaders = [...]string{
	"Cdn-Loop",
	"Trace",
	"Traceparent",
	"Tracestate",
	"Via",
	"X-Forwarded-For",
	"X-Forwarded-Host",
	"X-Forwarded-Port",
	"X-Forwarded-Proto",
	"X-Forwarded-Server",
	"X-Forwarded-Ssl",
	"X-Real-Ip",
	"X-Request-Start",
}

// platformConfig describes how a hosting platform exposes request metadata:
// which header carries the real client IP, and which vendor headers it adds
// that should be hidden from captured requests — users inspect their own
// traffic and shouldn't have to care which provider sits in front of httphq.
type platformConfig struct {
	ipHeader    string   // header with the real client IP; "" = TCP peer
	ipList      bool     // ipHeader is a comma-separated list; take the leftmost
	stripPrefix []string // captured-request header prefixes to drop as vendor noise
}

// platforms maps the PLATFORM env var to its config. Each platform overwrites
// (or reliably sets) its own headers; the operator is responsible for ensuring
// traffic cannot reach the app bypassing the platform.
var platforms = map[string]platformConfig{
	"direct":     {},
	"cloudflare": {ipHeader: "Cf-Connecting-Ip", stripPrefix: []string{"Cf-"}},
	"fly":        {ipHeader: "Fly-Client-Ip", stripPrefix: []string{"Fly-"}},
	"heroku":     {ipHeader: "X-Forwarded-For", ipList: true},
	"render":     {ipHeader: "X-Forwarded-For", ipList: true},
	"proxy":      {ipHeader: "X-Forwarded-For", ipList: true},
}

// currentPlatform is the config resolved once from PLATFORM at startup.
var currentPlatform platformConfig

// resolvePlatform maps a PLATFORM value to its config. An empty value means
// "direct"; an unrecognised value fails safe to "direct" so a typo never
// causes a spoofable header to be trusted.
func resolvePlatform(name string) platformConfig {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "direct"
	}
	if p, ok := platforms[name]; ok {
		return p
	}
	slog.Warn("unknown PLATFORM, falling back to direct", "platform", name)
	return platforms["direct"]
}

// contentSecurityPolicy is the CSP sent on every response, assembled once at
// startup because its only variable part is fixed for the process lifetime.
//
//   - script-src needs 'unsafe-eval' for Alpine, which compiles directive
//     expressions via the Function constructor, and the two CDN origins that
//     serve Alpine and the syntax highlighter to the endpoint page. All page
//     scripts are external so script-src does NOT need 'unsafe-inline'.
//   - style-src needs the CDN origin for the highlighter's theme, and
//     'unsafe-inline' because Alpine's x-show toggles visibility through an
//     inline display style. The application's own stylesheet is first-party
//     and covered by 'self'.
//   - The local design-tooling origin is allowed in development only: it serves
//     an injected picker script and opens a socket back to itself. Production
//     must never carry it, so it is gated on the environment rather than on a
//     request-time condition a client could influence.
var contentSecurityPolicy = buildContentSecurityPolicy()

func buildContentSecurityPolicy() string {
	designTooling := ""
	if !isProduction {
		designTooling = " http://localhost:8400"
	}
	return "default-src 'self'; " +
		"script-src 'self' 'unsafe-eval' https://unpkg.com https://cdn.jsdelivr.net" + designTooling + "; " +
		"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
		"img-src 'self' data:; " +
		"connect-src 'self' ws: wss:" + designTooling + "; " +
		"frame-ancestors 'none'"
}

// assetVersions maps a served static path to a short hash of its contents,
// computed once at startup. The stylesheet and the page scripts live at fixed
// paths, so without a version in the URL a CDN or a browser can pair a cached
// copy of one deploy with the HTML of the next: every class name the new markup
// asks for resolves to nothing and the page renders unstyled. Versioning the URL
// makes a changed asset a different URL, which no cache can confuse for the old
// one.
var assetVersions = map[string]string{}

// versionedAssets are the first-party files referenced from the templates whose
// contents change between deploys. Images are excluded: they are replaced rarely
// and never in a way that breaks a page that fetched the previous copy.
var versionedAssets = []string{
	"/app.css",
	"/index.js",
	"/render-body.js",
	"/har.js",
	"/endpoint.js",
}

func hashAsset(path string) string {
	f, err := os.Open("./public" + path)
	if err != nil {
		slog.Warn("asset missing, serving unversioned", "path", path, "err", err)
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		slog.Warn("asset unreadable, serving unversioned", "path", path, "err", err)
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))[:10]
}

// assetURL is exposed to templates as `asset`. Production reads the map computed
// at startup. Development re-hashes, but only when the file's modification time
// has moved, so an edited stylesheet is picked up without a restart and without
// re-reading every asset on every render.
var assetMu sync.Mutex
var assetStamps = map[string]time.Time{}

func assetURL(path string) string {
	if isProduction {
		if version := assetVersions[path]; version != "" {
			return path + "?v=" + version
		}
		return path
	}

	assetMu.Lock()
	defer assetMu.Unlock()
	info, err := os.Stat("./public" + path)
	if err == nil && !info.ModTime().Equal(assetStamps[path]) {
		assetStamps[path] = info.ModTime()
		assetVersions[path] = hashAsset(path)
	}
	if version := assetVersions[path]; version != "" {
		return path + "?v=" + version
	}
	return path
}

// trustedProxyConfig lists the peers whose X-Forwarded-* headers Fiber may
// honour once TrustProxy is enabled. httphq is only ever fronted by a reverse
// proxy that reaches it from a private, loopback or link-local address; no
// legitimate direct client connects from those ranges. Bounding trust to them
// means a public client that reaches the app directly still can't spoof the
// scheme or client IP.
func trustedProxyConfig() fiber.TrustProxyConfig {
	return fiber.TrustProxyConfig{
		Private:   true,
		Loopback:  true,
		LinkLocal: true,
	}
}

// endpointURLs builds the public capture and live-feed URLs shown on an
// endpoint page. The WebSocket scheme tracks the request scheme so an HTTPS
// page always advertises wss:// — a ws:// socket on an HTTPS page is blocked
// by browsers as mixed content.
func endpointURLs(scheme, host, endpointID string) (endpointURL, websocketURL string) {
	websocketScheme := "ws"
	if scheme == "https" {
		websocketScheme = "wss"
	}
	return scheme + "://" + host + "/to/" + endpointID,
		websocketScheme + "://" + host + "/ws/" + endpointID
}

// pageBaseURL is the scheme+host a rendered page is being served from, used to
// build the absolute URLs that canonical and Open Graph tags require. It tracks
// the request rather than a configured hostname so a self-hosted deployment
// advertises itself, not httphq.com.
func pageBaseURL(c fiber.Ctx) string {
	return c.Scheme() + "://" + string(c.Request().Host())
}

// omitHeader reports whether a captured-request header is infrastructure noise
// — a generic forwarding header or a vendor header added by the configured
// PLATFORM — and so should be hidden from the user.
func omitHeader(name string) bool {
	for _, h := range omittedHeaders {
		if strings.EqualFold(name, h) {
			return true
		}
	}
	for _, prefix := range currentPlatform.stripPrefix {
		if len(name) >= len(prefix) && strings.EqualFold(name[:len(prefix)], prefix) {
			return true
		}
	}
	return false
}

// endpointIDPattern bounds an endpoint ID to the shape haikunator emits
// (lowercase words and digits joined by hyphens). Rejecting anything else
// keeps attacker-controlled characters — quotes, angle brackets, parens —
// out of the rendered pages, the database, and the logs.
var endpointIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validEndpointID(id string) bool {
	return len(id) <= 64 && endpointIDPattern.MatchString(id)
}

// socketRegistry tracks WS UUIDs subscribed to each endpoint, so the capture
// hot path can fan-out without a DB round-trip.
type socketRegistry struct {
	mu    sync.RWMutex
	byEnd map[string]map[string]struct{}
}

func newSocketRegistry() *socketRegistry {
	return &socketRegistry{byEnd: make(map[string]map[string]struct{})}
}

func (r *socketRegistry) add(endpointID, uuid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set, ok := r.byEnd[endpointID]
	if !ok {
		set = make(map[string]struct{})
		r.byEnd[endpointID] = set
	}
	set[uuid] = struct{}{}
}

func (r *socketRegistry) remove(uuid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for endpointID, set := range r.byEnd {
		if _, ok := set[uuid]; ok {
			delete(set, uuid)
			if len(set) == 0 {
				delete(r.byEnd, endpointID)
			}
			return
		}
	}
}

func (r *socketRegistry) uuidsFor(endpointID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	set := r.byEnd[endpointID]
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for u := range set {
		out = append(out, u)
	}
	return out
}

func (r *socketRegistry) count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, set := range r.byEnd {
		n += len(set)
	}
	return n
}

// resolveClientIP returns the real client IP per the configured PLATFORM
// strategy: it reads the platform's client-IP header (leftmost entry when the
// header is a list) and validates it parses. With no platform configured, or
// when the header is missing or malformed, it falls back to the TCP peer.
func resolveClientIP(c fiber.Ctx) string {
	if currentPlatform.ipHeader != "" {
		v := c.Get(currentPlatform.ipHeader)
		if currentPlatform.ipList {
			if i := strings.IndexByte(v, ','); i >= 0 {
				v = v[:i]
			}
		}
		if ip := net.ParseIP(trimSpace(v)); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(c.IP()); ip != nil {
		return ip.String()
	}
	return ""
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func main() {
	/* Logging */

	env := os.Getenv("APPLICATION_ENV")
	if env == "" {
		env = "development"
	}
	logging.Init("httphq", env)

	/* Client IP strategy */

	currentPlatform = resolvePlatform(os.Getenv("PLATFORM"))
	slog.Info("platform resolved",
		"platform", os.Getenv("PLATFORM"), "ip_header", currentPlatform.ipHeader)

	/* Database */

	database.Connect("file:local.db?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")

	/* Haiku maker */

	haikuMaker := haikunator.New()

	/* Cron */

	c := cron.New()

	everyFiveMinutes := "*/5 * * * *"
	if _, err := c.AddFunc(everyFiveMinutes, func() {
		threshold := time.Now().Add(-1 * 4 * time.Hour)
		database.DeleteOldRequests(context.Background(), threshold)
	}); err != nil {
		slog.Error("cron job registration failed", "err", err)
		os.Exit(1)
	}

	c.Start()

	/* Server */

	engine := html.New("./src/views", ".html")

	for _, path := range versionedAssets {
		assetVersions[path] = hashAsset(path)
	}
	engine.AddFunc("asset", assetURL)

	if !isProduction {
		engine.Reload(true)
		engine.Debug(true)
	}

	application := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
		BodyLimit:   bodyLimit,
		// Trust the upstream proxy only when a PLATFORM is configured, so
		// c.Scheme() honours X-Forwarded-Proto (correct ws/wss + EndpointURL).
		// With no platform, the app is treated as directly exposed and
		// c.IP()/c.Scheme() reflect the real connection.
		//
		// In Fiber v3, TrustProxy only enables the check: X-Forwarded-* is
		// honoured only for a peer that TrustProxyConfig recognises, so a
		// TrustProxyConfig is required for c.Scheme() to report the forwarded
		// scheme rather than the raw connection. httphq is always fronted by a
		// reverse proxy that reaches it from a private, loopback or link-local
		// address, so trust those ranges; deployments must keep the app
		// unreachable except through that proxy.
		TrustProxy:       currentPlatform.ipHeader != "",
		TrustProxyConfig: trustedProxyConfig(),
		ProxyHeader:      fiber.HeaderXForwardedFor,
	})

	// Runs first so every request — including rate-limited ones — gets a
	// correlation ID and a structured access-log line.
	application.Use(requestLogger)

	maxRequestsPerMinute := 9999
	if isProduction {
		maxRequestsPerMinute = 125
	}
	application.Use(limiter.New(limiter.Config{
		Max:        maxRequestsPerMinute,
		Expiration: 1 * time.Minute,
		// Bucket on the spoof-resistant client IP (CF-Connecting-IP behind
		// Cloudflare) rather than c.IP(), which trusts a client-supplied
		// X-Forwarded-For and lets a caller reset its own limit.
		KeyGenerator: func(c fiber.Ctx) string { return resolveClientIP(c) },
	}))

	application.Use(compress.New())

	// Security headers
	application.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Content-Security-Policy", contentSecurityPolicy)
		return c.Next()
	})

	// Static handling
	application.Get("/*", static.New("./public"))

	// WS handling
	registry := newSocketRegistry()

	application.Use("/ws", func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	application.Get("/ws/:endpoint", func(c fiber.Ctx) error {
		if !validEndpointID(c.Params("endpoint")) {
			return c.SendStatus(http.StatusNotFound)
		}
		return c.Next()
	}, socketio.New(func(kws *socketio.Websocket) {
		endpointID := kws.Params("endpoint")
		kws.SetAttribute("endpointID", endpointID)
		registry.add(endpointID, kws.UUID)
		slog.Info("websocket connected", "endpoint_id", endpointID)
	}))

	socketio.On(socketio.EventDisconnect, func(ep *socketio.EventPayload) {
		registry.remove(ep.Kws.UUID)
	})

	socketio.On(socketio.EventClose, func(ep *socketio.EventPayload) {
		registry.remove(ep.Kws.UUID)
	})

	// HTTP handling

	application.Get("/api/health", func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	application.Get("/api/debug", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"host":         string(c.Request().Host()),
			"isProduction": isProduction,
			"requests":     database.CountRequests(c.Context()),
			"sockets":      registry.count(),
		})
	})

	application.Get("/api/endpoints/:endpoint/requests", func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		if !validEndpointID(endpointID) {
			return c.SendStatus(http.StatusNotFound)
		}
		search := c.Query("search")
		requests := database.GetRequestsForEndpointID(c.Context(), endpointID, search, 128)
		return c.JSON(fiber.Map{
			"requests": requests,
		})
	})

	application.Delete("/api/endpoints/:endpoint/requests", func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		if !validEndpointID(endpointID) {
			return c.SendStatus(http.StatusNotFound)
		}
		database.DeleteRequestsForEndpointID(c.Context(), endpointID)
		return c.SendStatus(http.StatusOK)
	})

	application.Delete("/api/endpoints/:endpoint/requests/:request", func(c fiber.Ctx) error {
		if !validEndpointID(c.Params("endpoint")) {
			return c.SendStatus(http.StatusNotFound)
		}
		requestUUID := c.Params("request")
		database.DeleteRequestForUUID(c.Context(), requestUUID)
		return c.SendStatus(http.StatusOK)
	})

	application.Get("/", func(c fiber.Ctx) error {
		base := pageBaseURL(c)
		return c.Render("index", fiber.Map{
			"Title":       "httphq: inspect HTTP requests in real time",
			"Description": "Generate a unique URL, point any client at it, and watch every request arrive: method, headers, body, query string, client IP. No account, free forever.",
			"Canonical":   base + "/",
			"SocialImage": base + "/social-card.png",
		})
	})

	application.Get("/contact", func(c fiber.Ctx) error {
		base := pageBaseURL(c)
		return c.Render("contact", fiber.Map{
			"Title":       "Contact | httphq",
			"Description": "Found a bug, have an idea, or want to say hi? Get in touch with the people who build httphq.",
			"Canonical":   base + "/contact",
			"SocialImage": base + "/social-card.png",
		})
	})

	application.Get("/:endpoint", func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		if !validEndpointID(endpointID) {
			return c.SendStatus(http.StatusNotFound)
		}
		host := string(c.Request().Host())
		endpointURL, websocketURL := endpointURLs(c.Scheme(), host, endpointID)
		return c.Render("endpoint", fiber.Map{
			"Title":       endpointID + " | httphq",
			"Description": "Live capture stream for " + endpointID + ". Requests sent to this endpoint appear here in real time and are deleted after 4 hours.",
			// No canonical: endpoint pages are per-user surfaces excluded by
			// robots.txt, and pointing them at a shared URL would be a lie.
			"AppScripts":           true,
			"EndpointID":           endpointID,
			"EndpointURL":          endpointURL,
			"EndpointWebSocketURL": websocketURL,
		})
	})

	application.Post("/endpoint", func(c fiber.Ctx) error {
		endpointID := haikuMaker.Haikunate()
		slog.InfoContext(c.Context(), "endpoint created", "endpoint_id", endpointID)
		return c.Redirect().To("/" + endpointID)
	})

	application.Use("/to/:endpoint", func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		if !validEndpointID(endpointID) {
			return c.SendStatus(http.StatusNotFound)
		}

		requestUUID := uuid.NewString()

		ip := resolveClientIP(c)

		method := c.Method()

		path := c.Path()

		queryString := string(c.Request().URI().QueryString())

		body := c.Body()

		headers := c.GetReqHeaders()

		if spoofCurl, ok := headers["Httphq-Spoof-Curl"]; ok && len(spoofCurl) > 0 && spoofCurl[0] == "true" {
			delete(headers, "Accept-Encoding")
			delete(headers, "Accept-Language")
			delete(headers, "Connection")
			delete(headers, "Httphq-Spoof-Curl")
			delete(headers, "Origin")
			delete(headers, "Referer")
			delete(headers, "Sec-Fetch-Dest")
			delete(headers, "Sec-Fetch-Mode")
			delete(headers, "Sec-Fetch-Site")
			delete(headers, "Sec-Ch-Ua")
			delete(headers, "Sec-Ch-Ua-Mobile")
			delete(headers, "Sec-Ch-Ua-Platform")

			headers["Content-Type"] = []string{"application/x-www-form-urlencoded"}
			headers["User-Agent"] = []string{"curl/7.79.1"}
		}
		for k := range headers {
			if omitHeader(k) {
				delete(headers, k)
			}
		}

		// Flatten []string headers to a {key: scalar-or-array} JSON; the UI
		// expects either string or string[] per RFC 7230 ambiguity.
		flatHeaders := make(map[string]any, len(headers))
		for k, vs := range headers {
			if len(vs) == 1 {
				flatHeaders[k] = vs[0]
			} else {
				flatHeaders[k] = vs
			}
		}
		jsonHeaders, err := json.Marshal(flatHeaders)
		if err != nil {
			slog.ErrorContext(c.Context(), "request header marshal failed", "err", err)
		}

		request := database.Request{
			UUID:        requestUUID,
			EndpointID:  endpointID,
			IP:          ip,
			Method:      method,
			Path:        path,
			QueryString: queryString,
			Body:        string(body),
			Headers:     datatypes.JSON(jsonHeaders),
		}
		database.CreateRequest(c.Context(), &request)

		// Fan-out to subscribed WS clients. Marshal once, then dispatch
		// asynchronously so a slow client doesn't stall the capture handler.
		if uuids := registry.uuidsFor(endpointID); len(uuids) > 0 {
			marshalled, marshalErr := json.Marshal(request)
			if marshalErr != nil {
				slog.ErrorContext(c.Context(), "websocket payload marshal failed", "err", marshalErr)
			} else {
				go socketio.EmitToList(uuids, marshalled)
			}
		}

		c.Set("Httphq-Request-Uuid", requestUUID)
		return c.SendStatus(http.StatusOK)
	})

	application.Use(func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNotFound)
	})

	host := "localhost:"
	if isProduction {
		host = ":"
	}
	address := host + strconv.Itoa(port)
	slog.Info("server listening", "address", address)
	if err := application.Listen(address); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
