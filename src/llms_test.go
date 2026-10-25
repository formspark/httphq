package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderLLMs(t *testing.T) {
	t.Run("is served as plain text", func(t *testing.T) {
		response := get(t, "/llms.txt")

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "text/plain; charset=utf-8", response.Header.Get("Content-Type"))
		assert.Contains(t, bodyOf(t, response), "# httphq")
	})

	// The figures are substituted from the constants that enforce them, and the
	// URLs from the request, so the file never quotes a stale limit or another
	// deployment's host.
	t.Run("states this deployment's URLs and the limits in force", func(t *testing.T) {
		body := bodyOf(t, get(t, "/llms.txt"))

		assert.Contains(t, body, "http://example.com/to/<endpoint-id>")
		assert.Contains(t, body, "deleted after 4 hours")
		assert.Contains(t, body, "limited to 1 MiB")
		assert.Contains(t, body, "150 requests per minute")
		assert.Contains(t, body, repositoryURL)
		assert.NotContains(t, body, "%!")
		assert.NotContains(t, body, "httphq.com")
	})

	t.Run("is open to crawlers", func(t *testing.T) {
		assert.Contains(t, bodyOf(t, get(t, "/robots.txt")), "Allow: /llms.txt$")
	})
}

func TestSizePhrase(t *testing.T) {
	t.Run("names whole mebibytes", func(t *testing.T) {
		assert.Equal(t, "1 MiB", sizePhrase(1<<20))
	})

	t.Run("falls back to kibibytes", func(t *testing.T) {
		assert.Equal(t, "512 KiB", sizePhrase(512<<10))
	})
}
