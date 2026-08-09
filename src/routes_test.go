package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"httphq/src/database"
)

// The whole package shares one database. Captures are keyed by endpoint ID and
// every test mints its own, so no test can see another's traffic.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "httphq-test")
	if err != nil {
		panic(err)
	}
	database.Connect("file:" + filepath.Join(dir, "test.db"))
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

var (
	testApplicationOnce sync.Once
	testApplication     *fiber.App
)

// application returns the fully wired app, built against the repository's real
// templates and static files. It is built once: socket lifecycle handlers are
// registered process-wide, so a per-test app would stack them up.
func application(t *testing.T) *fiber.App {
	t.Helper()
	testApplicationOnce.Do(func() {
		testApplication = newApplication(applicationConfig{
			viewsDir:  "./views",
			publicDir: "../public",
			registry:  newSocketRegistry(),
		})
	})
	return testApplication
}

var endpointCounter atomic.Uint64

// endpointID mints an ID unique to one test, so no test can see another's
// captures. Counted rather than derived from the test name, which carries
// characters the endpoint pattern rejects.
func endpointID(t *testing.T) string {
	t.Helper()
	return "test-" + strconv.FormatUint(endpointCounter.Add(1), 10)
}

type testRequest struct {
	method  string
	path    string
	body    string
	headers map[string]string
}

func do(t *testing.T, spec testRequest) *http.Response {
	t.Helper()

	var body io.Reader
	if spec.body != "" {
		body = strings.NewReader(spec.body)
	}
	req := httptest.NewRequest(spec.method, spec.path, body)
	for name, value := range spec.headers {
		req.Header.Set(name, value)
	}

	response, err := application(t).Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func get(t *testing.T, path string) *http.Response {
	t.Helper()
	return do(t, testRequest{method: http.MethodGet, path: path})
}

func bodyOf(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}

type requestListing struct {
	Requests []database.Request `json:"requests"`
	Total    int64              `json:"total"`
}

// listRequests reads back what an endpoint has captured, through the same API
// the page uses.
func listRequests(t *testing.T, id, search string) requestListing {
	t.Helper()

	response := get(t, "/api/endpoints/"+id+"/requests?search="+search)
	require.Equal(t, http.StatusOK, response.StatusCode)

	var payload requestListing
	require.NoError(t, json.Unmarshal([]byte(bodyOf(t, response)), &payload))
	return payload
}

func capturedRequests(t *testing.T, id, search string) []database.Request {
	t.Helper()
	return listRequests(t, id, search).Requests
}

func listedTotal(t *testing.T, id, search string) int64 {
	t.Helper()
	return listRequests(t, id, search).Total
}

func TestHealthRoute(t *testing.T) {
	t.Run("answers 200 so a platform probe can reach it", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, get(t, "/api/health").StatusCode)
	})
}

func TestDebugRoute(t *testing.T) {
	t.Run("reports process state and no captured data", func(t *testing.T) {
		response := get(t, "/api/debug")
		require.Equal(t, http.StatusOK, response.StatusCode)

		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(bodyOf(t, response)), &payload))

		assert.Contains(t, payload, "host")
		assert.Contains(t, payload, "requests")
		assert.Equal(t, false, payload["isProduction"])
		assert.Equal(t, float64(0), payload["sockets"])
	})
}

func TestPageRoutes(t *testing.T) {
	t.Run("the landing page renders with its canonical and social tags", func(t *testing.T) {
		response := get(t, "/")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>httphq: inspect HTTP requests in real time</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/" />`)
		assert.Contains(t, body, `<meta property="og:image" content="http://example.com/social-card.png" />`)
	})

	t.Run("the contact page renders", func(t *testing.T) {
		response := get(t, "/contact")
		body := bodyOf(t, response)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, body, "<title>Contact | httphq</title>")
		assert.Contains(t, body, `<link rel="canonical" href="http://example.com/contact" />`)
	})

	t.Run("an endpoint page advertises its capture and socket URLs", func(t *testing.T) {
		id := endpointID(t)
		body := bodyOf(t, get(t, "/"+id))

		assert.Contains(t, body, "http://example.com/to/"+id)
		assert.Contains(t, body, `data-endpoint-id="`+id+`"`)
		assert.Contains(t, body, "endpoint.js?v=")
	})

	// The page expires captures out of its own list, so it needs the window as
	// a number. Rendering it from the same constant the sweep uses is what keeps
	// the two from drifting.
	t.Run("an endpoint page carries the retention window", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"+endpointID(t)))

		assert.Contains(t, body, `data-retention-seconds="14400"`)
	})

	// robots.txt excludes endpoint pages, so a canonical URL pointing them at a
	// shared address would be a claim nothing else in the site makes.
	t.Run("an endpoint page carries no canonical URL", func(t *testing.T) {
		body := bodyOf(t, get(t, "/"+endpointID(t)))

		assert.NotContains(t, body, `rel="canonical"`)
	})

	t.Run("creating an endpoint redirects to a valid endpoint page", func(t *testing.T) {
		response := do(t, testRequest{method: http.MethodPost, path: "/endpoint"})

		// See Other, so the browser follows with a GET rather than replaying
		// the POST at the new location.
		assert.Equal(t, http.StatusSeeOther, response.StatusCode)
		location := response.Header.Get(fiber.HeaderLocation)
		assert.True(t, validEndpointID(strings.TrimPrefix(location, "/")),
			"generated endpoint %q must satisfy the validator every route applies", location)
	})

	t.Run("an unknown path is a 404", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, get(t, "/no/such/page").StatusCode)
	})

	t.Run("static files are served from the public directory", func(t *testing.T) {
		response := get(t, "/robots.txt")

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Contains(t, bodyOf(t, response), "User-agent")
	})
}

// An endpoint ID reaches the templates, the database and the logs, so every
// route that takes one rejects a malformed value before the handler runs.
func TestEndpointIDRejection(t *testing.T) {
	malformed := "Not_A_Valid_ID"

	routes := []testRequest{
		{method: http.MethodGet, path: "/" + malformed},
		{method: http.MethodGet, path: "/api/endpoints/" + malformed + "/requests"},
		{method: http.MethodDelete, path: "/api/endpoints/" + malformed + "/requests"},
		{method: http.MethodDelete, path: "/api/endpoints/" + malformed + "/requests/some-uuid"},
		{method: http.MethodPost, path: "/to/" + malformed},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			assert.Equal(t, http.StatusNotFound, do(t, route).StatusCode)
		})
	}
}

func TestCaptureRoute(t *testing.T) {
	t.Run("records the method, path, query string, body and headers", func(t *testing.T) {
		id := endpointID(t)

		response := do(t, testRequest{
			method:  http.MethodPut,
			path:    "/to/" + id + "/orders/8821?event=charge.succeeded",
			body:    `{"hello":"world"}`,
			headers: map[string]string{"Content-Type": "application/json", "X-Sample": "value"},
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.NotEmpty(t, response.Header.Get("Httphq-Request-Uuid"),
			"the capture UUID is how a caller finds its own request in the stream")

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Equal(t, http.MethodPut, captured[0].Method)
		assert.Equal(t, "/to/"+id+"/orders/8821", captured[0].Path)
		assert.Equal(t, "event=charge.succeeded", captured[0].QueryString)
		assert.Equal(t, `{"hello":"world"}`, captured[0].Body)
		assert.Contains(t, string(captured[0].Headers), "X-Sample")
		assert.Equal(t, response.Header.Get("Httphq-Request-Uuid"), captured[0].UUID)
	})

	t.Run("captures every method", func(t *testing.T) {
		id := endpointID(t)

		methods := []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
			http.MethodDelete, http.MethodHead, http.MethodOptions,
		}
		for _, method := range methods {
			assert.Equalf(t, http.StatusOK,
				do(t, testRequest{method: method, path: "/to/" + id}).StatusCode,
				"%s is a request a user may want to inspect", method)
		}

		assert.Len(t, capturedRequests(t, id, ""), len(methods))
	})

	t.Run("hides infrastructure headers from the capture", func(t *testing.T) {
		id := endpointID(t)

		do(t, testRequest{
			method: http.MethodPost,
			path:   "/to/" + id,
			body:   "x",
			headers: map[string]string{
				"X-Forwarded-For":  "198.51.100.1",
				"X-Forwarded-Host": "elsewhere.example",
				"Via":              "1.1 proxy",
				"X-Sample":         "value",
			},
		})

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		headers := string(captured[0].Headers)
		assert.Contains(t, headers, "X-Sample")
		assert.NotContains(t, headers, "X-Forwarded-For")
		assert.NotContains(t, headers, "X-Forwarded-Host")
		assert.NotContains(t, headers, "Via")
	})

	t.Run("an empty body is captured as an empty body", func(t *testing.T) {
		id := endpointID(t)

		do(t, testRequest{method: http.MethodGet, path: "/to/" + id})

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Empty(t, captured[0].Body)
	})

	t.Run("a body at the limit is captured whole", func(t *testing.T) {
		id := endpointID(t)
		body := strings.Repeat("a", bodyLimit)

		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: body})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Len(t, captured[0].Body, bodyLimit)
	})
}

func TestRequestsAPI(t *testing.T) {
	t.Run("lists an endpoint's captures, newest first", func(t *testing.T) {
		id := endpointID(t)
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "older"})
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "newer"})

		captured := capturedRequests(t, id, "")

		require.Len(t, captured, 2)
		assert.Equal(t, "newer", captured[0].Body)
		assert.Equal(t, "older", captured[1].Body)
	})

	t.Run("the search parameter narrows the list", func(t *testing.T) {
		id := endpointID(t)
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "alpha"})
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "beta"})

		assert.Len(t, capturedRequests(t, id, "alpha"), 1)
		assert.Empty(t, capturedRequests(t, id, "no-such-term"))
	})

	// The listing is both filtered and windowed, so its length says nothing
	// about the endpoint. A control that clears the whole endpoint has to
	// report what it will actually delete.
	t.Run("the total is the endpoint's, not the filtered view's", func(t *testing.T) {
		id := endpointID(t)
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "alpha"})
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "beta"})

		assert.Equal(t, int64(2), listedTotal(t, id, ""))
		assert.Equal(t, int64(2), listedTotal(t, id, "alpha"))
		assert.Equal(t, int64(2), listedTotal(t, id, "no-such-term"))
	})

	t.Run("an endpoint with no traffic totals zero", func(t *testing.T) {
		assert.Equal(t, int64(0), listedTotal(t, endpointID(t), ""))
	})

	t.Run("an endpoint with no traffic lists nothing", func(t *testing.T) {
		assert.Empty(t, capturedRequests(t, endpointID(t), ""))
	})

	t.Run("deleting one capture leaves the rest", func(t *testing.T) {
		id := endpointID(t)
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "keep"})
		doomed := do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "delete"})
		uuid := doomed.Header.Get("Httphq-Request-Uuid")

		response := do(t, testRequest{
			method: http.MethodDelete,
			path:   "/api/endpoints/" + id + "/requests/" + uuid,
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Equal(t, "keep", captured[0].Body)
	})

	t.Run("deleting an endpoint's captures clears only that endpoint", func(t *testing.T) {
		id, other := endpointID(t), endpointID(t)+"-other"
		do(t, testRequest{method: http.MethodPost, path: "/to/" + id, body: "x"})
		do(t, testRequest{method: http.MethodPost, path: "/to/" + other, body: "y"})

		response := do(t, testRequest{
			method: http.MethodDelete,
			path:   "/api/endpoints/" + id + "/requests",
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Empty(t, capturedRequests(t, id, ""))
		assert.Len(t, capturedRequests(t, other, ""), 1)
	})
}

func TestResponseHeaders(t *testing.T) {
	t.Run("every response carries the security headers", func(t *testing.T) {
		for _, path := range []string{"/", "/contact", "/api/health", "/no/such/page"} {
			t.Run(path, func(t *testing.T) {
				response := get(t, path)

				assert.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
				assert.Equal(t, "no-referrer", response.Header.Get("Referrer-Policy"))
				assert.Equal(t, "DENY", response.Header.Get("X-Frame-Options"))
				assert.Contains(t, response.Header.Get(fiber.HeaderContentSecurityPolicy), "frame-ancestors 'none'")
			})
		}
	})

	t.Run("a valid inbound request ID is echoed back", func(t *testing.T) {
		response := do(t, testRequest{
			method:  http.MethodGet,
			path:    "/api/health",
			headers: map[string]string{"X-Request-Id": "abcd-1234-efgh"},
		})

		assert.Equal(t, "abcd-1234-efgh", response.Header.Get("X-Request-Id"))
	})

	t.Run("a request with no ID is given one", func(t *testing.T) {
		assert.NotEmpty(t, get(t, "/api/health").Header.Get("X-Request-Id"))
	})
}
