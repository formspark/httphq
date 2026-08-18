package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
