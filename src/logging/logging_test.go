package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capture builds a logger writing to a buffer through the same handler stack
// Init installs, and returns a reader for the one record it emitted.
func capture(t *testing.T, log func(logger *slog.Logger)) map[string]any {
	t.Helper()

	var out bytes.Buffer
	handler := slog.NewJSONHandler(&out, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	})
	log(slog.New(ctxHandler{handler}))

	var record map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &record), "log output should be one JSON object")
	return record
}

func TestWithRequestID(t *testing.T) {
	t.Run("stamps the ID onto a record logged with the context", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.InfoContext(WithRequestID(context.Background(), "req-1"), "hello")
		})

		assert.Equal(t, "req-1", record["request_id"])
	})

	t.Run("adds no field when the context carries no ID", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.InfoContext(context.Background(), "hello")
		})

		assert.NotContains(t, record, "request_id")
	})

	// Groups and attribute sets are re-wrapped so the stamping survives, and a
	// logger built with .With() stays correlated.
	t.Run("survives a logger built with extra attributes", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.With(slog.String("service.name", "httphq")).
				InfoContext(WithRequestID(context.Background(), "req-2"), "hello")
		})

		assert.Equal(t, "req-2", record["request_id"])
		assert.Equal(t, "httphq", record["service.name"])
	})

	// A grouped logger keeps stamping, but the attribute lands inside the open
	// group like every other one. Nothing in httphq groups its logs; a caller
	// that starts to must read request_id from the group rather than the root.
	t.Run("survives a grouped logger", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.WithGroup("http").
				InfoContext(WithRequestID(context.Background(), "req-3"), "hello", "method", "GET")
		})

		assert.Equal(t, map[string]any{"method": "GET", "request_id": "req-3"}, record["http"])
	})
}

func TestResolveLevel(t *testing.T) {
	t.Run("production records info and above, other environments record everything", func(t *testing.T) {
		assert.Equal(t, slog.LevelInfo, resolveLevel("production", ""))
		assert.Equal(t, slog.LevelDebug, resolveLevel("development", ""))
		assert.Equal(t, slog.LevelDebug, resolveLevel("", ""))
	})

	t.Run("an operator override wins in either environment", func(t *testing.T) {
		overrides := map[string]slog.Level{
			"debug": slog.LevelDebug,
			"INFO":  slog.LevelInfo,
			"warn":  slog.LevelWarn,
			"Error": slog.LevelError,
		}

		for override, want := range overrides {
			assert.Equalf(t, want, resolveLevel("production", override), "LOG_LEVEL=%s", override)
			assert.Equalf(t, want, resolveLevel("development", override), "LOG_LEVEL=%s", override)
		}
	})

	// A typo must not silence a production deployment.
	t.Run("an unrecognised override leaves the environment default", func(t *testing.T) {
		assert.Equal(t, slog.LevelInfo, resolveLevel("production", "verbose"))
		assert.Equal(t, slog.LevelDebug, resolveLevel("development", "verbose"))
	})
}

func TestReplaceAttr(t *testing.T) {
	t.Run("lower-cases the level so it matches neighbouring services", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.WarnContext(context.Background(), "hello")
		})

		assert.Equal(t, "warn", record["level"])
	})

	// Access logs are built from a fixed, safe field set; this is the backstop
	// for an ad-hoc log line that names a secret.
	t.Run("masks a sensitive key wherever it appears", func(t *testing.T) {
		record := capture(t, func(logger *slog.Logger) {
			logger.InfoContext(context.Background(), "hello",
				"Authorization", "Bearer sk_live_1",
				"cookie", "session=1",
				"Token", "t",
				"password", "hunter2",
				"url.path", "/to/purple-frog-0691")
		})

		for _, key := range []string{"Authorization", "cookie", "Token", "password"} {
			assert.Equalf(t, "[redacted]", record[key], "%s must never reach the logs", key)
		}
		assert.Equal(t, "/to/purple-frog-0691", record["url.path"], "ordinary fields must survive")
	})
}
