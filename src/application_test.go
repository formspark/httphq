package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"httphq/src/database"
)

// storeCapture writes a capture for an endpoint at a chosen age. GORM only
// stamps CreatedAt when it is zero, so passing one keeps it.
func storeCapture(t *testing.T, endpointID, uuid string, createdAt time.Time) {
	t.Helper()
	require.NoError(t, database.CreateRequest(t.Context(), &database.Request{
		UUID:       uuid,
		EndpointID: endpointID,
		Method:     "POST",
		Path:       "/",
		CreatedAt:  createdAt,
	}))
}

// storedUUIDs is what the database still holds for an endpoint, named for the
// store rather than the endpoint so a reader is not left wondering whether it
// is the socket registry being asked.
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

	// Uptime checks and link checkers ask with HEAD. Each read route answers it
	// with the status and type GET would, and no body.
	t.Run("answers HEAD on every read route as GET would, without a body", func(t *testing.T) {
		id := endpointID(t)
		paths := []string{
			"/",
			"/contact",
			"/" + id,
			"/robots.txt",
			"/api/health",
			"/api/endpoints/" + id + "/requests",
		}
		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				want := get(t, path)
				head := do(t, testRequest{method: http.MethodHead, path: path})

				assert.Equal(t, want.StatusCode, head.StatusCode)
				assert.Equal(t, want.Header.Get("Content-Type"), head.Header.Get("Content-Type"))
				assert.Empty(t, bodyOf(t, head))
			})
		}
	})

	t.Run("answers HEAD on an unmatched path with a 404", func(t *testing.T) {
		head := do(t, testRequest{method: http.MethodHead, path: "/no/such/page"})
		assert.Equal(t, http.StatusNotFound, head.StatusCode)
	})

	t.Run("answers a method no page route takes with a 404", func(t *testing.T) {
		put := do(t, testRequest{method: http.MethodPut, path: "/"})
		assert.Equal(t, http.StatusNotFound, put.StatusCode)
	})
}

// robotsAllowPrefixes reads the Allow rules robots.txt states as plain path
// prefixes. Rules anchored with $ allow one page rather than a family of
// files, so they are left out.
func robotsAllowPrefixes(t *testing.T) []string {
	t.Helper()
	var prefixes []string
	for line := range strings.Lines(bodyOf(t, get(t, "/robots.txt"))) {
		rule, found := strings.CutPrefix(strings.TrimSpace(line), "Allow: ")
		if found && !strings.HasSuffix(rule, "$") {
			prefixes = append(prefixes, rule)
		}
	}
	return prefixes
}

// versionedAssetPattern finds every first-party file a page references
// through `asset`, with or without the absolute base Open Graph needs.
var versionedAssetPattern = regexp.MustCompile(`"(?:http://example\.com)?(/[^"?]+)\?v=`)

// robots.txt is an allowlist, so a file a crawlable page references is
// invisible to a crawler until a rule names it. Google renders the favicon in
// search results only when it can fetch it.
func TestRobots(t *testing.T) {
	t.Run("allows every asset the crawlable pages reference", func(t *testing.T) {
		prefixes := robotsAllowPrefixes(t)
		for _, page := range []string{"/", "/contact"} {
			matches := versionedAssetPattern.FindAllStringSubmatch(bodyOf(t, get(t, page)), -1)
			require.NotEmpty(t, matches)
			for _, match := range matches {
				assert.True(t, slices.ContainsFunc(prefixes, func(prefix string) bool {
					return strings.HasPrefix(match[1], prefix)
				}), "%s references %s, which robots.txt does not allow", page, match[1])
			}
		}
	})
}

func TestNewApplicationTimeouts(t *testing.T) {
	// The limiter only sees requests that finished arriving, so without these a
	// connection that never completes a request, never reads its response or
	// idles on keep-alive holds its slot indefinitely.
	t.Run("bounds how long a connection may read, write and idle", func(t *testing.T) {
		config := application(t).Config()

		assert.Equal(t, readTimeout, config.ReadTimeout)
		assert.Equal(t, writeTimeout, config.WriteTimeout)
		assert.Equal(t, idleTimeout, config.IdleTimeout)
		assert.Positive(t, readTimeout)
		assert.Positive(t, writeTimeout)
		assert.Positive(t, idleTimeout)
	})
}

func TestNewApplicationRecovers(t *testing.T) {
	// A store that was never opened makes the debug route dereference nothing,
	// which is the simplest real handler panic to provoke. One request failing
	// must not take the process, and every open feed with it.
	t.Run("a handler that panics is answered 500", func(t *testing.T) {
		shared := database.DB
		database.DB = nil
		t.Cleanup(func() { database.DB = shared })
		captureLogs(t)

		assert.Equal(t, http.StatusInternalServerError, get(t, "/api/debug").StatusCode)
	})
}

func TestDatabaseDSN(t *testing.T) {
	// The driver reads only `_pragma` keys and ignores anything else in the
	// DSN, so the settings are asserted on the store it actually opens.
	t.Run("opens the store with the settings it names", func(t *testing.T) {
		dsn := strings.Replace(databaseDSN, "local.db", filepath.Join(t.TempDir(), "local.db"), 1)
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = pool.Close() })

		var journalMode string
		var busyTimeout, synchronous int
		require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error)
		require.NoError(t, db.Raw("PRAGMA busy_timeout").Scan(&busyTimeout).Error)
		require.NoError(t, db.Raw("PRAGMA synchronous").Scan(&synchronous).Error)

		assert.Equal(t, "wal", journalMode, "readers must not block capture writes")
		assert.Equal(t, 5000, busyTimeout)
		assert.Equal(t, 1, synchronous, "NORMAL")
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

	// The window holds only if the sweep keeps running between restarts.
	t.Run("schedules the sweep every five minutes", func(t *testing.T) {
		scheduler, err := startRetentionSweep()
		require.NoError(t, err)
		t.Cleanup(func() { scheduler.Stop() })

		entries := scheduler.Entries()
		require.Len(t, entries, 1)
		next := entries[0].Schedule.Next(time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC))
		assert.Equal(t, time.Date(2026, 1, 1, 12, 5, 0, 0, time.UTC), next)
	})
}

func TestApplicationEnv(t *testing.T) {
	t.Run("is development when APPLICATION_ENV is unset", func(t *testing.T) {
		t.Setenv("APPLICATION_ENV", "")

		assert.Equal(t, "development", applicationEnv())
	})

	t.Run("is whatever APPLICATION_ENV names", func(t *testing.T) {
		t.Setenv("APPLICATION_ENV", "production")

		assert.Equal(t, "production", applicationEnv())
	})
}

// isProduction is read once at startup, so the tests set it directly.
func TestListenAddress(t *testing.T) {
	t.Run("binds loopback only outside production", func(t *testing.T) {
		setProduction(t, false)

		assert.Equal(t, "localhost:8080", listenAddress())
	})

	t.Run("binds every interface in production", func(t *testing.T) {
		setProduction(t, true)

		assert.Equal(t, ":8080", listenAddress())
	})
}

// setProduction sets isProduction for one test and restores it afterwards.
func setProduction(t *testing.T, production bool) {
	t.Helper()
	previous := isProduction
	isProduction = production
	t.Cleanup(func() { isProduction = previous })
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
