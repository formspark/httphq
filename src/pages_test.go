package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestRenderIndex(t *testing.T) {
	t.Run("renders with its canonical and social tags", func(t *testing.T) {
		response := get(t, "/")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>httphq: inspect HTTP requests in real time</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/" />`)
		assert.Contains(t, body, `<meta property="og:image" content="http://example.com/social-card.png" />`)
	})

	// The window is a promise the landing page makes before a visitor points
	// traffic at a public URL, so it is rendered from the constant the sweep
	// uses rather than typed into the copy.
	t.Run("states the retention window", func(t *testing.T) {
		assert.Contains(t, bodyOf(t, get(t, "/")), "deleted after 4 hours")
	})
}

func TestRenderContact(t *testing.T) {
	t.Run("renders with its canonical tag", func(t *testing.T) {
		response := get(t, "/contact")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>Contact | httphq</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/contact" />`)
	})
}

func TestRenderEndpoint(t *testing.T) {
	t.Run("advertises its capture and socket URLs", func(t *testing.T) {
		id := endpointID(t)
		body := bodyOf(t, get(t, "/"+id))

		assert.Contains(t, body, "http://example.com/to/"+id)
		assert.Contains(t, body, `data-endpoint-id="`+id+`"`)
		assert.Contains(t, body, "endpoint.js?v=")
	})

	// The page expires captures out of its own list, so it needs the window as
	// a number. Rendering it from the same constant the sweep uses is what keeps
	// the two from drifting.
	t.Run("carries the retention window as a number and as prose", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"+endpointID(t)))

		assert.Contains(t, body, `data-retention-seconds="14400"`)
		assert.Contains(t, body, "deleted after 4 hours")
	})

	// The prompt is built from the request, so a self-hosted deployment hands
	// out its own URLs rather than httphq.com's.
	t.Run("carries an agent prompt for this host", func(t *testing.T) {
		id := endpointID(t)
		body := bodyOf(t, get(t, "/"+id))

		assert.Contains(t, body, "http://example.com/api/endpoints/"+id+"/requests")
		assert.Contains(t, body, "150 requests per minute")
		assert.NotContains(t, body, "httphq.com")
	})

	// robots.txt excludes endpoint pages, so a canonical URL pointing them at a
	// shared address would be a claim nothing else in the site makes.
	t.Run("carries no canonical URL", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"+endpointID(t)))

		assert.NotContains(t, body, `rel="canonical"`)
	})
}

func TestCreateEndpoint(t *testing.T) {
	t.Run("redirects to a valid endpoint page", func(t *testing.T) {
		response := do(t, testRequest{method: http.MethodPost, path: "/endpoint"})

		// See Other, so the browser follows with a GET rather than replaying
		// the POST at the new location.
		assert.Equal(t, http.StatusSeeOther, response.StatusCode)
		location := response.Header.Get(fiber.HeaderLocation)
		assert.True(t, validEndpointID(strings.TrimPrefix(location, "/")),
			"generated endpoint %q must satisfy the validator every route applies", location)
	})
}
