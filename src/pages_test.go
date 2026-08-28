package main

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/atrox/haikunatorgo/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
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

func TestRenderIndex(t *testing.T) {
	t.Run("renders with its title and canonical tag", func(t *testing.T) {
		response := get(t, "/")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>httphq: inspect HTTP requests in real time</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/" />`)
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

// The endpoint ID is the only thing protecting a capture stream, and those
// streams routinely hold webhooks carrying credentials. These assert the three
// properties that makes necessary: enough room that ids do not repeat, an
// unpredictable source, and safety when minted from concurrent requests.
func TestNewEndpointID(t *testing.T) {
	words := haikunator.New()

	t.Run("emits an id the routes accept", func(t *testing.T) {
		id, err := newEndpointID(words)

		assert.NoError(t, err)
		assert.Regexp(t, endpointIDPattern, id)
	})

	t.Run("does not repeat across many mints", func(t *testing.T) {
		seen := make(map[string]struct{}, 10000)
		for range 10000 {
			id, err := newEndpointID(words)
			assert.NoError(t, err)
			_, duplicate := seen[id]
			assert.False(t, duplicate, "minted a duplicate endpoint id: %s", id)
			seen[id] = struct{}{}
		}
	})

	// haikunator's own generator is documented as unsafe for concurrent use,
	// and endpoints are minted from Fiber's per-request goroutines, so this is
	// the case that has to hold under -race.
	t.Run("is safe to mint concurrently", func(t *testing.T) {
		var group sync.WaitGroup
		ids := make([]string, 64)
		for i := range ids {
			group.Add(1)
			go func() {
				defer group.Done()
				id, err := newEndpointID(words)
				assert.NoError(t, err)
				ids[i] = id
			}()
		}
		group.Wait()

		unique := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			assert.Regexp(t, endpointIDPattern, id)
			unique[id] = struct{}{}
		}
		assert.Len(t, unique, len(ids))
	})
}
