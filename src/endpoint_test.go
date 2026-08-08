package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpointURLs(t *testing.T) {
	t.Run("an https page advertises a wss socket", func(t *testing.T) {
		endpointURL, websocketURL := endpointURLs("https", "httphq.com", "purple-frog-0691")

		assert.Equal(t, "https://httphq.com/to/purple-frog-0691", endpointURL)
		assert.Equal(t, "wss://httphq.com/ws/purple-frog-0691", websocketURL)
	})

	t.Run("an http page advertises a ws socket", func(t *testing.T) {
		endpointURL, websocketURL := endpointURLs("http", "httphq.com", "purple-frog-0691")

		assert.Equal(t, "http://httphq.com/to/purple-frog-0691", endpointURL)
		assert.Equal(t, "ws://httphq.com/ws/purple-frog-0691", websocketURL)
	})
}

// The endpoint ID reaches the rendered pages, the database and the logs, so the
// pattern is the only thing keeping attacker-controlled characters out of all
// three. Anything outside the shape haikunator emits must be rejected.
func TestValidEndpointID(t *testing.T) {
	t.Run("accepts the shape haikunator emits", func(t *testing.T) {
		accepted := []string{
			"purple-frog-0691",
			"a",
			"9",
			"cool-wave",
			"still-brook-1234",
			strings.Repeat("a", 64),
		}

		for _, id := range accepted {
			assert.Truef(t, validEndpointID(id), "%q should be accepted", id)
		}
	})

	t.Run("rejects anything that could break out of a page, a query or a log line", func(t *testing.T) {
		rejected := []string{
			"",
			"UPPER-case",
			"under_score",
			"has space",
			"dot.separated",
			"trailing-",
			"-leading",
			"double--hyphen",
			"slash/es",
			"quote'd",
			`quote"d`,
			"<script>",
			"semi;colon",
			"percent%20",
			"new\nline",
			"null\x00byte",
			strings.Repeat("a", 65),
		}

		for _, id := range rejected {
			assert.Falsef(t, validEndpointID(id), "%q should be rejected", id)
		}
	})
}
