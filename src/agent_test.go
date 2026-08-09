package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgentPrompt(t *testing.T) {
	const (
		endpointURL = "https://example.com/to/purple-frog-0691"
		apiURL      = "https://example.com/api/endpoints/purple-frog-0691/requests"
	)
	prompt := agentPrompt(endpointURL, apiURL, 150, 4*time.Hour)

	t.Run("carries both of the endpoint's URLs", func(t *testing.T) {
		assert.Contains(t, prompt, endpointURL)
		assert.Contains(t, prompt, apiURL)
	})

	// The prompt is read by something that will act on it, so the loop has to
	// be stated rather than implied.
	t.Run("states the cursor loop", func(t *testing.T) {
		assert.Contains(t, prompt, "?since=")
		assert.Contains(t, prompt, "cursor")
		assert.Contains(t, prompt, "hasMore")
	})

	// A prompt that restated these in prose would keep quoting the old number
	// after the constant behind it moved.
	t.Run("quotes the limits it was given", func(t *testing.T) {
		assert.Contains(t, prompt, "150 requests per minute")
		assert.Contains(t, prompt, "deleted after 4 hours")
	})

	t.Run("tracks the figures it is given rather than restating fixed ones", func(t *testing.T) {
		other := agentPrompt(endpointURL, apiURL, 60, 30*time.Minute)

		assert.Contains(t, other, "60 requests per minute")
		assert.Contains(t, other, "deleted after 30 minutes")
		assert.NotContains(t, other, "150")
		assert.NotContains(t, other, "4 hours")
	})

	// A self-hosted deployment has to produce a prompt pointing at itself.
	t.Run("names no hardcoded host", func(t *testing.T) {
		assert.NotContains(t, prompt, "httphq.com")
	})

	// The whole block is copied verbatim into another tool, so stray edge
	// whitespace travels with it.
	t.Run("has no leading or trailing whitespace", func(t *testing.T) {
		assert.Equal(t, strings.TrimSpace(prompt), prompt)
	})
}

func TestRetentionPhrase(t *testing.T) {
	for _, testCase := range []struct {
		in   time.Duration
		want string
	}{
		{4 * time.Hour, "4 hours"},
		{time.Hour, "1 hour"},
		{90 * time.Minute, "90 minutes"},
		{30 * time.Minute, "30 minutes"},
		{time.Minute, "1 minute"},
	} {
		t.Run(testCase.want, func(t *testing.T) {
			assert.Equal(t, testCase.want, retentionPhrase(testCase.in))
		})
	}
}
