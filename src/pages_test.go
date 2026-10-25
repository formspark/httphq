package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every page's head fields come from pageMeta, so what it emits is what the
// whole site advertises. These cover the fields that have to hold across all of
// them; a page's own copy is asserted beside its handler below.
func TestPageMeta(t *testing.T) {
	pages := map[string]string{
		"index":    "/",
		"contact":  "/contact",
		"endpoint": "/" + endpointID(t),
	}

	// The card carries a content hash, so it is matched by prefix rather than
	// in full: pinning the whole URL would pin the bytes of the image.
	t.Run("every page advertises the social card", func(t *testing.T) {
		for name, path := range pages {
			t.Run(name, func(t *testing.T) {
				assert.Contains(t, bodyOf(t, get(t, path)),
					`<meta property="og:image" content="http://example.com/social-card.png?v=`)
			})
		}
	})

	// The URLs track the request rather than a configured hostname, so a
	// self-hosted deployment advertises itself instead of httphq.com.
	t.Run("absolute URLs name the host that served the page", func(t *testing.T) {
		for name, path := range pages {
			t.Run(name, func(t *testing.T) {
				assert.NotContains(t, bodyOf(t, get(t, path)), "httphq.com")
			})
		}
	})

	// The window is a promise, so it is rendered from the constant the sweep
	// uses rather than typed into the copy of each page that states it.
	// The contact page is not in this list because it never states the window.
	t.Run("the pages that state the retention window render it from one constant", func(t *testing.T) {
		for _, name := range []string{"index", "endpoint"} {
			t.Run(name, func(t *testing.T) {
				assert.Contains(t, prose(t, get(t, pages[name])), "deleted after 4 hours")
			})
		}
	})
}

// A host served under both its bare and its www. name would have two
// addresses that each canonicalise to themselves.
func TestRedirectWWW(t *testing.T) {
	t.Run("a read on a www. host moves permanently to the bare host", func(t *testing.T) {
		for _, method := range readMethods {
			response := do(t, testRequest{method: method, path: "http://www.example.com/contact?ref=a&b=c"})

			assert.Equal(t, http.StatusMovedPermanently, response.StatusCode, method)
			assert.Equal(t, "http://example.com/contact?ref=a&b=c", response.Header.Get(fiber.HeaderLocation), method)
		}
	})

	// Webhook senders mostly do not follow redirects, so a capture sent to a
	// www. URL is stored rather than bounced.
	t.Run("a capture on a www. host is stored where it landed", func(t *testing.T) {
		id := endpointID(t)
		response := do(t, testRequest{method: http.MethodPost, path: "http://www.example.com/to/" + id, body: "{}"})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Len(t, capturedRequests(t, id, ""), 1)
	})

	t.Run("the bare host is served as it is", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(t, "http://example.com/contact").StatusCode)
	})
}

func TestRenderIndex(t *testing.T) {
	t.Run("renders with its title and canonical tag", func(t *testing.T) {
		response := get(t, "/")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>httphq: inspect HTTP requests in real time</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/" />`)
	})

	// A rich result is built from the structured data alone, so it has to parse
	// as JSON and describe the page it sits on with the copy its head states.
	t.Run("carries WebApplication structured data that parses", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"))
		_, script, found := strings.Cut(body, `<script type="application/ld+json">`)
		require.True(t, found, "the home page carries no JSON-LD")
		script, _, _ = strings.Cut(script, "</script>")

		var data map[string]any
		require.NoError(t, json.Unmarshal([]byte(script), &data))
		assert.Equal(t, "https://schema.org", data["@context"])
		assert.Equal(t, "WebApplication", data["@type"])
		assert.Equal(t, "httphq", data["name"])
		assert.Equal(t, "http://example.com/", data["url"])
		assert.Equal(t, indexDescription, data["description"])
		assert.Equal(t, "DeveloperApplication", data["applicationCategory"])
		assert.Equal(t, "Any", data["operatingSystem"])
		assert.Equal(t, true, data["isAccessibleForFree"])
		assert.Equal(t, map[string]any{"@type": "Offer", "price": "0", "priceCurrency": "USD"}, data["offers"])
		assert.Equal(t, repositoryURL, data["sameAs"])
	})
}

func TestRenderContact(t *testing.T) {
	t.Run("renders with its title and canonical tag", func(t *testing.T) {
		response := get(t, "/contact")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>Contact | httphq</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/contact" />`)
	})
}

func TestRenderEndpoint(t *testing.T) {
	// The page script opens the feed against the URL rendered here rather than
	// rebuilding it, so a socket URL the page never carries is a page whose feed
	// never connects.
	t.Run("advertises its capture and socket URLs", func(t *testing.T) {
		id := endpointID(t)
		body := bodyOf(t, get(t, "/"+id))

		assert.Contains(t, body, "http://example.com/to/"+id)
		assert.Contains(t, body, `data-endpoint-id="`+id+`"`)
		assert.Contains(t, body, `data-websocket-url="ws://example.com/ws/`+id+`"`)
		assert.Contains(t, body, "endpoint.js?v=")

		// html/template rewrites a URL attribute it does not trust to this
		// sentinel rather than failing, so an untyped ws:// URL would render a
		// page that looks intact and never opens its feed.
		assert.NotContains(t, body, "ZgotmplZ")
	})

	// The page expires captures out of its own list, so it needs the window as
	// a number as well as the prose every page states.
	t.Run("carries the retention window as a number", func(t *testing.T) {
		assert.Contains(t, bodyOf(t, get(t, "/"+endpointID(t))), `data-retention-seconds="14400"`)
	})

	// The prompt is built from the request, so a self-hosted deployment hands
	// out its own URLs and the limits actually in force.
	t.Run("carries an agent prompt for this host", func(t *testing.T) {
		id := endpointID(t)
		body := bodyOf(t, get(t, "/"+id))

		assert.Contains(t, body, "http://example.com/api/endpoints/"+id+"/requests")
		assert.Contains(t, body, "150 requests per minute")
	})

	// A path a visitor guesses as a page of the site has to be missing, not an
	// empty capture stream answering 200 for a search engine to index.
	t.Run("a reserved word or a dotted file name is a 404, not an endpoint page", func(t *testing.T) {
		for _, path := range []string{"/about", "/pricing", "/docs", "/sitemap.xml", "/favicon.svg"} {
			assert.Equal(t, http.StatusNotFound, get(t, path).StatusCode, path)
		}
	})

	// robots.txt excludes endpoint pages, so a canonical URL pointing them at a
	// shared address would be a claim nothing else in the site makes.
	t.Run("carries no canonical URL", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"+endpointID(t)))

		assert.NotContains(t, body, `rel="canonical"`)
		assert.NotContains(t, body, `property="og:url"`)
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

func TestIDMaker(t *testing.T) {
	// Every POST /endpoint draws from one generator, and its random source is
	// not safe for concurrent use. Run under -race, an unguarded draw is
	// reported here.
	t.Run("concurrent draws are safe", func(t *testing.T) {
		ids := &idMaker{haikunator: haikunator.New()}
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 200 {
					assert.True(t, validEndpointID(ids.next()))
				}
			})
		}
		wg.Wait()
	})
}
