package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// accessLogFor drives one request with the process logger pointed at a buffer
// and returns the access-log line it produced. The logger is process-wide, so
// it is restored before the next test reads it.
//
// The handler is a plain JSON handler rather than the one Init installs, so
// levels appear as slog spells them. Rendering them lower-case is the logging
// package's job and is covered there.
func accessLogFor(t *testing.T, spec testRequest) map[string]any {
	t.Helper()

	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	do(t, spec)

	// A request may log more than its access line, so the line is found by its
	// message rather than by position.
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		if record["msg"] == "request" {
			return record
		}
	}
	t.Fatalf("no access-log line in %q", out.String())
	return nil
}

// An inbound request ID is echoed and stamped on every log line, so its shape
// bounds what an untrusted caller can put in the logs.
func TestRequestIDPattern(t *testing.T) {
	t.Run("accepts the shapes a caller legitimately sends", func(t *testing.T) {
		accepted := []string{
			"70c33dcb-ee76-447d-a82f-26e3a6581010",
			"abcd1234",
			"TRACE-0001",
			strings.Repeat("a", 64),
		}

		for _, id := range accepted {
			assert.Truef(t, requestIDPattern.MatchString(id), "%q should be reused", id)
		}
	})

	t.Run("rejects anything oversized, empty or able to shape a log line", func(t *testing.T) {
		rejected := []string{
			"",
			"short",
			strings.Repeat("a", 65),
			"has space",
			"new\nline",
			`{"level":"error"}`,
			"under_score",
			"semi;colon",
		}

		for _, id := range rejected {
			assert.Falsef(t, requestIDPattern.MatchString(id), "%q should be replaced with a minted ID", id)
		}
	})
}

// The ID a caller reads back is the one stamped on every line logged while its
// request was handled, so the echo is what makes a log line findable from
// outside the process.
func TestRequestLogger(t *testing.T) {
	t.Run("a valid inbound request ID is echoed back", func(t *testing.T) {
		response := do(t, testRequest{
			method:  http.MethodGet,
			path:    "/api/health",
			headers: map[string]string{"X-Request-Id": "abcd-1234-efgh"},
		})

		assert.Equal(t, "abcd-1234-efgh", response.Header.Get("X-Request-Id"))
	})

	t.Run("a request with no ID is given one", func(t *testing.T) {
		assert.NotEmpty(t, get(t, "/api/health").Header.Get("X-Request-Id"))
	})

	// A caller's ID reaches every log line it produces, so one shaped like a log
	// record of its own is replaced rather than reused.
	t.Run("a malformed inbound request ID is replaced rather than echoed", func(t *testing.T) {
		forged := `{"level":"error"}`

		response := do(t, testRequest{
			method:  http.MethodGet,
			path:    "/api/health",
			headers: map[string]string{"X-Request-Id": forged},
		})

		echoed := response.Header.Get("X-Request-Id")
		assert.NotEmpty(t, echoed)
		assert.NotEqual(t, forged, echoed)
	})
}

// The status is what an operator reads a log for, and it is the one field the
// response cannot be asked for directly while a refusal is still unwinding.
func TestResponseStatus(t *testing.T) {
	t.Run("a served request logs the status it was answered with", func(t *testing.T) {
		record := accessLogFor(t, testRequest{method: http.MethodGet, path: "/api/health"})

		assert.Equal(t, float64(http.StatusOK), record["http.response.status_code"])
	})

	t.Run("a refused request logs its refusal, not a success", func(t *testing.T) {
		// The socket route refuses a request that is not an upgrade, which is
		// the shape every Fiber error takes: returned, not written.
		record := accessLogFor(t, testRequest{
			method: http.MethodGet, path: "/ws/purple-frog-0691",
		})

		assert.Equal(t, float64(http.StatusUpgradeRequired), record["http.response.status_code"])
	})

	t.Run("a handler that writes its own status is logged unchanged", func(t *testing.T) {
		record := accessLogFor(t, testRequest{method: http.MethodGet, path: "/no/such/page"})

		assert.Equal(t, float64(http.StatusNotFound), record["http.response.status_code"])
	})
}

// Probe traffic is constant and says nothing, so it logs at debug and stays out
// of production logs. Everything else has to remain visible.
func TestProbePaths(t *testing.T) {
	t.Run("only probe traffic is demoted", func(t *testing.T) {
		_, isProbe := probePaths["/api/health"]
		assert.True(t, isProbe)

		for _, path := range []string{"/", "/contact", "/api/debug", "/to/purple-frog-0691"} {
			_, demoted := probePaths[path]
			assert.Falsef(t, demoted, "%s carries real traffic and must be logged at info", path)
		}
	})

	t.Run("a probe logs below the level a production deployment records", func(t *testing.T) {
		record := accessLogFor(t, testRequest{method: http.MethodGet, path: "/api/health"})

		assert.Equal(t, slog.LevelDebug.String(), record["level"])
	})

	t.Run("real traffic stays visible", func(t *testing.T) {
		record := accessLogFor(t, testRequest{method: http.MethodGet, path: "/contact"})

		assert.Equal(t, slog.LevelInfo.String(), record["level"])
	})

	t.Run("a refusal is loud", func(t *testing.T) {
		record := accessLogFor(t, testRequest{
			method: http.MethodGet, path: "/ws/purple-frog-0691",
		})

		assert.Equal(t, slog.LevelError.String(), record["level"])
	})
}
