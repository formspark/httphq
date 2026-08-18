package main

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"httphq/src/logging"
)

// headerRequestID carries the correlation ID in both directions: read from the
// caller when it supplies one, and echoed so a caller can quote it in a report.
const headerRequestID = "X-Request-Id"

// requestIDPattern bounds an inbound X-Request-Id so an untrusted caller can't
// inject oversized or high-cardinality values into the logs.
var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// probePaths are hit by the platform's liveness/readiness health checks; their
// access log drops to debug so probe traffic stays out of production logs.
var probePaths = map[string]struct{}{
	"/api/health": {},
}

// responseStatus is the status the caller will be answered with.
//
// A handler that refuses a request returns an error rather than writing a
// status, and Fiber's error handler turns that into a response only after this
// middleware has unwound. Reading the response directly at that point yields
// the untouched default, so every refusal would be logged as a success.
func responseStatus(c fiber.Ctx, err error) int {
	if err == nil {
		return c.Response().StatusCode()
	}
	var fiberError *fiber.Error
	if errors.As(err, &fiberError) {
		return fiberError.Code
	}
	return http.StatusInternalServerError
}

// requestLogger assigns each request a correlation ID, reusing a valid inbound
// X-Request-Id and otherwise minting one. It exposes that ID on the context and
// the X-Request-Id response header, and emits one structured access-log line.
// Request headers and bodies are never logged, and the path is logged without
// its query string.
//
// The response header is written on the way out for the same reason the
// security headers are: a handler that resets the response it is building would
// otherwise drop it. See securityHeaders.
func requestLogger(c fiber.Ctx) error {
	id := c.Get(headerRequestID)
	if !requestIDPattern.MatchString(id) {
		id = uuid.NewString()
	}
	c.SetContext(logging.WithRequestID(c.Context(), id))

	start := time.Now()
	err := c.Next()
	c.Set(headerRequestID, id)

	level := slog.LevelInfo
	if _, isProbe := probePaths[c.Path()]; isProbe {
		level = slog.LevelDebug
	}
	attrs := []slog.Attr{
		slog.String("http.request.method", c.Method()),
		slog.String("url.path", c.Path()),
		slog.Int("http.response.status_code", responseStatus(c, err)),
		slog.Float64("http.server.request.duration", time.Since(start).Seconds()),
	}
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		level = slog.LevelError
	}
	slog.LogAttrs(c.Context(), level, "request", attrs...)
	return err
}
