package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointURLs(t *testing.T) {
	t.Run("an https page advertises a wss socket", func(t *testing.T) {
		endpointURL, websocketURL, apiURL := endpointURLs("https", "httphq.com", "purple-frog-0691")

		assert.Equal(t, "https://httphq.com/to/purple-frog-0691", endpointURL)
		assert.Equal(t, "wss://httphq.com/ws/purple-frog-0691", websocketURL)
		assert.Equal(t, "https://httphq.com/api/endpoints/purple-frog-0691/requests", apiURL)
	})

	t.Run("an http page advertises a ws socket", func(t *testing.T) {
		endpointURL, websocketURL, apiURL := endpointURLs("http", "httphq.com", "purple-frog-0691")

		assert.Equal(t, "http://httphq.com/to/purple-frog-0691", endpointURL)
		assert.Equal(t, "ws://httphq.com/ws/purple-frog-0691", websocketURL)
		assert.Equal(t, "http://httphq.com/api/endpoints/purple-frog-0691/requests", apiURL)
	})

	// The URLs track the request rather than a configured hostname, so a
	// self-hosted deployment advertises itself instead of httphq.com.
	t.Run("a self-hosted deployment advertises its own host", func(t *testing.T) {
		endpointURL, websocketURL, apiURL := endpointURLs("http", "localhost:8080", "purple-frog-0691")

		assert.Equal(t, "http://localhost:8080/to/purple-frog-0691", endpointURL)
		assert.Equal(t, "ws://localhost:8080/ws/purple-frog-0691", websocketURL)
		assert.Equal(t, "http://localhost:8080/api/endpoints/purple-frog-0691/requests", apiURL)
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

func TestReservedPaths(t *testing.T) {
	// A slug-shaped path renders an endpoint page, so a word a visitor or a
	// crawler would guess as a page of the site has to be refused by name.
	t.Run("rejects the words reserved for pages of the site", func(t *testing.T) {
		for _, id := range []string{"about", "pricing", "docs", "login", "sitemap"} {
			assert.Falsef(t, validEndpointID(id), "%q should be rejected", id)
		}
	})

	// Every minted ID carries a numeric token after a hyphen, which no
	// reserved word has, so the list can never refuse an ID the site handed out.
	t.Run("no reserved word has the shape a minted ID takes", func(t *testing.T) {
		for word := range reservedPaths {
			assert.NotContainsf(t, word, "-", "%q could collide with a minted ID", word)
		}
	})
}

// The guard runs before the handler, so no route can reach the database, a
// template or a log line with an unvalidated ID. Every route carrying an
// :endpoint parameter has to be behind it. A reserved word is refused on every
// route too, so no capture can be stored under an ID whose page cannot open.
func TestRequireValidEndpoint(t *testing.T) {
	for _, id := range []string{"Not_A_Valid_ID", "about"} {
		routes := []testRequest{
			{method: http.MethodGet, path: "/" + id},
			{method: http.MethodGet, path: "/api/endpoints/" + id + "/requests"},
			{method: http.MethodDelete, path: "/api/endpoints/" + id + "/requests"},
			{method: http.MethodDelete, path: "/api/endpoints/" + id + "/requests/some-uuid"},
			{method: http.MethodPost, path: "/to/" + id},
		}

		for _, route := range routes {
			t.Run(route.method+" "+route.path, func(t *testing.T) {
				assert.Equal(t, http.StatusNotFound, do(t, route).StatusCode)
			})
		}
	}
}

// The endpoint page answers a refused ID with whatever handler it is given, so
// a person reads the not-found page and the page handler never runs.
func TestRequireValidEndpointPage(t *testing.T) {
	app := fiber.New()
	missing := func(c fiber.Ctx) error { return c.Status(http.StatusTeapot).SendString("missing") }
	app.Get("/:endpoint", requireValidEndpointPage(missing), func(c fiber.Ctx) error {
		return c.SendString("page")
	})

	cases := map[string]struct {
		status int
		body   string
	}{
		"/quiet-sunset-4821": {http.StatusOK, "page"},
		"/Quiet-Sunset-4821": {http.StatusTeapot, "missing"},
		"/about":             {http.StatusTeapot, "missing"},
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil))
			require.NoError(t, err)
			t.Cleanup(func() { _ = response.Body.Close() })

			assert.Equal(t, want.status, response.StatusCode)
			assert.Equal(t, want.body, bodyOf(t, response))
		})
	}
}
