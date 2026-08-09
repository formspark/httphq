package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureRequest(t *testing.T) {
	t.Run("records the method, path, query string, body and headers", func(t *testing.T) {
		id := endpointID(t)

		response := do(t, testRequest{
			method:  http.MethodPut,
			path:    "/to/" + id + "/orders/8821?event=charge.succeeded",
			body:    `{"hello":"world"}`,
			headers: map[string]string{"Content-Type": "application/json", "X-Sample": "value"},
		})

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.NotEmpty(t, response.Header.Get(captureUUIDHeader),
			"the capture UUID is how a caller finds its own request in the stream")

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Equal(t, http.MethodPut, captured[0].Method)
		assert.Equal(t, "/to/"+id+"/orders/8821", captured[0].Path)
		assert.Equal(t, "event=charge.succeeded", captured[0].QueryString)
		assert.Equal(t, `{"hello":"world"}`, captured[0].Body)
		assert.Contains(t, string(captured[0].Headers), "X-Sample")
		assert.Equal(t, response.Header.Get(captureUUIDHeader), captured[0].UUID)
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

		get(t, "/to/"+id)

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Empty(t, captured[0].Body)
	})

	t.Run("a body at the limit is captured whole", func(t *testing.T) {
		id := endpointID(t)

		response := post(t, "/to/"+id, strings.Repeat("a", bodyLimit))

		assert.Equal(t, http.StatusOK, response.StatusCode)
		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Len(t, captured[0].Body, bodyLimit)
	})

	// The limit is what keeps one caller from filling the disk a whole shared
	// instance writes to. It is enforced by the server rather than the handler,
	// so the transport refuses the payload and the caller never gets an answer
	// that would read as a successful capture. The status a real client sees is
	// covered end to end, where there is a socket to see it on.
	t.Run("a body over the limit is refused by the transport", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/to/"+endpointID(t),
			strings.NewReader(strings.Repeat("a", bodyLimit+1)))

		_, err := application(t).Test(request)

		require.Error(t, err)
	})
}

func TestCaptureHeaders(t *testing.T) {
	t.Run("leaves a caller's own headers alone", func(t *testing.T) {
		withPlatform(t, "direct")

		headers := captureHeaders(map[string][]string{
			"Content-Type": {"application/json"},
			"X-Sample":     {"value"},
		})

		assert.Equal(t, map[string][]string{
			"Content-Type": {"application/json"},
			"X-Sample":     {"value"},
		}, headers)
	})

	t.Run("drops infrastructure headers", func(t *testing.T) {
		withPlatform(t, "cloudflare")

		headers := captureHeaders(map[string][]string{
			"X-Forwarded-For": {"198.51.100.1"},
			"Cf-Ray":          {"abc"},
			"X-Sample":        {"value"},
		})

		assert.Equal(t, map[string][]string{"X-Sample": {"value"}}, headers)
	})

	t.Run("spoofing curl replaces the browser's fingerprint", func(t *testing.T) {
		withPlatform(t, "direct")

		headers := captureHeaders(map[string][]string{
			spoofCurlHeader:   {"true"},
			"Sec-Fetch-Mode":  {"cors"},
			"Origin":          {"https://example.com"},
			"Accept-Encoding": {"gzip"},
			"Content-Type":    {"application/json"},
			"User-Agent":      {"Mozilla/5.0"},
			"X-Sample":        {"value"},
		})

		assert.Equal(t, map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
			"User-Agent":   {"curl/7.79.1"},
			"X-Sample":     {"value"},
		}, headers)
	})

	t.Run("only the opt-in value turns spoofing on", func(t *testing.T) {
		withPlatform(t, "direct")

		headers := captureHeaders(map[string][]string{
			spoofCurlHeader: {"false"},
			"User-Agent":    {"Mozilla/5.0"},
		})

		assert.Equal(t, map[string][]string{
			spoofCurlHeader: {"false"},
			"User-Agent":    {"Mozilla/5.0"},
		}, headers)
	})
}

// The stored shape is what every consumer reads: the request panel, the search
// index and the HAR export all expect a value to be a string or a list of them.
func TestFlattenHeaders(t *testing.T) {
	t.Run("a single value becomes a scalar", func(t *testing.T) {
		assert.Equal(t, map[string]any{"X-Sample": "value"},
			flattenHeaders(map[string][]string{"X-Sample": {"value"}}))
	})

	t.Run("a repeated header keeps every value in order", func(t *testing.T) {
		assert.Equal(t, map[string]any{"Set-Cookie": []string{"a=1", "b=2"}},
			flattenHeaders(map[string][]string{"Set-Cookie": {"a=1", "b=2"}}))
	})

	t.Run("no headers flatten to an empty object", func(t *testing.T) {
		assert.Empty(t, flattenHeaders(map[string][]string{}))
	})
}
