package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
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

// Headers stripped from captured requests so users see their original payload,
// not infrastructure-added headers.
var omittedHeaders = [...]string{
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

// resolveClientIP picks the most trustworthy client IP available. Cloudflare
// overwrites CF-Connecting-IP on every request, so it is the only header an
// upstream client cannot spoof — we prefer it. We then fall back through
// X-Real-Ip, X-Forwarded-For (leftmost) and the TCP peer for non-Cloudflare
// deployments, verifying each parses as an IP.
func resolveClientIP(c fiber.Ctx) string {
	if v := c.Get("Cf-Connecting-Ip"); v != "" {
		if ip := net.ParseIP(v); ip != nil {
			return ip.String()
		}
	}
	if v := c.Get("X-Real-Ip"); v != "" {
		if ip := net.ParseIP(v); ip != nil {
			return ip.String()
		}
	}
	if xff := c.Get(fiber.HeaderXForwardedFor); xff != "" {
		// leftmost
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				if ip := net.ParseIP(trimSpace(xff[:i])); ip != nil {
					return ip.String()
				}
				break
			}
		}
		if ip := net.ParseIP(trimSpace(xff)); ip != nil {
			return ip.String()
		}
	}
	peer := c.IP()
	if ip := net.ParseIP(peer); ip != nil {
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

	if !isProduction {
		engine.Reload(true)
		engine.Debug(true)
	}

	application := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/main",
		BodyLimit:   bodyLimit,
		// Trust the upstream proxy (Traefik in cluster) so c.IP, the limiter, and
		// header-based extraction reflect the real client. Pod CIDRs vary per
		// cluster; restrict via env if you need a tighter list.
		TrustProxy:  true,
		ProxyHeader: fiber.HeaderXForwardedFor,
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
		// CSP policy:
		// - script-src needs 'unsafe-eval' for Alpine (it compiles directive
		//   expressions via the Function constructor) and the Tailwind Play
		//   CDN (compiles utility classes at runtime). All page scripts are
		//   external so script-src does NOT need 'unsafe-inline'.
		// - style-src needs 'unsafe-inline' because Tailwind Play CDN injects
		//   generated styles into <style> tags, and Alpine x-show toggles via
		//   inline display style.
		c.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-eval' https://unpkg.com https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'")
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
		return c.Render("index", fiber.Map{
			"Title": "httphq",
		})
	})

	application.Get("/contact", func(c fiber.Ctx) error {
		return c.Render("contact", fiber.Map{
			"Title": "Contact | httphq",
		})
	})

	application.Get("/:endpoint", func(c fiber.Ctx) error {
		endpointID := c.Params("endpoint")
		if !validEndpointID(endpointID) {
			return c.SendStatus(http.StatusNotFound)
		}
		host := string(c.Request().Host())
		scheme := c.Scheme()
		websocketScheme := "ws"
		if scheme == "https" {
			websocketScheme = "wss"
		}
		return c.Render("endpoint", fiber.Map{
			"Title":                endpointID + " | httphq",
			"EndpointID":           endpointID,
			"EndpointURL":          scheme + "://" + host + "/to/" + endpointID,
			"EndpointWebSocketURL": websocketScheme + "://" + host + "/ws/" + endpointID,
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
		for _, omittedHeader := range omittedHeaders {
			delete(headers, omittedHeader)
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
