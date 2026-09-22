package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

	// WAL lets the retention sweep delete while captures are written, and the
	// busy timeout rides out the lock one takes rather than failing the other.
	databaseDSN = "file:local.db?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"

	// shutdownTimeout bounds the drain after SIGTERM. Kubernetes sends SIGKILL
	// 30 seconds after SIGTERM; a request still open after this is cut, which
	// leaves time to stop the sweep and close the store before the kill.
	shutdownTimeout = 10 * time.Second

	// Development runs effectively unlimited so a page under test is never
	// throttled; production bounds one client's share of a shared instance.
	// The production figure is quoted to pollers in the agent prompt, so it is
	// read from here rather than restated: see agentPrompt.
	developmentRequestsPerMinute = 9999
	productionRequestsPerMinute  = 150
)

// readMethods are the methods a page, a static file or a read-only API route
// answers. HEAD is registered beside GET rather than left to Fiber's automatic
// HEAD routes: Fiber appends those when the server starts, behind the catch-all
// 404 at the end of newApplication, so the catch-all would answer every HEAD.
var readMethods = []string{fiber.MethodGet, fiber.MethodHead}

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
	// Held rather than passed straight to the engine: the page handlers build
	// the absolute social image URL in Go, so they need the same index the
	// templates resolve their own asset references through.
	assets := newAssetIndex(config.publicDir, !isProduction)
	engine.AddFunc("asset", assets.url)
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

	// Runs first so every request, including the rate-limited ones, gets a
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

	application.Add(readMethods, "/*", static.New(config.publicDir))

	registerWebSockets(application, config.registry)

	application.Add(readMethods, "/api/health", handleHealth)
	application.Add(readMethods, "/api/debug", handleDebug(config.registry))
	application.Add(readMethods, "/api/endpoints/:endpoint/requests", requireValidEndpoint, handleListRequests)
	application.Delete("/api/endpoints/:endpoint/requests", requireValidEndpoint, handleDeleteRequests)
	application.Delete("/api/endpoints/:endpoint/requests/:request", requireValidEndpoint, handleDeleteRequest)

	application.Add(readMethods, "/", renderIndex(assets))
	application.Add(readMethods, "/contact", renderContact(assets))
	application.Add(readMethods, "/:endpoint", requireValidEndpoint, renderEndpoint(assets))
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

// startRetentionSweep sweeps once and then on a schedule, until the returned
// scheduler is stopped.
//
// The sweep at startup is what makes the window hold: cron's first tick is a
// full interval away, so a process that restarts more often than the interval
// would otherwise never sweep at all and captures would outlive the window for
// as long as the database file does.
func startRetentionSweep() (*cron.Cron, error) {
	sweepRetention()
	scheduler := cron.New()
	if _, err := scheduler.AddFunc(retentionSweep, sweepRetention); err != nil {
		return nil, fmt.Errorf("schedule the retention sweep: %w", err)
	}
	scheduler.Start()
	return scheduler, nil
}

// applicationEnv is the environment the process runs in. Unset means a
// developer's machine, never production.
func applicationEnv() string {
	if env := os.Getenv("APPLICATION_ENV"); env != "" {
		return env
	}
	return "development"
}

// listenAddress is where the server binds. Development binds loopback only, so
// a work-in-progress capture surface is not reachable from the network the
// machine happens to be on.
func listenAddress() string {
	host := "localhost:"
	if isProduction {
		host = ":"
	}
	return host + strconv.Itoa(port)
}

// run opens the store, starts the retention sweep and serves the application on
// listener until ctx is cancelled. It then drains the server, stops the sweep
// and closes the store, in that order, so nothing is left writing to a closed
// store.
func run(ctx context.Context, listener net.Listener, dsn string, config applicationConfig) error {
	if _, err := database.Connect(dsn); err != nil {
		return err
	}
	scheduler, err := startRetentionSweep()
	if err != nil {
		return errors.Join(err, database.Close())
	}
	serveErr := serve(ctx, newApplication(config), listener)
	// Stop returns a context that is done once a sweep already under way has
	// finished, so the store is not closed underneath it.
	<-scheduler.Stop().Done()
	return errors.Join(serveErr, database.Close())
}

// serve answers requests until ctx is cancelled, then stops accepting
// connections and waits up to shutdownTimeout for the requests in flight.
//
// The listener is closed after the shutdown as well. A signal that lands before
// the server has taken the listener finds nothing to shut down, and the server
// would then serve forever; closing it makes that late start fail at once.
func serve(ctx context.Context, application *fiber.App, listener net.Listener) error {
	served := make(chan error, 1)
	go func() {
		served <- application.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	slog.Info("shutting down", "timeout", shutdownTimeout.String())
	shutdownErr := application.ShutdownWithTimeout(shutdownTimeout)
	_ = listener.Close()
	return errors.Join(shutdownErr, <-served)
}

func main() {
	logging.Init("httphq", applicationEnv())

	currentPlatform = resolvePlatform(os.Getenv("PLATFORM"))
	slog.Info("platform resolved",
		"platform", os.Getenv("PLATFORM"), "ip_header", currentPlatform.ipHeader)

	// tcp4 is the network Fiber listened on by default before the listener
	// moved here.
	address := listenAddress()
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		exit("cannot listen", err)
	}
	slog.Info("server listening", "address", address)

	// Kubernetes sends SIGTERM before it kills a pod. Without the handler the
	// process dies on the spot and takes every request in flight with it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	err = run(ctx, listener, databaseDSN, applicationConfig{
		viewsDir:  "./src/views",
		publicDir: "./public",
		registry:  newSocketRegistry(),
	})
	stop()
	if err != nil {
		exit("server exited", err)
	}
	slog.Info("server stopped")
}

// exit logs why the process cannot go on and ends it.
func exit(message string, err error) {
	slog.Error(message, "err", err)
	os.Exit(1)
}
