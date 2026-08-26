// Package logging configures structured logging for httphq.
//
// Every log line is a single JSON object written to stdout, using
// OpenTelemetry attribute names (service.name, deployment.environment, ...) so
// the output stays portable across log backends. Init wires slog's default
// logger; request-scoped code passes a context.Context carrying the request
// ID, which is stamped onto every record for correlation.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey int

const requestIDKey ctxKey = iota

// WithRequestID returns a copy of ctx carrying id, so any log emitted
// downstream is correlated to the same request.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// ctxHandler wraps an slog.Handler and stamps request_id (read from the
// context) onto every record, so a log call deep inside a handler stays
// correlated without threading a logger through every signature.
type ctxHandler struct{ slog.Handler }

func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestIDFrom(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ctxHandler{h.Handler.WithAttrs(attrs)}
}

func (h ctxHandler) WithGroup(name string) slog.Handler {
	return ctxHandler{h.Handler.WithGroup(name)}
}

// sensitiveKeys are masked wherever they appear as a log attribute key. Access
// logs are already built from a fixed, safe field set; this is a backstop
// against secrets reaching ad-hoc application logs.
var sensitiveKeys = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"set-cookie":    {},
	"x-api-key":     {},
	"api_key":       {},
	"password":      {},
	"token":         {},
	"secret":        {},
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	// Lower-case the level (INFO -> info) so it matches the other services.
	if len(groups) == 0 && a.Key == slog.LevelKey {
		return slog.String(slog.LevelKey, strings.ToLower(a.Value.String()))
	}
	if _, ok := sensitiveKeys[strings.ToLower(a.Key)]; ok {
		return slog.String(a.Key, "[redacted]")
	}
	return a
}

// logLevels are the values LOG_LEVEL accepts, matched case-insensitively.
// Anything outside this set is not a level, which is what lets an unrecognised
// override fall through to the environment's own default.
var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// resolveLevel picks the log level for an environment: info in production, so
// a deployment is not paying to record its own debug chatter, and debug
// everywhere else. An operator overrides either with LOG_LEVEL; an unrecognised
// value leaves the environment's default in place.
func resolveLevel(env, override string) slog.Level {
	if level, ok := logLevels[strings.ToLower(override)]; ok {
		return level
	}
	if env == "production" {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}

// Init installs the process-wide slog logger: JSON to stdout, an OTel-named
// service.name/deployment.environment base, request_id stamping and
// sensitive-key redaction.
func Init(service, env string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       resolveLevel(env, os.Getenv("LOG_LEVEL")),
		ReplaceAttr: replaceAttr,
	})
	logger := slog.New(ctxHandler{handler}).With(
		slog.String("service.name", service),
		slog.String("deployment.environment", env),
	)
	slog.SetDefault(logger)
}
