package main

import (
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
	"github.com/stretchr/testify/require"

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
