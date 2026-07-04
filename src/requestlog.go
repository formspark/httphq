package main

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"httphq/src/logging"
)

// requestIDPattern bounds an inbound X-Request-Id so an untrusted caller can't
// inject oversized or high-cardinality values into the logs.
var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// probePaths are hit by the platform's liveness/readiness health checks; their
// access log drops to debug so probe traffic stays out of production logs.
var probePaths = map[string]struct{}{
	"/api/health": {},
}

// requestLogger assigns each request a correlation ID — reusing a valid
// inbound X-Request-Id, otherwise minting one — exposes it on the context and
// the X-Request-Id response header, and emits one structured access-log line.
// Request headers and bodies are never logged, and the path is logged without
// its query string.
func requestLogger(c fiber.Ctx) error {
	id := c.Get("X-Request-Id")
	if !requestIDPattern.MatchString(id) {
		id = uuid.NewString()
	}
	c.SetContext(logging.WithRequestID(c.Context(), id))
	c.Set("X-Request-Id", id)

	start := time.Now()
	err := c.Next()

	level := slog.LevelInfo
	if _, isProbe := probePaths[c.Path()]; isProbe {
		level = slog.LevelDebug
	}
	attrs := []slog.Attr{
		slog.String("http.request.method", c.Method()),
		slog.String("url.path", c.Path()),
		slog.Int("http.response.status_code", c.Response().StatusCode()),
		slog.Float64("http.server.request.duration", time.Since(start).Seconds()),
	}
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		level = slog.LevelError
	}
	slog.LogAttrs(c.Context(), level, "request", attrs...)
	return err
}
