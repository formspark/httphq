package main

import (
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
