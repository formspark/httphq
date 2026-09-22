package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"httphq/src/database"
)

// storeCapture writes a capture for an endpoint at a chosen age. GORM only
// stamps CreatedAt when it is zero, so passing one keeps it.
func storeCapture(t *testing.T, endpointID, uuid string, createdAt time.Time) {
	t.Helper()
	database.CreateRequest(t.Context(), &database.Request{
		UUID:       uuid,
		EndpointID: endpointID,
		Method:     "POST",
		Path:       "/",
		CreatedAt:  createdAt,
	})
}

// storedUUIDs is what the database still holds for an endpoint. Named for the
// store rather than the endpoint so it does not read as the socket registry's
// uuidsFor, which answers a different question about the same ID.
func storedUUIDs(ctx context.Context, endpointID string) []string {
	var uuids []string
	for _, request := range database.GetRequestsForEndpointID(ctx, endpointID, "", time.Time{}, 10) {
		uuids = append(uuids, request.UUID)
	}
	return uuids
}

// newApplication builds the whole routing surface, so these cover what the
// wiring itself decides rather than what any one handler does.
func TestNewApplication(t *testing.T) {
	t.Run("serves static files from the public directory", func(t *testing.T) {
		response := get(t, "/robots.txt")

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, bodyOf(t, response), "User-agent")
	})

	// The fallthrough runs after every route, including the prefix-matched
	// capture surface, so a path that matches nothing must not be captured.
	t.Run("an unmatched path is a 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(t, "/no/such/page").StatusCode)
	})
}

func TestSweepRetention(t *testing.T) {
	t.Run("drops captures older than the retention window", func(t *testing.T) {
		endpointID := "sweep-old"
		storeCapture(t, endpointID, "sweep-old-expired",
			time.Now().Add(-retentionWindow).Add(-time.Minute))

		sweepRetention()

		assert.Empty(t, storedUUIDs(t.Context(), endpointID))
	})

	t.Run("keeps captures inside the retention window", func(t *testing.T) {
		endpointID := "sweep-recent"
		storeCapture(t, endpointID, "sweep-recent-live",
			time.Now().Add(-retentionWindow).Add(time.Minute))

		sweepRetention()

		assert.Equal(t, []string{"sweep-recent-live"}, storedUUIDs(t.Context(), endpointID))
	})
}

// Cron's first tick is a full interval away, so a process restarting more often
// than the interval would never sweep and captures would outlive the window for
// as long as the database file does.
func TestStartRetentionSweep(t *testing.T) {
	t.Run("sweeps at startup rather than waiting for the first tick", func(t *testing.T) {
		endpointID := "sweep-boot"
		storeCapture(t, endpointID, "sweep-boot-expired",
			time.Now().Add(-retentionWindow).Add(-time.Minute))

		scheduler, err := startRetentionSweep()
		require.NoError(t, err)
		t.Cleanup(func() { scheduler.Stop() })

		assert.Empty(t, storedUUIDs(t.Context(), endpointID))
	})
}

// startRun runs the whole process on a loopback port against a store of its
// own, and waits until it answers. It returns the base URL, the cancel that
// stands in for SIGTERM, and where run's result arrives. The package's shared
// store is put back when the test ends.
func startRun(t *testing.T) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	previous := database.DB
	t.Cleanup(func() { database.DB = previous })

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	result := make(chan error, 1)
	dsn := "file:" + filepath.Join(t.TempDir(), "run.db")
	go func() {
		result <- run(ctx, listener, dsn, applicationConfig{
			viewsDir:  "./views",
			publicDir: "../public",
			registry:  newSocketRegistry(),
		})
	}()

	base := "http://" + listener.Addr().String()
	require.Eventually(t, func() bool { return healthy(base) }, 5*time.Second, 20*time.Millisecond)
	return base, cancel, result
}

// healthy reports whether the server at base answers its health check.
func healthy(base string) bool {
	response, err := http.Get(base + "/api/health")
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return response.StatusCode == http.StatusOK
}

// awaitRun is what run returned, failing the test if it never returns.
func awaitRun(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(shutdownTimeout + 5*time.Second):
		t.Fatal("run did not return after its context was cancelled")
		return nil
	}
}

// startCapture opens a capture whose body is only half sent, so the request is
// still in flight until finishCapture sends the rest.
func startCapture(t *testing.T, base string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp4", base[len("http://"):])
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	_, err = io.WriteString(conn, "POST /to/run-drain HTTP/1.1\r\nHost: localhost\r\n"+
		"Content-Type: text/plain\r\nContent-Length: 4\r\n\r\nab")
	require.NoError(t, err)
	return conn
}

// finishCapture sends the rest of the body and returns the status line.
func finishCapture(t *testing.T, conn net.Conn) string {
	t.Helper()
	_, err := io.WriteString(conn, "cd")
	require.NoError(t, err)
	status, err := bufio.NewReader(conn).ReadString('\n')
	require.NoError(t, err)
	return status
}

// run is what main does once it has read the environment and bound the port,
// so these cover the process lifecycle rather than any one request.
func TestRun(t *testing.T) {
	t.Run("serves until its context is cancelled, then stops accepting connections", func(t *testing.T) {
		base, cancel, result := startRun(t)

		cancel()

		require.NoError(t, awaitRun(t, result))
		assert.False(t, healthy(base))
	})

	// The point of the SIGTERM handler: a request that arrived before the
	// signal is answered rather than cut.
	t.Run("answers a request already in flight before it returns", func(t *testing.T) {
		base, cancel, result := startRun(t)
		conn := startCapture(t, base)

		cancel()
		time.Sleep(200 * time.Millisecond)
		assert.Empty(t, result, "run returned with a request still open")

		assert.Contains(t, finishCapture(t, conn), "200")
		require.NoError(t, awaitRun(t, result))
	})

	t.Run("returns the store's error without serving", func(t *testing.T) {
		previous := database.DB
		t.Cleanup(func() { database.DB = previous })
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })

		err = run(t.Context(), listener, "file:"+filepath.Join(t.TempDir(), "missing", "run.db"),
			applicationConfig{viewsDir: "./views", publicDir: "../public", registry: newSocketRegistry()})

		require.Error(t, err)
	})
}
