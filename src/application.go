package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/template/html/v2"
	"github.com/robfig/cron/v3"

	"httphq/src/database"
	"httphq/src/logging"
)

const (
	port      = 8080
	bodyLimit = 1 << 20 // 1 MiB

	// Captured data is ephemeral by design. The window is stated on the landing
	// page and on every endpoint page; changing it here changes a promise.
	retentionWindow = 4 * time.Hour
	retentionSweep  = "*/5 * * * *"

	// Development runs effectively unlimited so a page under test is never
	// throttled; production bounds one client's share of a shared instance.
	// The production figure is quoted to pollers in the agent prompt, so it is
	// read from here rather than restated: see agentPrompt.
	developmentRequestsPerMinute = 9999
	productionRequestsPerMinute  = 150
)

// applicationConfig locates the files the app serves and the socket registry it
// fans captures out to. The paths are arguments rather than constants so the
// process can be driven from a directory other than the repository root.
type applicationConfig struct {
	viewsDir  string
	publicDir string
	registry  *socketRegistry
}

// newApplication builds the fully wired HTTP application. Everything it needs
// beyond the database is passed in, so the whole routing surface can be
// exercised without a listening socket.
func newApplication(config applicationConfig) *fiber.App {
	engine := html.New(config.viewsDir, ".html")
	engine.AddFunc("asset", newAssetIndex(config.publicDir, !isProduction).url)
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

	maxRequestsPerMinute := developmentRequestsPerMinute
	if isProduction {
		maxRequestsPerMinute = productionRequestsPerMinute
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
	application.Use(securityHeaders(contentSecurityPolicy(!isProduction)))

	application.Get("/*", static.New(config.publicDir))

	registerWebSockets(application, config.registry)

	application.Get("/api/health", handleHealth)
	application.Get("/api/debug", handleDebug(config.registry))
	application.Get("/api/endpoints/:endpoint/requests", requireValidEndpoint, handleListRequests)
	application.Delete("/api/endpoints/:endpoint/requests", requireValidEndpoint, handleDeleteRequests)
	application.Delete("/api/endpoints/:endpoint/requests/:request", requireValidEndpoint, handleDeleteRequest)

	application.Get("/", renderIndex)
	application.Get("/contact", renderContact)
	application.Get("/:endpoint", requireValidEndpoint, renderEndpoint)
	application.Post("/endpoint", createEndpoint(haikunator.New()))

	// Prefix-matched so everything after the endpoint ID is captured as the
	// request path, and method-agnostic because any method is a valid capture.
	application.Use("/to/:endpoint", requireValidEndpoint, captureRequest(config.registry))

	application.Use(func(c fiber.Ctx) error {
		return c.SendStatus(http.StatusNotFound)
	})

	return application
}

// sweepRetention drops every capture older than the retention window.
func sweepRetention() {
	database.DeleteOldRequests(context.Background(), time.Now().Add(-retentionWindow))
}

// startRetentionSweep sweeps once and then on a schedule. The returned
// scheduler runs for the life of the process; there is no shutdown path that
// stops it.
//
// The sweep at startup is what makes the window hold: cron's first tick is a
// full interval away, so a process that restarts more often than the interval
// would otherwise never sweep at all and captures would outlive the window for
// as long as the database file does.
func startRetentionSweep() *cron.Cron {
	sweepRetention()
	scheduler := cron.New()
	if _, err := scheduler.AddFunc(retentionSweep, sweepRetention); err != nil {
		slog.Error("cron job registration failed", "err", err)
		os.Exit(1)
	}
	scheduler.Start()
	return scheduler
}

func main() {
	env := os.Getenv("APPLICATION_ENV")
	if env == "" {
		env = "development"
	}
	logging.Init("httphq", env)

	currentPlatform = resolvePlatform(os.Getenv("PLATFORM"))
	slog.Info("platform resolved",
		"platform", os.Getenv("PLATFORM"), "ip_header", currentPlatform.ipHeader)

	database.Connect("file:local.db?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")

	startRetentionSweep()

	application := newApplication(applicationConfig{
		viewsDir:  "./src/views",
		publicDir: "./public",
		registry:  newSocketRegistry(),
	})

	// Development binds loopback only, so a work-in-progress capture surface is
	// not reachable from the network the machine happens to be on.
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
