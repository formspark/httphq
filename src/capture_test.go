package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"httphq/src/database"
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

	t.Run("stores the client IP the platform resolves", func(t *testing.T) {
		withPlatform(t, "heroku")
		id := endpointID(t)

		do(t, testRequest{
			method:  http.MethodPost,
			path:    "/to/" + id,
			body:    "x",
			headers: map[string]string{"X-Forwarded-For": "192.0.2.66, 198.51.100.1"},
		})

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		assert.Equal(t, "198.51.100.1", captured[0].IP)
	})

	// flattenHeaders decides the stored shape, and every consumer downstream
	// reads it: the request panel, the search index and the HAR export all
	// expect a scalar or a list. This drives a real request through the wire so
	// the shape is proven where it is actually produced.
	t.Run("a repeated header is stored as a list, a single one as a scalar", func(t *testing.T) {
		id := endpointID(t)

		do(t, testRequest{
			method:   http.MethodPost,
			path:     "/to/" + id,
			body:     "x",
			headers:  map[string]string{"X-Single": "one"},
			repeated: map[string][]string{"X-Multi": {"first", "second"}},
		})

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		var headers map[string]any
		require.NoError(t, json.Unmarshal(captured[0].Headers, &headers))
		assert.Equal(t, "one", headers["X-Single"])
		assert.Equal(t, []any{"first", "second"}, headers["X-Multi"])
	})

	// The opt-in is read off the wire rather than from a map a test built, so
	// this covers the header name and value the browser actually sends.
	t.Run("spoofing curl is honoured on a request off the wire", func(t *testing.T) {
		id := endpointID(t)

		do(t, testRequest{
			method: http.MethodPost,
			path:   "/to/" + id,
			body:   "hello=world",
			headers: map[string]string{
				spoofCurlHeader: "true",
				"User-Agent":    "Mozilla/5.0",
				"Origin":        "https://example.com",
			},
		})

		captured := capturedRequests(t, id, "")
		require.Len(t, captured, 1)
		headers := string(captured[0].Headers)
		assert.Contains(t, headers, "curl/7.79.1")
		assert.NotContains(t, headers, "Mozilla")
		assert.NotContains(t, headers, "Origin")
		assert.NotContains(t, headers, spoofCurlHeader)
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
	// instance writes to. The transport refuses the payload, so the caller never
	// gets an answer that would read as a successful capture.
	//
	// The in-memory transport will not send a body shorter than the length it
	// declares, so the handler's own guard against an oversized declaration
	// cannot be reached from here. It is covered end to end, where there is a
	// socket to see both the status and what the endpoint was left holding.
	t.Run("a body over the limit is refused by the transport", func(t *testing.T) {
		id := endpointID(t)
		request := httptest.NewRequest(http.MethodPost, "/to/"+id,
			strings.NewReader(strings.Repeat("a", bodyLimit+1)))

		_, err := application(t).Test(request)

		require.Error(t, err)
		assert.Empty(t, capturedRequests(t, id, ""))
	})
}

// A sender is told its capture was stored, with the UUID to find it by, so a
// capture the store refused must not be answered as one it kept.
func TestCaptureRequestStoreFailure(t *testing.T) {
	t.Run("a capture the store refuses is answered 503 without a UUID", func(t *testing.T) {
		brokenStore(t)

		response := post(t, "/to/"+endpointID(t), "payload")

		assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
		assert.Empty(t, response.Header.Get(captureUUIDHeader))
	})
}

func gzipped(t *testing.T, payload []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return out.Bytes()
}

// zstdEncoded compresses payload with the encoder's defaults.
func zstdEncoded(t *testing.T, payload []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	encoder, err := zstd.NewWriter(&out)
	require.NoError(t, err)
	_, err = encoder.Write(payload)
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	return out.Bytes()
}

// Bodies are stored as the sender meant them, decoded, and decoding is held to
// the same limit as the body itself.
func TestCaptureBody(t *testing.T) {
	t.Run("a gzip body is stored decoded", func(t *testing.T) {
		id := endpointID(t)
		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id,
			body: string(gzipped(t, []byte("hello"))), headers: map[string]string{"Content-Encoding": "gzip"}})

		require.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "hello", capturedRequests(t, id, "")[0].Body)
	})

	t.Run("a zstd body is stored decoded", func(t *testing.T) {
		id := endpointID(t)
		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id,
			body: string(zstdEncoded(t, []byte("hello"))), headers: map[string]string{"Content-Encoding": "zstd"}})

		require.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "hello", capturedRequests(t, id, "")[0].Body)
	})

	// A frame can declare a huge decode window regardless of how little it
	// carries, which is what the default decoder would allocate up front. This
	// one declares 512 MiB around a single byte; the capped decoder refuses it.
	t.Run("a zstd body that declares a window beyond the body limit is refused", func(t *testing.T) {
		id := endpointID(t)
		frame := []byte{0x28, 0xb5, 0x2f, 0xfd, 0x00, 0x98, 0x0a, 0x00, 0x00, 'A'}
		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id,
			body: string(frame), headers: map[string]string{"Content-Encoding": "zstd"}})

		assert.Equal(t, http.StatusRequestEntityTooLarge, response.StatusCode)
		assert.Empty(t, capturedRequests(t, id, ""))
	})

	t.Run("a body that is not valid zstd is refused", func(t *testing.T) {
		id := endpointID(t)
		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id,
			body: "not zstd", headers: map[string]string{"Content-Encoding": "zstd"}})

		assert.Equal(t, http.StatusBadRequest, response.StatusCode)
		assert.Empty(t, capturedRequests(t, id, ""))
	})

	// Each stage is bounded by the body limit but the number of stages is not,
	// so a chain is refused rather than decoded stage after stage.
	t.Run("a chain of encodings is refused", func(t *testing.T) {
		id := endpointID(t)
		body := gzipped(t, gzipped(t, []byte("hello")))
		response := do(t, testRequest{method: http.MethodPost, path: "/to/" + id,
			body: string(body), headers: map[string]string{"Content-Encoding": "gzip, gzip"}})

		assert.Equal(t, http.StatusUnsupportedMediaType, response.StatusCode)
		assert.Empty(t, capturedRequests(t, id, ""))
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

// watching registers a subscriber on an endpoint and hands it back, so a test
// can read what fan-out offered it straight from its outbox.
func watching(registry *socketRegistry, endpointID string) *subscriber {
	feed := newSubscriber()
	registry.add(endpointID, feed)
	return feed
}

// Fan-out is what makes the feed live, so what matters is that a capture
// reaches every socket watching its own endpoint and no others. Outboxes are
// read directly rather than through a socket: the payload a watching page
// receives is the subject here, not the transport under it.
func TestBroadcastCapture(t *testing.T) {
	captured := database.Request{
		UUID:       "capture-uuid",
		EndpointID: "watched",
		Method:     http.MethodPost,
		Body:       `{"hello":"world"}`,
	}

	t.Run("every socket watching the endpoint receives the capture", func(t *testing.T) {
		registry := newSocketRegistry()
		first, second := watching(registry, "watched"), watching(registry, "watched")

		broadcastCapture(t.Context(), registry, captured)

		for _, feed := range []*subscriber{first, second} {
			require.Len(t, feed.outbox, 1)
			var payload database.Request
			require.NoError(t, json.Unmarshal(<-feed.outbox, &payload),
				"a page parses the pushed payload as a capture")
			assert.Equal(t, captured.UUID, payload.UUID)
			assert.Equal(t, captured.Body, payload.Body)
		}
	})

	// Fan-out is the one place a capture is re-serialised after it was stored,
	// so a row whose headers are not valid JSON fails here and nowhere else.
	// Nothing above this reports it, and half a payload would reach the page as
	// a capture it cannot parse, so the dispatch is dropped and logged.
	t.Run("a capture that will not marshal is logged rather than dispatched", func(t *testing.T) {
		logs := captureLogs(t)
		registry := newSocketRegistry()
		feed := watching(registry, "watched")

		broadcastCapture(t.Context(), registry, database.Request{
			UUID:       "capture-uuid",
			EndpointID: "watched",
			Headers:    datatypes.JSON("not json"),
		})

		assert.Empty(t, feed.outbox)
		assert.Contains(t, logs.String(), "websocket payload marshal failed")
	})

	// The endpoint ID is the only isolation between two users of one instance,
	// so a page open on another endpoint must never be pushed this traffic.
	t.Run("a socket watching another endpoint is left alone", func(t *testing.T) {
		registry := newSocketRegistry()
		feed := watching(registry, "other")

		broadcastCapture(t.Context(), registry, captured)

		assert.Empty(t, feed.outbox)
	})

	// A socket whose client stopped reading must not hold every later capture
	// in memory, and fan-out must not wait on it.
	t.Run("a socket that stops keeping up is closed rather than queued for", func(t *testing.T) {
		registry := newSocketRegistry()
		feed := watching(registry, "watched")

		for range outboxSize + 1 {
			broadcastCapture(t.Context(), registry, captured)
		}

		assert.Len(t, feed.outbox, outboxSize)
		assert.True(t, isClosed(feed), "the overflowing socket is told to stop")
	})
}

// isClosed reports whether a subscriber has been told to stop.
func isClosed(feed *subscriber) bool {
	select {
	case <-feed.closed:
		return true
	default:
		return false
	}
}
