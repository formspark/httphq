package main

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"httphq/src/database"
)

// settleTimeout bounds the wait for anything a connection does on a goroutine of
// its own: registration on connect, a pushed capture, the drop on close.
// Generous because it covers a scheduling delay rather than any work.
const settleTimeout = 2 * time.Second

// servedApplication starts an application on a loopback port of its own and
// returns its registry and the host it answers on. The socket routes are the
// one surface Fiber's in-memory test transport cannot reach, because an upgrade
// needs a real connection underneath it.
//
// It builds its own application rather than sharing the package's. socketio
// dispatches lifecycle events through package-level state, so the shared
// application's handlers fire for these connections too; a registry ignores a
// UUID it does not hold, which is what keeps that harmless.
func servedApplication(t *testing.T) (*socketRegistry, string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	registry := newSocketRegistry()
	application := newApplication(applicationConfig{
		viewsDir:  "./views",
		publicDir: "../public",
		registry:  registry,
	})
	go func() {
		// The banner is Fiber's own stdout output, not a log line, so it would
		// land in the middle of the test run's output rather than in a buffer.
		_ = application.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()
	t.Cleanup(func() { _ = application.Shutdown() })

	return registry, listener.Addr().String()
}

// openSocket connects to an endpoint's live feed and closes it at the end of the
// test, so a socket left open cannot be counted by the next one.
func openSocket(t *testing.T, host, endpointID string) *websocket.Conn {
	t.Helper()

	connection, response, err := websocket.DefaultDialer.Dial(
		"ws://"+host+"/ws/"+endpointID, nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	t.Cleanup(func() { _ = connection.Close() })
	return connection
}

// awaitCount waits for the registry to hold the expected number of sockets.
// Connect and disconnect are both handled on the connection's own goroutine, so
// neither is observable the instant the client returns.
func awaitCount(t *testing.T, registry *socketRegistry, want int) {
	t.Helper()
	require.Eventually(t, func() bool { return registry.count() == want },
		settleTimeout, 10*time.Millisecond,
		"registry should hold %d socket(s), holds %d", want, registry.count())
}

func TestSocketRegistry(t *testing.T) {
	t.Run("a new registry holds nothing", func(t *testing.T) {
		registry := newSocketRegistry()

		assert.Equal(t, 0, registry.count())
		assert.Nil(t, registry.uuidsFor("purple-frog-0691"))
	})

	t.Run("subscribers are returned per endpoint", func(t *testing.T) {
		registry := newSocketRegistry()
		registry.add("first", "uuid-1")
		registry.add("first", "uuid-2")
		registry.add("second", "uuid-3")

		assert.ElementsMatch(t, []string{"uuid-1", "uuid-2"}, registry.uuidsFor("first"))
		assert.Equal(t, []string{"uuid-3"}, registry.uuidsFor("second"))
		assert.Equal(t, 3, registry.count())
	})

	t.Run("adding the same socket twice counts once", func(t *testing.T) {
		registry := newSocketRegistry()
		registry.add("first", "uuid-1")
		registry.add("first", "uuid-1")

		assert.Equal(t, 1, registry.count())
	})

	t.Run("removing a socket leaves its neighbours subscribed", func(t *testing.T) {
		registry := newSocketRegistry()
		registry.add("first", "uuid-1")
		registry.add("first", "uuid-2")

		registry.remove("uuid-1")

		assert.Equal(t, []string{"uuid-2"}, registry.uuidsFor("first"))
	})

	// A page left open for hours reconnects repeatedly, so an endpoint whose
	// last socket closes must not keep an entry for every one it ever had.
	t.Run("an endpoint is forgotten once its last socket goes", func(t *testing.T) {
		registry := newSocketRegistry()
		registry.add("first", "uuid-1")

		registry.remove("uuid-1")

		assert.Equal(t, 0, registry.count())
		assert.Nil(t, registry.uuidsFor("first"))
	})

	t.Run("removing an unknown socket changes nothing", func(t *testing.T) {
		registry := newSocketRegistry()
		registry.add("first", "uuid-1")

		registry.remove("uuid-never-seen")

		assert.Equal(t, 1, registry.count())
	})
}

// The socket is the only route that answers something other than an HTTP
// response, so the guard in front of it decides what an ordinary caller gets.
func TestRegisterWebSockets(t *testing.T) {
	t.Run("a plain request is refused rather than served", func(t *testing.T) {
		response := get(t, "/ws/purple-frog-0691")

		assert.Equal(t, http.StatusUpgradeRequired, response.StatusCode)
	})

	// The guard is mounted on the prefix, so it answers before the route can
	// reject the ID. A malformed ID is refused for the wrong-looking reason,
	// which is fine: neither answer reveals whether the endpoint exists.
	t.Run("the guard covers the whole socket prefix", func(t *testing.T) {
		for _, path := range []string{"/ws", "/ws/", "/ws/Not_A_Valid_ID"} {
			t.Run(path, func(t *testing.T) {
				assert.Equal(t, http.StatusUpgradeRequired, get(t, path).StatusCode)
			})
		}
	})
}

// The live feed, driven over a real connection. Everything below the upgrade is
// what the registry is for, and none of it can be reached through the in-memory
// transport the rest of the suite uses.
func TestLiveFeed(t *testing.T) {
	t.Run("a connected socket is registered against its endpoint", func(t *testing.T) {
		registry, host := servedApplication(t)
		id := endpointID(t)

		openSocket(t, host, id)

		awaitCount(t, registry, 1)
		assert.Len(t, registry.uuidsFor(id), 1)
	})

	// The endpoint ID is the only isolation between two users of one instance,
	// so two pages on one process must not land in each other's stream.
	t.Run("two endpoints keep separate subscriber lists", func(t *testing.T) {
		registry, host := servedApplication(t)
		first, second := endpointID(t), endpointID(t)

		openSocket(t, host, first)
		openSocket(t, host, second)

		awaitCount(t, registry, 2)
		assert.Len(t, registry.uuidsFor(first), 1)
		assert.Len(t, registry.uuidsFor(second), 1)
	})

	// The whole point of the feed: a capture that arrives over the wire reaches
	// a page watching that endpoint, with no poll and no reload.
	t.Run("a capture arrives on the socket watching its endpoint", func(t *testing.T) {
		registry, host := servedApplication(t)
		id := endpointID(t)
		connection := openSocket(t, host, id)
		awaitCount(t, registry, 1)

		response, err := http.Post("http://"+host+"/to/"+id,
			"text/plain", nil)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())

		require.NoError(t, connection.SetReadDeadline(time.Now().Add(settleTimeout)))
		_, payload, err := connection.ReadMessage()
		require.NoError(t, err)

		var pushed database.Request
		require.NoError(t, json.Unmarshal(payload, &pushed),
			"a page parses the pushed payload as a capture")
		assert.Equal(t, id, pushed.EndpointID)
		assert.Equal(t, http.MethodPost, pushed.Method)
	})

	// A page left open for hours reconnects repeatedly. Without this the
	// registry grows a dead entry per reconnect and every capture is marshalled
	// for sockets that closed hours ago.
	t.Run("a closed socket is dropped from the registry", func(t *testing.T) {
		registry, host := servedApplication(t)
		connection := openSocket(t, host, endpointID(t))
		awaitCount(t, registry, 1)

		require.NoError(t, connection.Close())

		awaitCount(t, registry, 0)
	})
}
