package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// An inbound request ID is echoed and stamped on every log line, so its shape
// bounds what an untrusted caller can put in the logs.
func TestRequestIDPattern(t *testing.T) {
	t.Run("accepts the shapes a caller legitimately sends", func(t *testing.T) {
		accepted := []string{
			"70c33dcb-ee76-447d-a82f-26e3a6581010",
			"abcd1234",
			"TRACE-0001",
			strings.Repeat("a", 64),
		}

		for _, id := range accepted {
			assert.Truef(t, requestIDPattern.MatchString(id), "%q should be reused", id)
		}
	})

	t.Run("rejects anything oversized, empty or able to shape a log line", func(t *testing.T) {
		rejected := []string{
			"",
			"short",
			strings.Repeat("a", 65),
			"has space",
			"new\nline",
			`{"level":"error"}`,
			"under_score",
			"semi;colon",
		}

		for _, id := range rejected {
			assert.Falsef(t, requestIDPattern.MatchString(id), "%q should be replaced with a minted ID", id)
		}
	})
}

func TestProbePaths(t *testing.T) {
	// Probe traffic is constant and says nothing, so it logs at debug and stays
	// out of production logs. Everything else has to remain visible.
	t.Run("only probe traffic is demoted", func(t *testing.T) {
		_, isProbe := probePaths["/api/health"]
		assert.True(t, isProbe)

		for _, path := range []string{"/", "/contact", "/api/debug", "/to/purple-frog-0691"} {
			_, demoted := probePaths[path]
			assert.Falsef(t, demoted, "%s carries real traffic and must be logged at info", path)
		}
	})
}
