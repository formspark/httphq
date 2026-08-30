package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"httphq/src/database"
)

// Shared fixtures for the tests that drive real requests through the wired
// application. Everything here is used from more than one subject's test file;
// anything specific to one subject belongs beside that subject instead.

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

// withPlatform points the process-wide platform config at name for the duration
// of one test. Client-IP resolution and header stripping both read that global,
// so a test that changes it has to put it back.
func withPlatform(t *testing.T, name string) {
	t.Helper()
	previous := currentPlatform
	currentPlatform = resolvePlatform(name)
	t.Cleanup(func() { currentPlatform = previous })
}

// captureLogs points the process logger at a buffer for one test and hands the
// buffer back. Several subjects report through the logs and nowhere else, so
// reading them is the only way to assert what happened. The logger is
// process-wide, so it is restored before the next test reads it.
//
// Debug is recorded as well as info, because a line demoted to debug is still a
// line a test may be asserting on.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &out
}

// contextWith builds a request context from peerIP carrying the given headers,
// so a test can exercise the header-reading helpers without a live server.
func contextWith(t *testing.T, app *fiber.App, peerIP string, headers map[string]string) fiber.Ctx {
	t.Helper()

	fctx := &fasthttp.RequestCtx{}
	if peerIP != "" {
		fctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(peerIP)})
	}

	c := app.AcquireCtx(fctx)
	t.Cleanup(func() { app.ReleaseCtx(c) })

	for name, value := range headers {
		c.Request().Header.Set(name, value)
	}
	return c
}

type testRequest struct {
	method  string
	path    string
	body    string
	headers map[string]string
	// repeated carries headers sent more than once, which the single-valued
	// map above cannot express. RFC 7230 allows a header to repeat and the
	// capture handler stores those values as a list, so the two shapes are
	// spelled apart rather than collapsed.
	repeated map[string][]string
}

// applyHeaders writes a spec's headers onto a request. The two maps are walked
// separately because they say different things: one header per name, and one
// name carrying several.
func applyHeaders(request *http.Request, spec testRequest) {
	for name, value := range spec.headers {
		request.Header.Set(name, value)
	}
	for name, values := range spec.repeated {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
}

func do(t *testing.T, spec testRequest) *http.Response {
	t.Helper()

	var body io.Reader
	if spec.body != "" {
		body = strings.NewReader(spec.body)
	}
	req := httptest.NewRequest(spec.method, spec.path, body)
	applyHeaders(req, spec)

	response, err := application(t).Test(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func get(t *testing.T, path string) *http.Response {
	t.Helper()
	return do(t, testRequest{method: http.MethodGet, path: path})
}

func post(t *testing.T, path, body string) *http.Response {
	t.Helper()
	return do(t, testRequest{method: http.MethodPost, path: path, body: body})
}

func bodyOf(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return string(body)
}

// prose collapses the whitespace a template's own line breaks introduce, so an
// assertion about a rendered sentence does not depend on where the formatter
// happened to wrap it. Use it for copy; markup is matched on bodyOf.
func prose(t *testing.T, response *http.Response) string {
	t.Helper()
	return strings.Join(strings.Fields(bodyOf(t, response)), " ")
}

// requestListing is the listing response, decoded. Tests read what an endpoint
// captured back through the same API the page polls, so a capture is only
// proven stored once it is also readable.
type requestListing struct {
	Requests []database.Request `json:"requests"`
	Total    int64              `json:"total"`
	Cursor   string             `json:"cursor"`
	HasMore  bool               `json:"hasMore"`
}

// listRequests reads back what an endpoint has captured. An empty `since` is
// the browser's call, with no cursor.
func listRequests(t *testing.T, id, search, since string) requestListing {
	t.Helper()

	response := get(t, "/api/endpoints/"+id+"/requests?search="+search+
		"&since="+url.QueryEscape(since))
	require.Equal(t, http.StatusOK, response.StatusCode)

	var payload requestListing
	require.NoError(t, json.Unmarshal([]byte(bodyOf(t, response)), &payload))
	return payload
}

func capturedRequests(t *testing.T, id, search string) []database.Request {
	t.Helper()
	return listRequests(t, id, search, "").Requests
}

func listedTotal(t *testing.T, id, search string) int64 {
	t.Helper()
	return listRequests(t, id, search, "").Total
}
