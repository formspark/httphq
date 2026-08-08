package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
